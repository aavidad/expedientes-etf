SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL
 OR NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='expediente' AND table_name='expediente' AND column_name='fecha_no_aceptacion')
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo IN ('entrada/no_aceptacion','entrada/no_aceptacion/lista_espera')) THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES
 ('entrada/no_aceptacion',false),('entrada/no_aceptacion/lista_espera',true);

-- Se activan las versiones oficiales ya convertidas, sin alterar sus archivos.
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'entrada/no_aceptacion',codigo,'1.0.0','plantillas/entrada/no_aceptacion/v1.0.0.docx',decode('61a4bf9d0d9a846f99fd3116169f562348157b72d4aa959842466c5c41422817','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'entrada/no_aceptacion/lista_espera',codigo,'1.0.0','plantillas/entrada/no_aceptacion/lista_espera/v1.0.0.docx',decode('29886b8dc296b4627befba0c7c42914828ae775011577aa018035404811f8128','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version)
 SELECT tipo_documento,idioma,id,1 FROM documento.plantilla
 WHERE tipo_documento IN ('entrada/no_aceptacion','entrada/no_aceptacion/lista_espera');
