SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('identidad.usuario_equipo') IS NULL OR to_regclass('identidad.revision_restauracion') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- La referencia procede del procedimiento de sistemas, fuera de la copia que
-- se restaura. La aplicación no puede iniciarlo ni alterar sus propuestas.
CREATE TABLE identidad.revision_restauracion (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), copia_en timestamptz NOT NULL CHECK (isfinite(copia_en)),
 iniciada_en timestamptz NOT NULL DEFAULT clock_timestamp(), referencia jsonb NOT NULL,
 anulada_en timestamptz, confirmada_en timestamptz, confirmada_por uuid REFERENCES identidad.usuario(id),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0)
);
CREATE UNIQUE INDEX revision_restauracion_abierta ON identidad.revision_restauracion ((true)) WHERE confirmada_en IS NULL AND anulada_en IS NULL;
CREATE TABLE identidad.cambio_restauracion (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), revision_id uuid NOT NULL REFERENCES identidad.revision_restauracion(id),
 guid uuid NOT NULL, anterior jsonb, propuesto jsonb, decision text CHECK (decision IN ('aplicar','descartar')),
 decidido_por uuid REFERENCES identidad.usuario(id), decidido_en timestamptz,
 UNIQUE(revision_id,guid)
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['revision_restauracion','cambio_restauracion'] LOOP
  EXECUTE format('ALTER TABLE identidad.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE identidad.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario ON identidad.%I TO etf_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('CREATE POLICY administracion ON identidad.%I FOR SELECT TO etf_app USING (identidad.es_administrador())',tabla);
  EXECUTE format('REVOKE ALL ON identidad.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON identidad.%I TO etf_app,etf_copia',tabla);
 END LOOP;
END $$;

CREATE FUNCTION identidad.exportar_identidad() RETURNS jsonb
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
 SELECT coalesce(jsonb_agg(jsonb_build_object('guid',u.object_guid_ad,'cuenta',coalesce(u.cuenta_ad,u.cuenta_ad_anterior),
 'nombre',u.nombre_mostrado,'estado',u.estado,
 'perfiles',coalesce((SELECT jsonb_agg(jsonb_build_object('perfil',p.perfil,'desde',p.desde,'hasta',p.hasta) ORDER BY p.perfil)
 FROM identidad.usuario_perfil p WHERE p.usuario_id=u.id AND (p.hasta IS NULL OR p.hasta>greatest(p.desde,statement_timestamp()))),'[]'::jsonb),
 'equipo',(SELECT jsonb_build_object('equipo_id',e.equipo_id,'puesto',e.puesto,'desde',e.desde,'hasta',e.hasta)
 FROM identidad.usuario_equipo e WHERE e.usuario_id=u.id AND (e.hasta IS NULL OR e.hasta>greatest(e.desde,statement_timestamp())))) ORDER BY u.object_guid_ad),'[]'::jsonb)
 FROM identidad.usuario u
$$;
REVOKE EXECUTE ON FUNCTION identidad.exportar_identidad() FROM PUBLIC,etf_app,etf_copia;

CREATE FUNCTION identidad.iniciar_revision_restauracion(fecha timestamptz, vigente jsonb) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE revision uuid; restaurada jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
 IF fecha IS NULL OR NOT isfinite(fecha) OR fecha>clock_timestamp() OR jsonb_typeof(vigente) IS DISTINCT FROM 'array'
 OR jsonb_array_length(vigente)=0 OR EXISTS(SELECT FROM jsonb_array_elements(vigente) v
 WHERE v->>'guid' IS NULL OR v->>'estado' NOT IN ('activo','baja','pendiente_alta') OR jsonb_typeof(v->'perfiles') IS DISTINCT FROM 'array')
 OR (SELECT count(*)<>count(DISTINCT (v->>'guid')::uuid) FROM jsonb_array_elements(vigente) v) THEN
  RAISE EXCEPTION 'restauracion.referencia_invalida';
 END IF;
 restaurada := identidad.exportar_identidad();
 -- Exigir al menos una persona administradora vigente presente en la copia.
 IF NOT EXISTS(SELECT FROM jsonb_array_elements(vigente) v JOIN jsonb_array_elements(restaurada) r ON v->>'guid'=r->>'guid'
 WHERE v->>'estado'='activo' AND r->>'estado'='activo'
 AND EXISTS(SELECT FROM jsonb_array_elements(r->'perfiles') p WHERE p->>'perfil'='administracion' AND (p->>'desde')::timestamptz<=clock_timestamp() AND (p->>'hasta' IS NULL OR (p->>'hasta')::timestamptz>clock_timestamp()))
 AND EXISTS(SELECT FROM jsonb_array_elements(v->'perfiles') p WHERE p->>'perfil'='administracion' AND (p->>'desde')::timestamptz<=clock_timestamp() AND (p->>'hasta' IS NULL OR (p->>'hasta')::timestamptz>clock_timestamp()))
 ) THEN RAISE EXCEPTION 'restauracion.sin_administracion'; END IF;
 UPDATE identidad.revision_restauracion SET anulada_en=clock_timestamp(),version=version+1 WHERE confirmada_en IS NULL AND anulada_en IS NULL;
 INSERT INTO identidad.revision_restauracion(copia_en,referencia) VALUES(fecha,vigente) RETURNING id INTO revision;
 INSERT INTO identidad.cambio_restauracion(revision_id,guid,anterior,propuesto)
 SELECT revision,coalesce((r->>'guid')::uuid,(v->>'guid')::uuid),r,v
 FROM jsonb_array_elements(restaurada) r FULL JOIN jsonb_array_elements(vigente) v ON r->>'guid'=v->>'guid'
 WHERE (r-'nombre'-'cuenta') IS DISTINCT FROM (v-'nombre'-'cuenta');
 DELETE FROM identidad.sesion;
 PERFORM set_config('app.usuario_id','',true);
 PERFORM set_config('app.perfiles','',true);
 PERFORM auditoria.registrar(clock_timestamp(),'identidad.restauracion_iniciar','identidad.revision_restauracion',revision,NULL,'ok');
 RETURN revision;
END $$;
REVOKE EXECUTE ON FUNCTION identidad.iniciar_revision_restauracion(timestamptz,jsonb) FROM PUBLIC,etf_app,etf_copia;

-- Se comprueba en cada lectura de usuario. Ninguna sesión anterior ni un perfil
-- administrativo revocado después de la copia puede sortear la revisión.
CREATE FUNCTION identidad.comprobar_revision_restauracion(persona uuid) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE referencia jsonb; guid uuid;
BEGIN
 SELECT r.referencia INTO referencia FROM identidad.revision_restauracion r WHERE confirmada_en IS NULL AND anulada_en IS NULL;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT object_guid_ad INTO guid FROM identidad.usuario WHERE id=persona AND estado='activo';
 IF NOT EXISTS(SELECT FROM jsonb_array_elements(referencia) v CROSS JOIN LATERAL jsonb_array_elements(v->'perfiles') p
 WHERE (v->>'guid')::uuid=guid AND v->>'estado'='activo' AND p->>'perfil'='administracion'
 AND (p->>'desde')::timestamptz<=clock_timestamp() AND (p->>'hasta' IS NULL OR (p->>'hasta')::timestamptz>clock_timestamp()))
 OR NOT EXISTS(SELECT FROM identidad.usuario_perfil WHERE usuario_id=persona AND perfil='administracion' AND desde<=clock_timestamp() AND (hasta IS NULL OR hasta>clock_timestamp()))
 THEN RAISE EXCEPTION 'restauracion.acceso_bloqueado'; END IF;
 RETURN true;
END $$;
REVOKE EXECUTE ON FUNCTION identidad.comprobar_revision_restauracion(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.comprobar_revision_restauracion(uuid) TO etf_app;

CREATE FUNCTION identidad.resolver_cambio_restauracion(revision uuid, cambio uuid, esperada bigint, eleccion text) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE c identidad.cambio_restauracion; persona uuid; actual bigint; actor uuid; perfiles jsonb; equipo jsonb; ahora timestamptz;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF NOT identidad.es_administrador() OR NOT identidad.comprobar_revision_restauracion(actor) THEN RAISE EXCEPTION 'identidad.acceso_denegado'; END IF;
 SELECT version INTO actual FROM identidad.revision_restauracion WHERE id=revision AND confirmada_en IS NULL AND anulada_en IS NULL FOR UPDATE;
 IF actual IS DISTINCT FROM esperada THEN RAISE EXCEPTION 'identidad.conflicto'; END IF;
 SELECT * INTO c FROM identidad.cambio_restauracion WHERE id=cambio AND revision_id=revision FOR UPDATE;
 IF NOT FOUND OR c.decision IS NOT NULL OR eleccion NOT IN ('aplicar','descartar') OR eleccion IS NULL THEN RAISE EXCEPTION 'identidad.conflicto'; END IF;
 IF eleccion='aplicar' THEN
  ahora:=clock_timestamp();
  SELECT id,version INTO persona,actual FROM identidad.usuario WHERE object_guid_ad=c.guid;
  IF persona=actor THEN RAISE EXCEPTION 'restauracion.revision_propia'; END IF;
  IF persona IS NULL THEN
   UPDATE identidad.usuario SET cuenta_ad_anterior=cuenta_ad,cuenta_ad=NULL WHERE cuenta_ad=c.propuesto->>'cuenta' AND object_guid_ad<>c.guid;
   INSERT INTO identidad.usuario(object_guid_ad,cuenta_ad,nombre_mostrado,primer_acceso_en,ultimo_acceso_en)
   VALUES(c.guid,c.propuesto->>'cuenta',c.propuesto->>'nombre',ahora,ahora) RETURNING id,version INTO persona,actual;
  END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('perfil',p->>'perfil','desde',greatest((p->>'desde')::timestamptz,ahora),'hasta',p->'hasta')),'[]'::jsonb)
  INTO perfiles FROM jsonb_array_elements(coalesce(c.propuesto->'perfiles','[]'::jsonb)) p
  WHERE p->>'hasta' IS NULL OR (p->>'hasta')::timestamptz>ahora;
  equipo:=nullif(c.propuesto->'equipo','null'::jsonb);
  IF equipo IS NOT NULL THEN
   equipo:=jsonb_set(equipo,'{desde}',to_jsonb(greatest((equipo->>'desde')::timestamptz,ahora)));
   IF equipo->>'hasta' IS NOT NULL AND (equipo->>'hasta')::timestamptz<=ahora THEN equipo:=NULL; END IF;
  END IF;
  IF c.propuesto IS NULL OR c.propuesto->>'estado'='baja' THEN perfiles:='[]'::jsonb;equipo:=NULL; END IF;
  BEGIN
  IF identidad.cambiar_perfiles(persona,actual,CASE WHEN c.propuesto IS NULL OR c.propuesto->>'estado'='baja' THEN 'baja' ELSE 'activo' END,perfiles,equipo) IS DISTINCT FROM true THEN RAISE EXCEPTION 'restauracion.revision_propia'; END IF;
  EXCEPTION WHEN foreign_key_violation THEN RAISE EXCEPTION 'restauracion.equipo_no_disponible'; END;
  IF c.propuesto->>'estado'='pendiente_alta' THEN UPDATE identidad.usuario SET estado='pendiente_alta' WHERE id=persona; END IF;
  DELETE FROM identidad.sesion WHERE usuario_id=persona AND usuario_id<>actor;
 END IF;
 UPDATE identidad.cambio_restauracion SET decision=eleccion,decidido_por=actor,decidido_en=clock_timestamp() WHERE id=cambio;
 UPDATE identidad.revision_restauracion SET version=version+1 WHERE id=revision;
 PERFORM auditoria.registrar(clock_timestamp(),'identidad.restauracion_'||eleccion,'identidad.cambio_restauracion',cambio,NULL,'ok');
END $$;
REVOKE EXECUTE ON FUNCTION identidad.resolver_cambio_restauracion(uuid,uuid,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.resolver_cambio_restauracion(uuid,uuid,bigint,text) TO etf_app;

CREATE FUNCTION identidad.confirmar_revision_restauracion(revision uuid, esperada bigint) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE actual bigint; actor uuid;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF NOT identidad.es_administrador() OR NOT identidad.comprobar_revision_restauracion(actor) THEN RAISE EXCEPTION 'identidad.acceso_denegado'; END IF;
 SELECT version INTO actual FROM identidad.revision_restauracion WHERE id=revision AND confirmada_en IS NULL AND anulada_en IS NULL FOR UPDATE;
 IF actual IS DISTINCT FROM esperada THEN RAISE EXCEPTION 'identidad.conflicto'; END IF;
 IF EXISTS(SELECT FROM identidad.cambio_restauracion WHERE revision_id=revision AND decision IS NULL) THEN RAISE EXCEPTION 'restauracion.cambios_pendientes'; END IF;
 UPDATE identidad.revision_restauracion SET confirmada_en=clock_timestamp(),confirmada_por=actor,version=version+1 WHERE id=revision;
 DELETE FROM identidad.sesion WHERE usuario_id<>actor;
 PERFORM auditoria.registrar(clock_timestamp(),'identidad.restauracion_confirmar','identidad.revision_restauracion',revision,NULL,'ok');
END $$;
REVOKE EXECUTE ON FUNCTION identidad.confirmar_revision_restauracion(uuid,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.confirmar_revision_restauracion(uuid,bigint) TO etf_app;

-- Durante la revisión, tampoco una consulta SQL con un perfil restaurado
-- permite leer las propuestas de una persona administradora ya revocada.
DROP POLICY administracion ON identidad.revision_restauracion;
DROP POLICY administracion ON identidad.cambio_restauracion;
CREATE POLICY administracion ON identidad.revision_restauracion FOR SELECT TO etf_app USING (identidad.es_administrador() AND identidad.comprobar_revision_restauracion(nullif(current_setting('app.usuario_id',true),'')::uuid));
CREATE POLICY administracion ON identidad.cambio_restauracion FOR SELECT TO etf_app USING (identidad.es_administrador() AND identidad.comprobar_revision_restauracion(nullif(current_setting('app.usuario_id',true),'')::uuid));
