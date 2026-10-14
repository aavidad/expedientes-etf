SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.registro_peticion') IS NULL
 OR to_regclass('documento.retirada_personal') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
CREATE TABLE documento.retirada_personal (
 expediente_id uuid PRIMARY KEY REFERENCES expediente.expediente(id),
 retirado_en timestamptz NOT NULL CHECK(isfinite(retirado_en))
);
ALTER TABLE documento.retirada_personal ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.retirada_personal FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.retirada_personal FROM PUBLIC,etf_app,etf_copia;
CREATE POLICY propietario ON documento.retirada_personal TO etf_propietario USING(true) WITH CHECK(true);
GRANT SELECT ON documento.retirada_personal TO etf_copia;
CREATE POLICY copia ON documento.retirada_personal FOR SELECT TO etf_copia USING(true);
CREATE POLICY conservacion_lectura ON documento.emision FOR SELECT TO etf_propietario USING(true);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['registro_salida','registro_peticion','peticion_idempotente','peticion_spaf'] LOOP
  EXECUTE format('CREATE POLICY conservacion_propietario ON documento.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
 END LOOP;
END $$;

-- La retirada sólo sustituye los dos campos libres. Conserva estado, fechas,
-- versión, autoría, emisión y huellas; no altera la inmutabilidad de emisiones.
CREATE OR REPLACE FUNCTION documento.comprobar_registro_salida() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN
 IF TG_OP='UPDATE' AND current_user='etf_propietario'
 AND (to_jsonb(NEW)-ARRAY['numero_registro','referencia_externa'])=(to_jsonb(OLD)-ARRAY['numero_registro','referencia_externa'])
 AND NEW.numero_registro IS NOT DISTINCT FROM (CASE WHEN OLD.numero_registro IS NULL THEN NULL ELSE '—' END)
 AND NEW.referencia_externa IS NOT DISTINCT FROM (CASE WHEN OLD.referencia_externa IS NULL THEN NULL ELSE '—' END) THEN RETURN NEW; END IF;
 IF NOT EXISTS(SELECT 1 FROM documento.emision e JOIN documento.tipo_documento t ON t.codigo=e.tipo_documento WHERE e.id=NEW.emision_id AND t.requiere_registro) THEN
  RAISE EXCEPTION 'documento.registro_solo_oficios' USING ERRCODE='42501'; END IF;
 IF TG_OP='INSERT' AND (NEW.estado<>'pendiente' OR NEW.version<>1) THEN RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (NEW.emision_id<>OLD.emision_id OR NEW.version<>OLD.version+1 OR NEW.estado<>'anotado'
 OR NEW.anotado_por::text IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')) THEN
  RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION documento.comprobar_registro_salida() FROM PUBLIC,etf_app,etf_copia;

-- Puerto interno. No entrega datos ni permite a Administración abrir fichas.
CREATE FUNCTION documento.api_anonimizar_expediente(p_id uuid,p_ahora timestamptz) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_app'
 OR nullif(current_setting('app.usuario_id',true),'') IS NULL
 OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
  RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501'; END IF;
 IF p_id IS NULL OR p_ahora IS NULL OR NOT isfinite(p_ahora) THEN RAISE EXCEPTION 'documento.peticion_invalida'; END IF;
 IF EXISTS(SELECT 1 FROM documento.retirada_personal WHERE expediente_id=p_id) THEN RETURN; END IF;
 UPDATE documento.registro_salida r SET numero_registro=CASE WHEN numero_registro IS NULL THEN NULL ELSE '—' END,
  referencia_externa=CASE WHEN referencia_externa IS NULL THEN NULL ELSE '—' END
 FROM documento.emision e WHERE r.emision_id=e.id AND e.expediente_id=p_id;
 DELETE FROM documento.registro_peticion r USING documento.emision e WHERE r.emision_id=e.id AND e.expediente_id=p_id;
 DELETE FROM documento.peticion_idempotente r USING documento.emision e WHERE r.emision_id=e.id AND e.expediente_id=p_id;
 DELETE FROM documento.peticion_spaf r USING documento.emision e WHERE r.emision_id=e.id AND e.expediente_id=p_id;
 INSERT INTO documento.retirada_personal VALUES(p_id,p_ahora);
END $$;
REVOKE EXECUTE ON FUNCTION documento.api_anonimizar_expediente(uuid,timestamptz) FROM PUBLIC,etf_app,etf_copia;

CREATE FUNCTION documento.impedir_cambios_retirados() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,documento AS $$
DECLARE recurso uuid; id_emision uuid;
BEGIN
 IF TG_TABLE_NAME='emision' THEN recurso:=NEW.expediente_id;
 ELSE
  id_emision:=coalesce((to_jsonb(NEW)->>'emision_id')::uuid,(to_jsonb(OLD)->>'emision_id')::uuid);
  SELECT expediente_id INTO recurso FROM documento.emision WHERE id=id_emision;
 END IF;
 IF EXISTS(SELECT 1 FROM documento.retirada_personal WHERE expediente_id=recurso) THEN RAISE EXCEPTION 'documento.peticion_invalida'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION documento.impedir_cambios_retirados() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER impedir_cambios_retirados BEFORE INSERT ON documento.emision FOR EACH ROW EXECUTE FUNCTION documento.impedir_cambios_retirados();
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['registro_salida','registro_peticion','peticion_idempotente','peticion_spaf'] LOOP
  EXECUTE format('CREATE TRIGGER impedir_cambios_retirados BEFORE INSERT OR UPDATE ON documento.%I FOR EACH ROW EXECUTE FUNCTION documento.impedir_cambios_retirados()',tabla);
 END LOOP;
END $$;
