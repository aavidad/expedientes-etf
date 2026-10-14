SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$
BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.expediente') IS NULL
 OR to_regclass('expediente.api_ambito_historial') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- El propietario sin acceso de inicio sólo lee el ámbito autorizado dentro
-- de la función de auditoría. FORCE RLS sigue vigente y no concede escritura.
CREATE POLICY expediente_ambito_historial ON expediente.expediente
 FOR SELECT TO etf_propietario USING (
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')='{}'
 OR equipo_id=ANY(current_setting('app.equipos',true)::uuid[])) AND (
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))));

-- Contrato de lectura de expediente para auditoría, sin nombres ni otros datos.
CREATE VIEW expediente.api_ambito_historial WITH (security_invoker=true,security_barrier=true)
 AS SELECT id,equipo_id FROM expediente.expediente;
REVOKE ALL ON expediente.api_ambito_historial FROM PUBLIC,etf_app,etf_copia;
