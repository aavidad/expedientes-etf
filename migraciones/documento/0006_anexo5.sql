SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo='salida/spm/anexo5') THEN
 RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
INSERT INTO documento.tipo_documento VALUES('salida/spm/anexo5',false);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'salida/spm/anexo5',codigo,'1.0.0','plantillas/salida/spm/anexo5/v1.0.0.docx',decode('ec7f18849ec5346dbf48f977ec1b16ad1a337824ef71a300660da36c30faee2a','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.referencia_activa SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento='salida/spm/anexo5';
-- El profesional sólo obtiene plantillas y emisiones del Anexo 5 de su equipo.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['plantilla','referencia_activa'] LOOP
 EXECUTE format('CREATE POLICY spm_profesional ON documento.%I FOR SELECT TO etf_app USING (
 tipo_documento=''salida/spm/anexo5'' AND nullif(current_setting(''app.usuario_id'',true),'''') IS NOT NULL AND
 ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')))',tabla);
 END LOOP;
END $$;
CREATE POLICY spm_profesional_lectura ON documento.emision FOR SELECT TO etf_app USING (
 tipo_documento='salida/spm/anexo5' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY spm_profesional_escritura ON documento.emision FOR INSERT TO etf_app WITH CHECK (
 tipo_documento='salida/spm/anexo5' AND generada_por::text=nullif(current_setting('app.usuario_id',true),'') AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY spm_profesional_peticion ON documento.peticion_idempotente TO etf_app
 USING (usuario_id::text=nullif(current_setting('app.usuario_id',true),'') AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK (usuario_id::text=nullif(current_setting('app.usuario_id',true),'') AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND EXISTS (
 SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.tipo_documento='salida/spm/anexo5'));
