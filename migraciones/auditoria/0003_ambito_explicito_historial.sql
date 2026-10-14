SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regprocedure('auditoria.historial_expediente(uuid)') IS NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- Las políticas permisivas del propietario se combinan con OR. La función
-- verifica además el ámbito de sesión, sin depender de políticas de otros usos.
CREATE OR REPLACE FUNCTION auditoria.historial_expediente(p_id uuid)
 RETURNS TABLE(id bigint,accion text,ocurrido_en timestamptz,usuario_id uuid,resultado text)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,auditoria AS $$
BEGIN
 IF nullif(current_setting('app.usuario_id',true),'') IS NULL
 OR NOT coalesce(string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf']::text[],false)
 OR NOT EXISTS(SELECT 1 FROM expediente.api_ambito_historial e WHERE e.id=p_id
 AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')='{}'
      OR e.equipo_id=ANY(current_setting('app.equipos',true)::uuid[]))
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
      OR e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT e.id,e.accion,e.ocurrido_en,e.usuario_id,e.resultado
 FROM auditoria.evento e WHERE e.expediente_id=p_id
 ORDER BY e.ocurrido_en DESC,e.id DESC LIMIT 100;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.historial_expediente(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.historial_expediente(uuid) TO etf_app;
