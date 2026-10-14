SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('estadistica.memoria') IS NULL OR to_regclass('estadistica.memoria_revision') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
CREATE TABLE estadistica.memoria_revision (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), memoria_id uuid NOT NULL REFERENCES estadistica.memoria(id),
 revision integer NOT NULL CHECK(revision>0), informe jsonb NOT NULL CHECK(jsonb_typeof(informe)='object'),
 huella text NOT NULL CHECK(huella ~ '^[0-9a-f]{64}$'), motor text NOT NULL,
 cerrada_por uuid NOT NULL REFERENCES identidad.usuario(id),cerrada_en timestamptz NOT NULL,
 UNIQUE(memoria_id,revision)
);
ALTER TABLE estadistica.memoria ADD CONSTRAINT memoria_revision_actual_fk
 FOREIGN KEY(id,revision_actual) REFERENCES estadistica.memoria_revision(memoria_id,revision) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE estadistica.memoria_base_equipo (
 memoria_id uuid NOT NULL,revision integer NOT NULL,equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),
 informe jsonb NOT NULL CHECK(jsonb_typeof(informe)='object' AND NOT informe ? 'Narrativas' AND NOT informe ? 'Composicion' AND NOT informe ? 'Fuentes'),
 PRIMARY KEY(memoria_id,revision,equipo_id),
 FOREIGN KEY(memoria_id,revision) REFERENCES estadistica.memoria_revision(memoria_id,revision)
);
CREATE TABLE estadistica.memoria_acto (
 memoria_id uuid NOT NULL REFERENCES estadistica.memoria(id),version integer NOT NULL CHECK(version>0),
 accion text NOT NULL CHECK(accion IN ('cerrar','reabrir')),motivo text,
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),ocurrido_en timestamptz NOT NULL,
 CHECK((accion='cerrar' AND motivo IS NULL) OR (accion='reabrir' AND motivo IS NOT NULL AND length(btrim(motivo)) BETWEEN 1 AND 4000)),
 PRIMARY KEY(memoria_id,version)
);
CREATE TABLE estadistica.peticion_idempotente (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),clave text NOT NULL CHECK(length(clave) BETWEEN 1 AND 128),
 ruta text NOT NULL,huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[0-9a-f]{64}$'),
 respuesta jsonb NOT NULL CHECK(jsonb_typeof(respuesta)='object'),creada_en timestamptz NOT NULL,
 PRIMARY KEY(usuario_id,clave,ruta,creada_en)
);
CREATE INDEX memoria_idempotencia_busqueda ON estadistica.peticion_idempotente(usuario_id,clave,ruta,creada_en DESC);
CREATE TABLE estadistica.memoria_discrepancia (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),memoria_id uuid NOT NULL,revision integer NOT NULL,
 equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),expediente_id uuid,evento_id bigint,
 apartados text[] NOT NULL CHECK(cardinality(apartados)>0 AND apartados <@ ARRAY['1','2','3','4','5','6','7','8','9','10','11.1','11.2','12.1','12.2','13.1','13.2','13.3','14','14.1','15','16','17','18','19','20','21','22','23','24','25','26','27','28']::text[]),
 detectada_en timestamptz NOT NULL,
 FOREIGN KEY(memoria_id,revision) REFERENCES estadistica.memoria_revision(memoria_id,revision),
 UNIQUE NULLS NOT DISTINCT(memoria_id,revision,equipo_id,evento_id)
);
CREATE INDEX memoria_discrepancia_lectura ON estadistica.memoria_discrepancia(memoria_id,revision,detectada_en);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['memoria_revision','memoria_base_equipo','memoria_acto','peticion_idempotente','memoria_discrepancia'] LOOP
  EXECUTE format('ALTER TABLE estadistica.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE estadistica.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON estadistica.%I FROM PUBLIC,etf_app,etf_copia',t);
  EXECUTE format('GRANT SELECT ON estadistica.%I TO etf_copia',t);
 END LOOP;
END $$;
GRANT SELECT,INSERT ON estadistica.memoria_revision,estadistica.memoria_base_equipo,estadistica.memoria_acto,estadistica.peticion_idempotente TO etf_app;
GRANT SELECT ON estadistica.memoria_discrepancia TO etf_app;
GRANT UPDATE(estado,revision_actual) ON estadistica.memoria TO etf_app;
CREATE POLICY revision_leer ON estadistica.memoria_revision FOR SELECT TO etf_app
 USING(EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id));
CREATE POLICY revision_insertar ON estadistica.memoria_revision FOR INSERT TO etf_app
 WITH CHECK(cerrada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id AND m.estado='cerrada' AND m.revision_actual=revision));
CREATE POLICY base_leer ON estadistica.memoria_base_equipo FOR SELECT TO etf_app
 USING(EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id));
CREATE POLICY base_insertar ON estadistica.memoria_base_equipo FOR INSERT TO etf_app
 WITH CHECK(EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id AND m.estado='cerrada' AND m.revision_actual=revision));
CREATE POLICY acto_leer ON estadistica.memoria_acto FOR SELECT TO etf_app
 USING(EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id));
CREATE POLICY acto_insertar ON estadistica.memoria_acto FOR INSERT TO etf_app
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id AND (accion='cerrar' OR 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')))));
CREATE POLICY idempotencia_propia ON estadistica.peticion_idempotente TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) OR 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'))))
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) OR 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'))));
CREATE POLICY discrepancia_coordinacion ON estadistica.memoria_discrepancia FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')));
-- Las funciones internas tienen acceso explícito y acotado bajo FORCE RLS.
CREATE POLICY memoria_interna ON estadistica.memoria TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY base_interna ON estadistica.memoria_base_equipo TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY discrepancia_interna ON estadistica.memoria_discrepancia TO etf_propietario USING(true) WITH CHECK(true);
CREATE FUNCTION estadistica.proteger_memoria() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,estadistica AS $$
BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.estado<>'borrador' OR NEW.revision_actual IS NOT NULL THEN RAISE EXCEPTION 'estadistica.memoria_acto_invalido'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.anio<>OLD.anio OR NEW.equipo_id IS DISTINCT FROM OLD.equipo_id OR NEW.id<>OLD.id OR NEW.version<>OLD.version+1 THEN
  RAISE EXCEPTION 'estadistica.memoria_acto_invalido';
 END IF;
 IF OLD.estado='cerrada' THEN
  IF NEW.estado<>'borrador' OR NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'))) OR NEW.textos IS DISTINCT FROM OLD.textos OR NEW.revision_actual IS DISTINCT FROM OLD.revision_actual THEN
   RAISE EXCEPTION 'estadistica.memoria_cerrada';
  END IF;
 END IF;
 IF NEW.estado IS DISTINCT FROM OLD.estado THEN
  IF NOT EXISTS(SELECT 1 FROM estadistica.memoria_acto a WHERE a.memoria_id=OLD.id AND a.version=NEW.version AND a.usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND a.accion=CASE WHEN NEW.estado='cerrada' THEN 'cerrar' ELSE 'reabrir' END) THEN
   RAISE EXCEPTION 'estadistica.memoria_acto_invalido';
  END IF;
  IF NEW.estado='cerrada' AND NEW.revision_actual IS DISTINCT FROM coalesce(OLD.revision_actual,0)+1 THEN RAISE EXCEPTION 'estadistica.memoria_acto_invalido'; END IF;
 ELSIF NEW.revision_actual IS DISTINCT FROM OLD.revision_actual THEN RAISE EXCEPTION 'estadistica.memoria_acto_invalido';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION estadistica.proteger_memoria() FROM PUBLIC;
CREATE TRIGGER memoria_inmutable BEFORE INSERT OR UPDATE ON estadistica.memoria FOR EACH ROW EXECUTE FUNCTION estadistica.proteger_memoria();
CREATE FUNCTION estadistica.bases_equipo(p_equipo uuid) RETURNS TABLE(memoria_id uuid,revision integer,informe jsonb)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,estadistica AS $$
BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR p_equipo IS NULL OR NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND p_equipo=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))) THEN
  RAISE EXCEPTION 'estadistica.memoria_acceso_denegado';
 END IF;
 RETURN QUERY SELECT b.memoria_id,b.revision,b.informe FROM estadistica.memoria_base_equipo b JOIN estadistica.memoria m ON m.id=b.memoria_id
 WHERE b.equipo_id=p_equipo AND m.estado='cerrada' AND m.revision_actual=b.revision ORDER BY b.memoria_id;
END $$;
REVOKE EXECUTE ON FUNCTION estadistica.bases_equipo(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.bases_equipo(uuid) TO etf_app;
CREATE FUNCTION estadistica.avisar_discrepancia(p_memoria uuid,p_revision integer,p_equipo uuid,p_expediente uuid,p_evento bigint,p_apartados text[],p_fecha timestamptz) RETURNS void
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,estadistica AS $$
BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND p_equipo=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))) THEN
  RAISE EXCEPTION 'estadistica.memoria_acceso_denegado';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM estadistica.memoria_base_equipo b JOIN estadistica.memoria m ON m.id=b.memoria_id
  WHERE b.memoria_id=p_memoria AND b.revision=p_revision AND b.equipo_id=p_equipo AND m.estado='cerrada' AND m.revision_actual=p_revision) THEN
  RAISE EXCEPTION 'estadistica.memoria_revision_inexistente';
 END IF;
 INSERT INTO estadistica.memoria_discrepancia(memoria_id,revision,equipo_id,expediente_id,evento_id,apartados,detectada_en)
 VALUES(p_memoria,p_revision,p_equipo,p_expediente,p_evento,p_apartados,p_fecha) ON CONFLICT DO NOTHING;
END $$;
REVOKE EXECUTE ON FUNCTION estadistica.avisar_discrepancia(uuid,integer,uuid,uuid,bigint,text[],timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.avisar_discrepancia(uuid,integer,uuid,uuid,bigint,text[],timestamptz) TO etf_app;
