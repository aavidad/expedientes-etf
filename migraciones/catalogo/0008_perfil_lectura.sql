SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.firmante_textos') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='catalogo' AND tablename='equipo' AND policyname='lectura') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Corrige el código de perfil conservando los permisos de los demás perfiles.
-- No cambia políticas del SMI ni concede escrituras.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['equipo','centro_ss','municipio'] LOOP
  EXECUTE format('DROP POLICY lectura ON catalogo.%I',tabla);
  EXECUTE format('CREATE POLICY lectura ON catalogo.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''') IS NOT NULL
   AND string_to_array(current_setting(''app.perfiles'',true),'','') &&
   ARRAY[''coordinacion'',''profesional_etf'',''consulta'',''administracion'',''sistema''])',tabla);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['parametro','tramo','version_textos','firmante','firmante_textos'] LOOP
  EXECUTE format('DROP POLICY lectura ON catalogo.%I',tabla);
  EXECUTE format('CREATE POLICY lectura ON catalogo.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''') IS NOT NULL
   AND string_to_array(current_setting(''app.perfiles'',true),'','') &&
   ARRAY[''administracion'',''coordinacion'',''profesional_etf'',''sistema''])',tabla);
 END LOOP;
END $$;
DROP POLICY lectura ON catalogo.version;
CREATE POLICY lectura ON catalogo.version FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND (
 'administracion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR (estado='publicada' AND string_to_array(current_setting('app.perfiles',true),',')
 && ARRAY['coordinacion','profesional_etf','consulta','sistema'])));
