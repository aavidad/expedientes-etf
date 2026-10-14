SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo IN ('finalizacion/anexo11','finalizacion/anexo12')) THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES ('finalizacion/anexo11',false),('finalizacion/anexo12',false);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('finalizacion/anexo11','es','1.0.0','plantillas/finalizacion/anexo11/v1.0.0.docx',decode('18a824cf3f690922d465c472b4c41eb4308f529d53b9763c839e087a4056f4b7','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('finalizacion/anexo12','es','1.0.0','plantillas/finalizacion/anexo12/v1.0.0.docx',decode('2f29b450be0ed48cc381539c312c61daacb801f0e1afe00b29edc6f784a89c15','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('finalizacion/anexo11','en','1.0.0','plantillas/finalizacion/anexo11/v1.0.0.docx',decode('d7f6c83986d735360e047e4faddaf4bf1457e9306bc2334b52d3db483a7670f3','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('finalizacion/anexo12','en','1.0.0','plantillas/finalizacion/anexo12/v1.0.0.docx',decode('96d98284a05f5a8d3c05fbced6e130f5ca51d83c1acb160d696feaf6208bc001','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version)
 SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12');
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['plantilla','referencia_activa'] LOOP
  EXECUTE format('CREATE POLICY finalizacion_lectura ON documento.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND
   ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND
   tipo_documento IN (''finalizacion/anexo11'',''finalizacion/anexo12''))',tabla);
 END LOOP;
END $$;
CREATE POLICY finalizacion_lectura ON documento.emision FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12') AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY finalizacion_escritura ON documento.emision FOR INSERT TO etf_app WITH CHECK (
 generada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12') AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY finalizacion_actor ON documento.peticion_idempotente TO etf_app USING (
 usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))) WITH CHECK (
 usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
-- Las políticas heredadas de Coordinación no amplían la emisión de Anexos:
-- también con ambos perfiles se exige el equipo asignado en la sesión.
CREATE POLICY finalizacion_ambito ON documento.emision AS RESTRICTIVE FOR INSERT TO etf_app
 WITH CHECK (tipo_documento NOT IN ('finalizacion/anexo11','finalizacion/anexo12') OR (
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
