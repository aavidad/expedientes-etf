SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo='salida/anexo6') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES('salida/anexo6',false);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('salida/anexo6','es','1.0.0','plantillas/salida/anexo6/v1.0.0.docx',decode('81b276cd308b692c8f8aecfca841c3a62866177bf3e572bd2c56f5df9be5cded','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 VALUES('salida/anexo6','en','1.0.0','plantillas/salida/anexo6/v1.0.0.docx',decode('73bb9622356a090cb1a754b4dba48986427e06776bad58833a3444f49520effe','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.referencia_activa SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento='salida/anexo6';

-- El acceso nuevo se limita al Anexo 6 y al equipo de la sesión.
CREATE POLICY plantilla_anexo6 ON documento.plantilla FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND tipo_documento='salida/anexo6'
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY referencia_anexo6 ON documento.referencia_activa FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND tipo_documento='salida/anexo6'
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY emision_anexo6_lectura ON documento.emision FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND tipo_documento='salida/anexo6'
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY emision_anexo6_crear ON documento.emision FOR INSERT TO etf_app WITH CHECK(
 generada_por::text=nullif(current_setting('app.usuario_id',true),'') AND tipo_documento='salida/anexo6'
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY peticion_anexo6 ON documento.peticion_idempotente TO etf_app
 USING(usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.tipo_documento='salida/anexo6'))
 WITH CHECK(usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.tipo_documento='salida/anexo6'));
