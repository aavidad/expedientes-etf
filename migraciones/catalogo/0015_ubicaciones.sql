DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador' OR to_regclass('catalogo.version_textos') IS NULL OR to_regclass('catalogo.ubicacion_vigencia') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
SET LOCAL lock_timeout='3s';
SET LOCAL statement_timeout='30s';
SET LOCAL search_path=pg_catalog,catalogo;

-- Instantáneas de estado: las identidades originales siguen siendo inmutables.
CREATE TABLE catalogo.ubicacion_vigencia (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 tipo text NOT NULL CHECK(tipo IN ('equipo','centro')),
 id uuid NOT NULL,
 activo_desde date NOT NULL CHECK(isfinite(activo_desde)),
 activo_hasta date CHECK(isfinite(activo_hasta) AND activo_hasta>activo_desde),
 PRIMARY KEY(version_id,tipo,id)
);
CREATE TABLE catalogo.municipio_ambito (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 municipio_id uuid NOT NULL REFERENCES catalogo.municipio(id),
 ambito_diputacion boolean NOT NULL,
 PRIMARY KEY(version_id,municipio_id)
);
CREATE TRIGGER proteger_vigencia BEFORE INSERT OR UPDATE OR DELETE ON catalogo.ubicacion_vigencia FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('asignacion_municipios');
CREATE TRIGGER proteger_ambito BEFORE INSERT OR UPDATE OR DELETE ON catalogo.municipio_ambito FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('asignacion_municipios');
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['ubicacion_vigencia','municipio_ambito'] LOOP
  EXECUTE format('ALTER TABLE catalogo.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE catalogo.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario ON catalogo.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
  EXECUTE format('CREATE POLICY lectura ON catalogo.%I FOR SELECT TO etf_app USING(nullif(current_setting(''app.usuario_id'',true),'''') IS NOT NULL AND string_to_array(current_setting(''app.perfiles'',true),'','') && ARRAY[''administracion'',''coordinacion'',''profesional_etf'',''sistema''] AND EXISTS(SELECT 1 FROM catalogo.version v WHERE v.id=version_id))',tabla);
  EXECUTE format('REVOKE ALL ON catalogo.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON catalogo.%I TO etf_app,etf_copia',tabla);
  EXECUTE format('CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.%I FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio()',tabla);
 END LOOP;
END $$;

CREATE FUNCTION catalogo.editar_ubicacion(p_esperada integer,p_desde date,p_cambio jsonb) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,catalogo AS $$
DECLARE anterior catalogo.version; nueva uuid; actor uuid; identificador uuid; clase text; codigo_entrada text; desde date; hasta date;
BEGIN
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501'; END IF;
 clase:=p_cambio->>'tipo'; codigo_entrada:=p_cambio->>'codigo';
 IF clase IS NULL OR clase NOT IN ('equipo','centro','municipio') OR p_esperada IS NULL OR p_esperada<0 OR p_desde IS NULL OR NOT isfinite(p_desde) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 PERFORM pg_advisory_xact_lock(697012,hashtext('asignacion_municipios'));
 SELECT * INTO anterior FROM catalogo.version WHERE ambito='asignacion_municipios' AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 IF coalesce(anterior.numero,0)<>p_esperada THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=coalesce(anterior.numero,0)::text; END IF;
 IF anterior.id IS NOT NULL AND p_desde<=anterior.vigente_desde THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
 identificador:=nullif(p_cambio->>'id','')::uuid;
 IF clase='municipio' THEN
  IF NOT EXISTS(SELECT 1 FROM catalogo.municipio WHERE id=identificador AND ine=codigo_entrada) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 ELSE
  desde:=substring(p_cambio->>'activo_desde',1,10)::date;
  hasta:=substring(p_cambio->>'activo_hasta',1,10)::date;
  IF desde IS NULL OR NOT isfinite(desde) OR hasta IS NOT NULL AND (NOT isfinite(hasta) OR hasta<=desde) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
  IF identificador IS NULL THEN
   IF desde<p_desde THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
   IF clase='equipo' THEN
    INSERT INTO catalogo.equipo(codigo,clave_texto,activo_desde,activo_hasta) VALUES(codigo_entrada,'equipo.'||codigo_entrada,desde,hasta) RETURNING id INTO identificador;
   ELSE
    INSERT INTO catalogo.centro_ss(codigo,clave_texto,activo_desde,activo_hasta) VALUES(codigo_entrada,'centro.'||codigo_entrada,desde,hasta) RETURNING id INTO identificador;
   END IF;
  ELSIF clase='equipo' THEN
   IF NOT EXISTS(SELECT 1 FROM catalogo.equipo WHERE id=identificador AND codigo=codigo_entrada) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
  ELSE
   IF NOT EXISTS(SELECT 1 FROM catalogo.centro_ss WHERE id=identificador AND codigo=codigo_entrada) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
  END IF;
 END IF;
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES('asignacion_municipios',p_esperada+1,p_desde,'ubicaciones.publicacion',actor) RETURNING id INTO nueva;
 INSERT INTO catalogo.municipio_asignacion SELECT nueva,municipio_id,centro_ss_id,equipo_id,zts FROM catalogo.municipio_asignacion WHERE version_id=anterior.id;
 INSERT INTO catalogo.municipio_ambito
 SELECT nueva,m.id,coalesce(a.ambito_diputacion,m.ambito_diputacion) FROM catalogo.municipio m LEFT JOIN catalogo.municipio_ambito a ON a.municipio_id=m.id AND a.version_id=anterior.id;
 INSERT INTO catalogo.ubicacion_vigencia
 SELECT nueva,'equipo',e.id,coalesce(v.activo_desde,e.activo_desde),CASE WHEN v.id IS NOT NULL THEN v.activo_hasta ELSE e.activo_hasta END FROM catalogo.equipo e LEFT JOIN catalogo.ubicacion_vigencia v ON v.id=e.id AND v.tipo='equipo' AND v.version_id=anterior.id;
 INSERT INTO catalogo.ubicacion_vigencia
 SELECT nueva,'centro',e.id,coalesce(v.activo_desde,e.activo_desde),CASE WHEN v.id IS NOT NULL THEN v.activo_hasta ELSE e.activo_hasta END FROM catalogo.centro_ss e LEFT JOIN catalogo.ubicacion_vigencia v ON v.id=e.id AND v.tipo='centro' AND v.version_id=anterior.id;
 IF clase='municipio' THEN
  UPDATE catalogo.municipio_asignacion SET centro_ss_id=(p_cambio->>'centro_ss_id')::uuid,equipo_id=(p_cambio->>'equipo_id')::uuid WHERE version_id=nueva AND municipio_id=identificador;
  IF NOT FOUND THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
  UPDATE catalogo.municipio_ambito SET ambito_diputacion=(p_cambio->>'ambito_diputacion')::boolean WHERE version_id=nueva AND municipio_id=identificador;
 ELSE
  UPDATE catalogo.ubicacion_vigencia SET activo_desde=desde,activo_hasta=hasta WHERE version_id=nueva AND tipo=clase AND id=identificador;
 END IF;
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM paquete.referencia_activa WHERE tipo='catalogo';
 IF (SELECT count(*) FROM catalogo.version_textos WHERE version_id=nueva)<>2 THEN RAISE EXCEPTION 'paquete.publicacion_invalida'; END IF;
 PERFORM catalogo.publicar(nueva,actor);
 RETURN nueva;
END $$;
REVOKE ALL ON FUNCTION catalogo.editar_ubicacion(integer,date,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalogo.editar_ubicacion(integer,date,jsonb) TO etf_app;
