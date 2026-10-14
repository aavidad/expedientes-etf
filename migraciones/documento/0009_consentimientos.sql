SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo IN ('consentimiento/anexo1','consentimiento/anexo2')) THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES
 ('consentimiento/anexo1',false),('consentimiento/anexo2',false);

-- Cada idioma activa su versión inmutable; no se guardan documentos de familias.
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('consentimiento/anexo1','es','1.0.0','plantillas/consentimiento/anexo1/v1.0.0.docx',decode('528fe7f9376178176ee4a10a00d5af921cbedd28976ec8f73e3d216ff0f2accb','hex'),'consentimiento',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('consentimiento/anexo2','es','1.0.0','plantillas/consentimiento/anexo2/v1.0.0.docx',decode('8d4f5f2cb7ec7a989efafe10ecf2ddc586fc1c8c4514b3882acde5062e5bf1cd','hex'),'consentimiento',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('consentimiento/anexo1','en','1.0.0','plantillas/consentimiento/anexo1/v1.0.0.docx',decode('04cac690ee3f20b0add1bd1c34dd33fe93f39a9948a0d3f1e5c786834d2a95ab','hex'),'consentimiento',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('consentimiento/anexo2','en','1.0.0','plantillas/consentimiento/anexo2/v1.0.0.docx',decode('fd52773198ec7df7770825706054bcef49cfc190c5f790d08834d5394cfc6965','hex'),'consentimiento',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version)
 SELECT tipo_documento,idioma,id,1 FROM documento.plantilla
 WHERE tipo_documento IN ('consentimiento/anexo1','consentimiento/anexo2');

-- El perfil vigente del ETF sólo emite consentimientos de su propio equipo.
CREATE POLICY consentimiento_profesional ON documento.plantilla FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])));
CREATE POLICY consentimiento_profesional ON documento.referencia_activa FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])));
CREATE POLICY consentimiento_profesional_lectura ON documento.emision FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY consentimiento_profesional_escritura ON documento.emision FOR INSERT TO etf_app
 WITH CHECK(generada_por::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])
 AND tipo_documento IN ('consentimiento/anexo1','consentimiento/anexo2'));
CREATE POLICY consentimiento_profesional ON documento.peticion_idempotente TO etf_app
 USING(usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])))
 WITH CHECK(usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id
 AND e.tipo_documento IN ('consentimiento/anexo1','consentimiento/anexo2')
 AND e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
