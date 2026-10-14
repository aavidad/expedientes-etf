DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador' OR to_regclass('catalogo.lista_opcion') IS NULL OR to_regclass('paquete.referencia_activa') IS NULL OR to_regclass('catalogo.parametro') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
SET LOCAL lock_timeout='3s';
SET LOCAL statement_timeout='30s';
SET LOCAL search_path=pg_catalog,catalogo;
CREATE TABLE catalogo.parametro (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 clave text NOT NULL CHECK(clave ~ '^[a-z][a-z0-9_]{0,63}$'),
 valor_entero integer NOT NULL CHECK(valor_entero>=0),
 unidad text NOT NULL CHECK(unidad ~ '^[a-z][a-z0-9_]{0,63}$'), PRIMARY KEY(version_id,clave)
);
CREATE TABLE catalogo.tramo (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 tipo text NOT NULL CHECK(tipo ~ '^[a-z][a-z0-9_]{0,63}$'), orden integer NOT NULL CHECK(orden>=0),
 desde numeric NOT NULL CHECK(desde>=0 AND desde<>'NaN'::numeric),hasta numeric CHECK(hasta>desde AND hasta<>'NaN'::numeric),
 unidad text NOT NULL CHECK(unidad ~ '^[a-z][a-z0-9_]{0,63}$'),PRIMARY KEY(version_id,tipo,orden),
 EXCLUDE USING gist(version_id WITH =,tipo WITH =,numrange(desde,hasta,'[)') WITH &&)
);
CREATE TABLE catalogo.version_textos (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),codigo text NOT NULL,version_contenido text NOT NULL,
 PRIMARY KEY(version_id,codigo),FOREIGN KEY(codigo,version_contenido) REFERENCES paquete.idioma_instalado(codigo,version)
);
CREATE TABLE catalogo.firmante (
 id uuid NOT NULL,version integer NOT NULL CHECK(version>0),
 tipo text NOT NULL CHECK(tipo IN ('firmante','destinatario')),rol text NOT NULL CHECK(rol ~ '^[a-z][a-z0-9_]{0,63}$'),
 nombre text NOT NULL CHECK(length(btrim(nombre)) BETWEEN 1 AND 200),
 cargo_codigo text NOT NULL CHECK(cargo_codigo ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 organismo_codigo text NOT NULL CHECK(organismo_codigo ~ '^[A-Z][A-Z0-9_]{0,63}$'),direccion text NOT NULL CHECK(length(direccion)<=500),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),vigente_hasta date CHECK(isfinite(vigente_hasta) AND vigente_hasta>vigente_desde),
 creada_por uuid NOT NULL,creada_en timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(id,version),
 EXCLUDE USING gist(tipo WITH =,rol WITH =,daterange(vigente_desde,vigente_hasta,'[)') WITH &&) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE catalogo.firmante_textos (
 id uuid NOT NULL,version integer NOT NULL,codigo text NOT NULL,version_contenido text NOT NULL,
 PRIMARY KEY(id,version,codigo),FOREIGN KEY(id,version) REFERENCES catalogo.firmante(id,version),
 FOREIGN KEY(codigo,version_contenido) REFERENCES paquete.idioma_instalado(codigo,version)
);
CREATE TRIGGER proteger_parametro BEFORE INSERT OR UPDATE OR DELETE ON catalogo.parametro FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('plazos');
CREATE TRIGGER proteger_tramo BEFORE INSERT OR UPDATE OR DELETE ON catalogo.tramo FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('tramos');
CREATE TRIGGER textos_inmutables BEFORE UPDATE OR DELETE ON catalogo.version_textos FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();
CREATE TRIGGER firmante_textos_inmutables BEFORE UPDATE OR DELETE ON catalogo.firmante_textos FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();
CREATE FUNCTION catalogo.proteger_firmante() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,catalogo AS $$
BEGIN
 IF TG_OP='UPDATE' AND pg_trigger_depth()=2 AND current_user='etf_propietario'
 AND (to_jsonb(NEW)-'vigente_hasta')=(to_jsonb(OLD)-'vigente_hasta')
 AND NEW.vigente_hasta IS NOT NULL AND (OLD.vigente_hasta IS NULL OR NEW.vigente_hasta<OLD.vigente_hasta) THEN RETURN NEW; END IF;
 RAISE EXCEPTION 'catalogo.version_inmutable';
END $$;
CREATE TRIGGER proteger_firmante BEFORE UPDATE OR DELETE ON catalogo.firmante FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_firmante();
CREATE FUNCTION catalogo.cerrar_firmante() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,catalogo AS $$
BEGIN
 IF NEW.version>1 THEN
  UPDATE catalogo.firmante SET vigente_hasta=NEW.vigente_desde WHERE id=NEW.id AND version=NEW.version-1 AND vigente_desde<NEW.vigente_desde AND (vigente_hasta IS NULL OR vigente_hasta>NEW.vigente_desde);
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER cerrar_firmante BEFORE INSERT ON catalogo.firmante FOR EACH ROW EXECUTE FUNCTION catalogo.cerrar_firmante();
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['parametro','tramo','version_textos','firmante','firmante_textos'] LOOP
  EXECUTE format('ALTER TABLE catalogo.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE catalogo.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario ON catalogo.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
  EXECUTE format('CREATE POLICY lectura ON catalogo.%I FOR SELECT TO etf_app USING(nullif(current_setting(''app.usuario_id'',true),'''') IS NOT NULL AND string_to_array(current_setting(''app.perfiles'',true),'','') && ARRAY[''administracion'',''coordinacion'',''equipo'',''sistema''])',tabla);
  EXECUTE format('REVOKE ALL ON catalogo.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON catalogo.%I TO etf_app,etf_copia',tabla);
  EXECUTE format('CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.%I FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio()',tabla);
 END LOOP;
END $$;
CREATE FUNCTION catalogo.editar(p_ambito text,p_esperada integer,p_desde date,p_cambio jsonb) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,catalogo AS $$
DECLARE anterior catalogo.version; nueva uuid; actor uuid;
BEGIN
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501'; END IF;
 IF p_ambito NOT IN ('listas','plazos','tramos') OR p_esperada<0 THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 PERFORM pg_advisory_xact_lock(697012,hashtext(p_ambito));
 SELECT * INTO anterior FROM catalogo.version WHERE ambito=p_ambito AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 IF coalesce(anterior.numero,0)<>p_esperada THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=coalesce(anterior.numero,0)::text; END IF;
 IF anterior.id IS NOT NULL AND p_desde<=anterior.vigente_desde THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES(p_ambito,p_esperada+1,p_desde,'catalogo.edicion',actor) RETURNING id INTO nueva;
 IF p_ambito='listas' THEN
  INSERT INTO catalogo.lista_opcion SELECT nueva,lista,codigo,orden,admite_texto,activa FROM catalogo.lista_opcion WHERE version_id=anterior.id;
  INSERT INTO catalogo.lista_opcion VALUES(nueva,p_cambio->>'lista',p_cambio->>'codigo',(p_cambio->>'orden')::integer,(p_cambio->>'admite_texto')::boolean,(p_cambio->>'activa')::boolean)
   ON CONFLICT(version_id,lista,codigo) DO UPDATE SET orden=EXCLUDED.orden,admite_texto=EXCLUDED.admite_texto,activa=EXCLUDED.activa;
 ELSIF p_ambito='plazos' THEN
  INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
  INSERT INTO catalogo.parametro VALUES(nueva,p_cambio->>'clave',(p_cambio->>'valor')::integer,p_cambio->>'unidad') ON CONFLICT(version_id,clave) DO UPDATE SET valor_entero=EXCLUDED.valor_entero,unidad=EXCLUDED.unidad;
 ELSE
  INSERT INTO catalogo.tramo SELECT nueva,tipo,orden,desde,hasta,unidad FROM catalogo.tramo WHERE version_id=anterior.id;
  INSERT INTO catalogo.tramo VALUES(nueva,p_cambio->>'tipo',(p_cambio->>'orden')::integer,(p_cambio->>'desde')::numeric,(p_cambio->>'hasta')::numeric,p_cambio->>'unidad') ON CONFLICT(version_id,tipo,orden) DO UPDATE SET desde=EXCLUDED.desde,hasta=EXCLUDED.hasta,unidad=EXCLUDED.unidad;
 END IF;
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM paquete.referencia_activa WHERE tipo='catalogo';
 IF p_ambito='listas' AND (SELECT count(*) FROM catalogo.version_textos WHERE version_id=nueva)<>2 THEN RAISE EXCEPTION 'paquete.publicacion_invalida'; END IF;
 PERFORM catalogo.publicar(nueva,actor);
 RETURN nueva;
END $$;
CREATE FUNCTION catalogo.guardar_firmante(p_esperada integer,p_desde date,p_cambio jsonb) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,catalogo AS $$
DECLARE anterior catalogo.firmante; identificador uuid; actor uuid;
BEGIN
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501'; END IF;
 identificador:=coalesce(nullif(p_cambio->>'id','')::uuid,gen_random_uuid());
 PERFORM pg_advisory_xact_lock(697015,hashtext((p_cambio->>'tipo')||':'||(p_cambio->>'rol')));
 SELECT * INTO anterior FROM catalogo.firmante WHERE tipo=p_cambio->>'tipo' AND rol=p_cambio->>'rol' AND id<>identificador ORDER BY version DESC LIMIT 1;
 IF anterior.id IS NOT NULL THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=anterior.version::text; END IF;
 SELECT * INTO anterior FROM catalogo.firmante WHERE id=identificador ORDER BY version DESC LIMIT 1;
 IF coalesce(anterior.version,0)<>p_esperada THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=coalesce(anterior.version,0)::text; END IF;
 IF anterior.id IS NOT NULL AND (p_desde<=anterior.vigente_desde OR p_cambio->>'rol'<>anterior.rol OR p_cambio->>'tipo'<>anterior.tipo) THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
 INSERT INTO catalogo.firmante(id,version,tipo,rol,nombre,cargo_codigo,organismo_codigo,direccion,vigente_desde,vigente_hasta,creada_por)
 VALUES(identificador,p_esperada+1,p_cambio->>'tipo',p_cambio->>'rol',p_cambio->>'nombre',p_cambio->>'cargo_codigo',p_cambio->>'organismo_codigo',p_cambio->>'direccion',p_desde,substring(p_cambio->>'vigente_hasta',1,10)::date,actor);
 INSERT INTO catalogo.firmante_textos SELECT identificador,p_esperada+1,codigo,version_contenido FROM paquete.referencia_activa WHERE tipo='catalogo';
 IF (SELECT count(*) FROM catalogo.firmante_textos WHERE id=identificador AND version=p_esperada+1)<>2 THEN RAISE EXCEPTION 'paquete.publicacion_invalida'; END IF;
 RETURN identificador;
END $$;
REVOKE ALL ON FUNCTION catalogo.proteger_firmante(),catalogo.cerrar_firmante(),catalogo.editar(text,integer,date,jsonb),catalogo.guardar_firmante(integer,date,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalogo.editar(text,integer,date,jsonb),catalogo.guardar_firmante(integer,date,jsonb) TO etf_app;
