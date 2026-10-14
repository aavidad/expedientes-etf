SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('paquete.referencia_activa') IS NULL
 OR to_regclass('paquete.idioma_instalado') IS NULL
 OR EXISTS(SELECT 1 FROM pg_policy WHERE polrelid='paquete.referencia_activa'::regclass AND polname='consentimiento_profesional_lectura') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Se leen las referencias activas y los idiomas verificados; no se amplía la escritura.
CREATE POLICY consentimiento_profesional_lectura ON paquete.referencia_activa FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND tipo IN ('catalogo','idioma'));
CREATE POLICY consentimiento_profesional_lectura ON paquete.idioma_instalado FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND estado='verificada');
