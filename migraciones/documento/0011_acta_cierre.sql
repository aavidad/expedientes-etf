SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo='cierre/acta_cierre') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES('cierre/acta_cierre',true);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('cierre/acta_cierre','es','1.0.0','plantillas/cierre/acta_cierre/v1.0.0.docx',decode('60fadf6f8e1acf53d52775d92a832b84bf283b1c074d2629bf4e22b9e321f206','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('cierre/acta_cierre','en','1.0.0','plantillas/cierre/acta_cierre/v1.0.0.docx',decode('bf2ef4ca6675e3905534a44c5dc17bf738d5691aa3ef6d887dffea3eaafd20d5','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1');
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version)
 SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento='cierre/acta_cierre';
CREATE POLICY acta_cierre_coordinacion ON documento.emision AS RESTRICTIVE FOR INSERT TO etf_app
 WITH CHECK(tipo_documento<>'cierre/acta_cierre' OR (
 generada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))));
