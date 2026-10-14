SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('evaluacion.instrumento_aplicado') IS NULL
 OR to_regclass('evaluacion.retirada_personal') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;

-- El módulo guarda su propia marca para impedir que se repongan textos después.
CREATE TABLE evaluacion.retirada_personal (
 expediente_id uuid PRIMARY KEY REFERENCES expediente.expediente(id),
 retirado_en timestamptz NOT NULL CHECK(isfinite(retirado_en))
);
ALTER TABLE evaluacion.retirada_personal ENABLE ROW LEVEL SECURITY;
ALTER TABLE evaluacion.retirada_personal FORCE ROW LEVEL SECURITY;
REVOKE ALL ON evaluacion.retirada_personal FROM PUBLIC,etf_app,etf_copia;
CREATE POLICY propietario ON evaluacion.retirada_personal TO etf_propietario USING(true) WITH CHECK(true);
GRANT SELECT ON evaluacion.retirada_personal TO etf_copia;
CREATE POLICY copia ON evaluacion.retirada_personal FOR SELECT TO etf_copia USING(true);

-- La excepción no cambia versión, sujeto ni puntuaciones. Sólo permite al
-- propietario retirar descripción; etf_app conserva su control de versiones.
CREATE OR REPLACE FUNCTION evaluacion.proteger_aplicacion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,evaluacion AS $$
BEGIN
 IF current_user='etf_propietario'
 AND (to_jsonb(NEW)-'datos')=(to_jsonb(OLD)-'datos')
 AND NEW.datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(OLD.datos)
  WHERE key=ANY(ARRAY['id','expediente_id','equipo_id','instrumento','momento','sujeto','fecha','version_escala_id','version','puntuaciones'])) THEN RETURN NEW; END IF;
 IF NEW.id<>OLD.id OR NEW.expediente_id<>OLD.expediente_id OR NEW.equipo_id<>OLD.equipo_id OR NEW.instrumento<>OLD.instrumento
 OR NEW.momento<>OLD.momento OR NEW.sujeto<>OLD.sujeto OR NEW.adulto_id IS DISTINCT FROM OLD.adulto_id OR NEW.version<>OLD.version+1 THEN
  RAISE EXCEPTION 'evaluacion.peticion_invalida' USING ERRCODE='23514';
 END IF; RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION evaluacion.proteger_aplicacion() FROM PUBLIC,etf_app,etf_copia;

-- Puerto interno del módulo: no es ejecutable por la conexión de aplicación.
-- Expediente lo llama como propietario después de comprobar las dos personas;
-- datos, marca y auditoría del acto se confirman en esa misma transacción.
CREATE FUNCTION evaluacion.api_anonimizar_expediente(p_id uuid,p_ahora timestamptz) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog,evaluacion AS $$
BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_app'
 OR nullif(current_setting('app.usuario_id',true),'') IS NULL
 OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
  RAISE EXCEPTION 'evaluacion.acceso_denegado' USING ERRCODE='42501'; END IF;
 IF p_id IS NULL OR p_ahora IS NULL OR NOT isfinite(p_ahora) THEN RAISE EXCEPTION 'evaluacion.peticion_invalida'; END IF;
 IF EXISTS(SELECT 1 FROM evaluacion.retirada_personal WHERE expediente_id=p_id) THEN RETURN; END IF;
 UPDATE evaluacion.instrumento_aplicado SET datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(datos)
  WHERE key=ANY(ARRAY['id','expediente_id','equipo_id','codigo','fecha','sujeto','momento'])) WHERE expediente_id=p_id;
 UPDATE evaluacion.aplicacion_escala SET datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(datos)
  WHERE key=ANY(ARRAY['id','expediente_id','equipo_id','instrumento','momento','sujeto','fecha','version_escala_id','version','puntuaciones'])) WHERE expediente_id=p_id;
 DELETE FROM evaluacion.peticion WHERE expediente_id=p_id;
 INSERT INTO evaluacion.retirada_personal VALUES(p_id,p_ahora);
END $$;
REVOKE EXECUTE ON FUNCTION evaluacion.api_anonimizar_expediente(uuid,timestamptz) FROM PUBLIC,etf_app,etf_copia;

CREATE FUNCTION evaluacion.impedir_cambios_retirados() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,evaluacion AS $$
DECLARE recurso uuid:=coalesce((to_jsonb(NEW)->>'expediente_id')::uuid,(to_jsonb(OLD)->>'expediente_id')::uuid);
BEGIN
 IF EXISTS(SELECT 1 FROM evaluacion.retirada_personal WHERE expediente_id=recurso) THEN RAISE EXCEPTION 'evaluacion.peticion_invalida'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION evaluacion.impedir_cambios_retirados() FROM PUBLIC,etf_app,etf_copia;
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['instrumento_aplicado','aplicacion_escala','respuesta','peticion'] LOOP
  EXECUTE format('CREATE TRIGGER impedir_cambios_retirados BEFORE INSERT OR UPDATE OR DELETE ON evaluacion.%I FOR EACH ROW EXECUTE FUNCTION evaluacion.impedir_cambios_retirados()',tabla);
 END LOOP;
END $$;
