SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regprocedure('expediente.proteger_expediente_cerrado()') IS NULL
 OR to_regclass('expediente.carga_inicial') IS NULL
 OR to_regclass('expediente.seguimiento') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
ALTER TABLE expediente.expediente DROP CONSTRAINT expediente_origen_check,
 ADD CONSTRAINT expediente_origen_check CHECK(origen IN ('derivacion','carga_inicial','traslado')),
 ADD COLUMN traslado_origen_id uuid REFERENCES expediente.expediente(id),
 ADD COLUMN traslado_equipo_id uuid REFERENCES catalogo.equipo(id),
 ADD COLUMN copia_traslado jsonb,
 ADD COLUMN servicio_derivante text,
 ADD CHECK ((origen<>'traslado' AND traslado_origen_id IS NULL AND traslado_equipo_id IS NULL AND copia_traslado IS NULL AND servicio_derivante IS NULL)
 OR (origen='traslado' AND traslado_origen_id IS NOT NULL AND traslado_origen_id<>id AND traslado_equipo_id IS NOT NULL
 AND copia_traslado IS NOT NULL AND jsonb_typeof(copia_traslado)='object' AND servicio_derivante IS NOT NULL AND servicio_derivante='OTRO_ETF'));
-- Conserva la carga histórica al ampliar la procedencia con un traslado.
ALTER TABLE expediente.expediente DROP CONSTRAINT carga_procedencia,
 ADD CONSTRAINT carga_procedencia CHECK (
 (origen IN ('derivacion','traslado') AND carga_id IS NULL AND equipo_carga_id IS NULL AND cargado_por IS NULL)
 OR (origen='carga_inicial' AND carga_id IS NOT NULL AND equipo_carga_id IS NOT NULL AND cargado_por IS NOT NULL
  AND (equipo_id IS NULL OR equipo_id=equipo_carga_id)));
CREATE TABLE expediente.traslado (
 origen_id uuid PRIMARY KEY, destino_id uuid NOT NULL UNIQUE REFERENCES expediente.expediente(id),
 equipo_origen_id uuid NOT NULL, equipo_destino_id uuid NOT NULL REFERENCES catalogo.equipo(id),
 trasladado_por uuid NOT NULL REFERENCES identidad.usuario(id), trasladado_en timestamptz NOT NULL,
 CHECK(origen_id<>destino_id AND equipo_origen_id<>equipo_destino_id),
 FOREIGN KEY(origen_id,equipo_origen_id) REFERENCES expediente.expediente(id,equipo_id)
);
ALTER TABLE expediente.traslado ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.traslado FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.traslado FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT,INSERT ON expediente.traslado TO etf_app;
GRANT SELECT ON expediente.traslado TO etf_copia;
CREATE POLICY lectura ON expediente.traslado FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR
 ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND (equipo_origen_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) OR equipo_destino_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])))));
CREATE POLICY escritura ON expediente.traslado FOR INSERT TO etf_app WITH CHECK (
 trasladado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])));
CREATE FUNCTION expediente.proteger_traslado() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_TABLE_NAME='traslado' THEN
  IF NOT EXISTS(SELECT 1 FROM expediente.expediente e JOIN expediente.episodio p ON p.expediente_id=e.id
   WHERE e.id=NEW.origen_id AND e.estado='cerrada' AND p.motivo_cierre='TRASLADO' AND p.cerrado_por=NEW.trasladado_por
   AND NOT EXISTS(SELECT 1 FROM expediente.episodio hijo WHERE hijo.reapertura_de_id=p.id))
   OR NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.destino_id AND e.origen='traslado'
    AND e.traslado_origen_id=NEW.origen_id AND e.traslado_equipo_id=NEW.equipo_destino_id AND e.estado='derivacion_recibida') THEN
   RAISE EXCEPTION 'expediente.traslado_invalido';
  END IF;
 ELSE
  IF TG_OP='UPDATE' AND (NEW.traslado_origen_id IS DISTINCT FROM OLD.traslado_origen_id OR NEW.traslado_equipo_id IS DISTINCT FROM OLD.traslado_equipo_id
   OR NEW.copia_traslado IS DISTINCT FROM OLD.copia_traslado OR NEW.servicio_derivante IS DISTINCT FROM OLD.servicio_derivante OR NEW.origen IS DISTINCT FROM OLD.origen) THEN
   RAISE EXCEPTION 'expediente.asignacion_inmutable';
  END IF;
  IF NEW.origen='traslado' AND ((NEW.equipo_id IS NOT NULL AND NEW.equipo_id<>NEW.traslado_equipo_id)
   OR (TG_OP='INSERT' AND (NEW.estado<>'derivacion_recibida' OR NOT coalesce('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])),false)))) THEN
   RAISE EXCEPTION 'expediente.traslado_invalido';
  END IF;
 END IF;
 RETURN NEW;
END $$;
-- Coordinación inserta únicamente los menores de la copia inmutable durante
-- su aceptación. No obtiene permisos de edición ordinaria de menores.
CREATE POLICY coordinacion_copia ON expediente.menor FOR INSERT TO etf_app WITH CHECK (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=menor.expediente_id AND e.equipo_id=menor.equipo_id AND e.origen='traslado'
  AND e.estado='aceptada' AND e.modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid
  AND EXISTS(SELECT 1 FROM expediente.episodio p WHERE p.expediente_id=e.id AND p.reapertura_de_id IS NULL AND p.cierre_fecha IS NULL AND p.creado_por=e.modificado_por AND p.creado_en=e.modificado_en)
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(e.copia_traslado->'menores') j WHERE (j->>'id')::uuid=menor.id
   AND (j->>'orden')::integer=menor.orden AND (j-'sexo')=(menor.datos-'sexo'))));
REVOKE EXECUTE ON FUNCTION expediente.proteger_traslado() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_traslado BEFORE INSERT OR UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_traslado();
CREATE TRIGGER proteger_traslado BEFORE INSERT ON expediente.traslado FOR EACH ROW EXECUTE FUNCTION expediente.proteger_traslado();

-- El seguimiento tiene su propio agregado y versión. No reabre ni modifica
-- el episodio de intervención ni la ficha cerrada.
CREATE TABLE expediente.seguimiento (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL UNIQUE, equipo_id uuid NOT NULL, episodio_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0), estado text NOT NULL CHECK(estado IN ('activo','terminado')),
 fecha_cese date NOT NULL CHECK(isfinite(fecha_cese)), tramo_desde date NOT NULL CHECK(isfinite(tramo_desde)), fecha_fin date NOT NULL CHECK(isfinite(fecha_fin) AND fecha_fin>fecha_cese),
 fecha_terminacion date CHECK(isfinite(fecha_terminacion)), motivo text, motivo_reapertura text,
 regla_id uuid NOT NULL REFERENCES catalogo.version(id),
 creado_por uuid NOT NULL REFERENCES identidad.usuario(id), creado_en timestamptz NOT NULL,
 modificado_por uuid REFERENCES identidad.usuario(id), modificado_en timestamptz NOT NULL,
 FOREIGN KEY(episodio_id,expediente_id,equipo_id) REFERENCES expediente.episodio(id,expediente_id,equipo_id),
 CHECK((estado='activo' AND fecha_terminacion IS NULL AND motivo IS NULL) OR
  (estado='terminado' AND fecha_terminacion BETWEEN greatest(fecha_cese,tramo_desde) AND fecha_fin AND length(btrim(motivo)) BETWEEN 1 AND 4000))
);
CREATE INDEX seguimiento_vencimiento ON expediente.seguimiento(fecha_fin,expediente_id) WHERE estado='activo';
CREATE TABLE expediente.seguimiento_historia (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), seguimiento_id uuid NOT NULL REFERENCES expediente.seguimiento(id),
 expediente_id uuid NOT NULL, equipo_id uuid NOT NULL, version integer NOT NULL,
 accion text NOT NULL CHECK(accion IN ('iniciar','terminar','reabrir','vencer')), datos jsonb NOT NULL,
 registrado_por uuid REFERENCES identidad.usuario(id), registrado_en timestamptz NOT NULL,
 UNIQUE(seguimiento_id,version), FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['seguimiento','seguimiento_historia'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_app,etf_copia',tabla);
  EXECUTE format($sql$CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))$sql$,tabla);
  EXECUTE format('CREATE POLICY propietario ON expediente.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
 END LOOP;
END $$;
GRANT INSERT ON expediente.seguimiento TO etf_app;
GRANT UPDATE(version,estado,fecha_fin,regla_id,tramo_desde,fecha_terminacion,motivo,motivo_reapertura,modificado_por,modificado_en) ON expediente.seguimiento TO etf_app;
CREATE POLICY escritura ON expediente.seguimiento TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK(modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
-- Un desencadenante de propietario sólo añade la foto del seguimiento: no
-- permite al rol de aplicación insertar, actualizar ni borrar la historia.
CREATE FUNCTION expediente.historiar_seguimiento() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$ DECLARE accion text; BEGIN
 accion:=CASE WHEN TG_OP='INSERT' THEN 'iniciar' WHEN NEW.estado='activo' THEN 'reabrir' WHEN NEW.modificado_por IS NULL THEN 'vencer' ELSE 'terminar' END;
 INSERT INTO expediente.seguimiento_historia(seguimiento_id,expediente_id,equipo_id,version,accion,datos,registrado_por,registrado_en)
 VALUES(NEW.id,NEW.expediente_id,NEW.equipo_id,NEW.version,accion,to_jsonb(NEW)-ARRAY['creado_por','modificado_por'],NEW.modificado_por,NEW.modificado_en);
 RETURN NULL;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.historiar_seguimiento() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER historiar_seguimiento AFTER INSERT OR UPDATE ON expediente.seguimiento FOR EACH ROW EXECUTE FUNCTION expediente.historiar_seguimiento();
CREATE FUNCTION expediente.proteger_seguimiento() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.version<>1 OR NEW.estado<>'activo' OR NEW.tramo_desde<>NEW.fecha_cese OR NEW.creado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
   OR NEW.modificado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid OR NEW.fecha_cese>(NEW.creado_en AT TIME ZONE 'Europe/Madrid')::date OR NEW.fecha_fin IS DISTINCT FROM catalogo.fin_seguimiento(NEW.fecha_cese,NEW.regla_id,(NEW.creado_en AT TIME ZONE 'Europe/Madrid')::date)
   OR NOT EXISTS(SELECT 1 FROM expediente.expediente e JOIN expediente.episodio p ON p.expediente_id=e.id
     WHERE e.id=NEW.expediente_id AND e.estado='cerrada' AND e.subprograma='REUNIFICACION' AND p.id=NEW.episodio_id AND p.cierre_fecha IS NOT NULL
     AND NOT EXISTS(SELECT 1 FROM expediente.episodio hijo WHERE hijo.reapertura_de_id=p.id)) THEN
   RAISE EXCEPTION 'expediente.seguimiento_invalido';
  END IF;
 ELSE
  IF NEW.version<>OLD.version+1 OR NEW.estado=OLD.estado OR (NEW.estado='terminado' AND (NEW.tramo_desde<>OLD.tramo_desde OR NEW.fecha_fin<>OLD.fecha_fin OR NEW.regla_id<>OLD.regla_id)) OR (NEW.estado='activo' AND (NEW.tramo_desde<>(NEW.modificado_en AT TIME ZONE 'Europe/Madrid')::date OR NEW.tramo_desde<OLD.fecha_terminacion))
   OR (to_jsonb(NEW)-ARRAY['version','estado','fecha_fin','regla_id','tramo_desde','fecha_terminacion','motivo','motivo_reapertura','modificado_por','modificado_en']) IS DISTINCT FROM
      (to_jsonb(OLD)-ARRAY['version','estado','fecha_fin','regla_id','tramo_desde','fecha_terminacion','motivo','motivo_reapertura','modificado_por','modificado_en']) THEN
   RAISE EXCEPTION 'expediente.seguimiento_invalido';
  END IF;
  IF current_user='etf_propietario' AND NEW.modificado_por IS NULL THEN
   IF NEW.estado<>'terminado' OR NEW.fecha_fin>(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date
    OR NEW.fecha_terminacion<>NEW.fecha_fin OR NEW.motivo<>'PLAZO_CUMPLIDO' THEN RAISE EXCEPTION 'expediente.seguimiento_invalido'; END IF;
  ELSE
   IF NEW.modificado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
    OR (NEW.estado='activo' AND (NOT coalesce('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])),false) OR NEW.fecha_fin IS DISTINCT FROM catalogo.fin_seguimiento(NEW.tramo_desde,NEW.regla_id,(NEW.modificado_en AT TIME ZONE 'Europe/Madrid')::date)))
    OR (NEW.estado='terminado' AND (NEW.fecha_terminacion>=(NEW.modificado_en AT TIME ZONE 'Europe/Madrid')::date+1 OR NEW.fecha_terminacion>=NEW.fecha_fin))
    OR NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.expediente_id AND e.estado='cerrada') THEN
    RAISE EXCEPTION 'expediente.seguimiento_invalido';
   END IF;
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_seguimiento() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_seguimiento BEFORE INSERT OR UPDATE ON expediente.seguimiento FOR EACH ROW EXECUTE FUNCTION expediente.proteger_seguimiento();
CREATE FUNCTION expediente.impedir_reapertura_con_seguimiento() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF OLD.estado='cerrada' AND NEW.estado<>'cerrada' AND EXISTS(SELECT 1 FROM expediente.seguimiento WHERE expediente_id=NEW.id AND estado='activo') THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.impedir_reapertura_con_seguimiento() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER impedir_reapertura_con_seguimiento BEFORE UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.impedir_reapertura_con_seguimiento();
CREATE FUNCTION expediente.vencer_seguimientos() RETURNS TABLE(expediente_id uuid)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF nullif(current_setting('app.usuario_id',true),'') IS NOT NULL OR nullif(current_setting('app.perfiles',true),'') IS NOT NULL THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY UPDATE expediente.seguimiento s SET version=s.version+1,estado='terminado',fecha_terminacion=s.fecha_fin,
 motivo='PLAZO_CUMPLIDO',modificado_por=NULL,modificado_en=clock_timestamp()
 WHERE s.estado='activo' AND s.fecha_fin<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date RETURNING s.expediente_id;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.vencer_seguimientos() FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.vencer_seguimientos() TO etf_app;
-- Contrato de lectura estadística: la historia conserva intervalos de cada
-- tramo activo; una reapertura no borra una terminación anterior.
CREATE VIEW expediente.consulta_seguimientos WITH(security_invoker=true) AS
 SELECT h.seguimiento_id,h.expediente_id,h.equipo_id,h.version,h.datos->>'episodio_id' AS episodio_id,
 (h.datos->>'fecha_cese')::date AS fecha_cese,(h.datos->>'tramo_desde')::date AS vigente_desde,
 coalesce((SELECT (fin.datos->>'fecha_terminacion')::date FROM expediente.seguimiento_historia fin
  WHERE fin.seguimiento_id=h.seguimiento_id AND fin.version>h.version AND fin.accion IN ('terminar','vencer')
  ORDER BY fin.version LIMIT 1),(h.datos->>'fecha_fin')::date) AS vigente_hasta,
 h.registrado_en FROM expediente.seguimiento_historia h WHERE h.accion IN ('iniciar','reabrir');
REVOKE ALL ON expediente.consulta_seguimientos FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.consulta_seguimientos TO etf_app,etf_copia;

CREATE FUNCTION expediente.comprobar_enlace_traslado() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF NEW.origen='traslado' AND NOT EXISTS(SELECT 1 FROM expediente.traslado t WHERE t.destino_id=NEW.id AND t.origen_id=NEW.traslado_origen_id AND t.equipo_destino_id=NEW.traslado_equipo_id) THEN
  RAISE EXCEPTION 'expediente.traslado_invalido';
 END IF;
 RETURN NULL;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.comprobar_enlace_traslado() FROM PUBLIC,etf_app,etf_copia;
CREATE CONSTRAINT TRIGGER comprobar_enlace_traslado AFTER INSERT OR UPDATE ON expediente.expediente
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expediente.comprobar_enlace_traslado();
-- La aceptación de un traslado materializa su copia. Coordinación sólo puede
-- añadir la historia inicial de ese destino; no obtiene edición de familias.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['adulto','familia_revision','adultos_revision','adulto_revision'] LOOP
  EXECUTE format($sql$CREATE POLICY coordinacion_copia ON expediente.%1$I FOR INSERT TO etf_app WITH CHECK (
   nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
   AND 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
   AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=%1$I.expediente_id AND e.equipo_id=%1$I.equipo_id AND e.origen='traslado'
    AND e.estado='aceptada' AND EXISTS(SELECT 1 FROM expediente.episodio p WHERE p.expediente_id=e.id AND p.reapertura_de_id IS NULL AND p.cierre_fecha IS NULL AND p.creado_por=e.modificado_por AND p.creado_en=e.modificado_en) AND e.modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid))$sql$,tabla);
 END LOOP;
END $$;
