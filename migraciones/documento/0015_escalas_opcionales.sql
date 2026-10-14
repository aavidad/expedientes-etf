SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('documento.activacion_plantilla') IS NULL
 OR (SELECT count(*) FROM documento.plantilla WHERE tipo_documento IN
 ('finalizacion/anexo11','finalizacion/anexo12') AND idioma IN ('es','en') AND version='1.0.1') <> 4
 THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;

-- El paquete inicial conserva todas las versiones anteriores y sus emisiones.
-- No se sustituye una publicación de Administración ni se declara un antivirus.
WITH nuevas(tipo,idioma,huella) AS (VALUES
 ('finalizacion/anexo11','es','f6be3b6a1acc646c81d2a41499d069423008f20cd851e595e70e527c88b5fda8'),
 ('finalizacion/anexo12','es','ad9da04fc689db0804c4e69bc0d0ec32a071d2c9637d3e57a70f81a4a7b1daff'),
 ('finalizacion/anexo11','en','f0220bce5622604530d173e48abe669fd51aa5c1e645e4ac093beda749663109'),
 ('finalizacion/anexo12','en','9aa8db70349bff16a8aa8c5d0a483e4a6d2845592feaddbb0ea1caa0ae7660ad')
)
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,
 rol_firmante,vigente_desde,comprobacion)
SELECT n.tipo,n.idioma,'1.0.2','plantillas/'||n.tipo||'/v1.0.2.docx',decode(n.huella,'hex'),
 p.rol_firmante,DATE '2026-10-14','motor_docx_1'
FROM nuevas n JOIN documento.plantilla p
 ON p.tipo_documento=n.tipo AND p.idioma=n.idioma AND p.version='1.0.0'
 AND p.ruta_relativa='plantillas/'||n.tipo||'/v1.0.0.docx' AND p.subida_por IS NULL
ON CONFLICT(tipo_documento,idioma,version) DO NOTHING;
DO $$ BEGIN
 IF (SELECT count(*) FROM documento.plantilla WHERE tipo_documento IN
 ('finalizacion/anexo11','finalizacion/anexo12') AND idioma IN ('es','en') AND version='1.0.2') <> 4
 THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;

-- Sólo avanzan las referencias del paquete inicial que nadie haya sustituido.
UPDATE documento.referencia_activa r SET plantilla_id=n.id,version=r.version+1
FROM documento.plantilla anterior,documento.plantilla n
WHERE anterior.id=r.plantilla_id AND anterior.subida_por IS NULL
 AND ((anterior.version='1.0.0' AND r.version=1) OR (anterior.version='1.0.1' AND r.version=2))
 AND anterior.ruta_relativa='plantillas/'||anterior.tipo_documento||'/v'||anterior.version||'.docx'
 AND r.tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12')
 AND n.tipo_documento=r.tipo_documento AND n.idioma=r.idioma AND n.version='1.0.2'
 AND n.ruta_relativa='plantillas/'||n.tipo_documento||'/v1.0.2.docx' AND n.subida_por IS NULL
 AND NOT EXISTS(SELECT 1 FROM documento.activacion_plantilla a
                WHERE a.tipo_documento=r.tipo_documento AND a.idioma=r.idioma)
 AND NOT EXISTS(SELECT 1 FROM documento.archivo_plantilla a
                JOIN documento.plantilla p ON p.id=a.anterior_id
                WHERE p.tipo_documento=r.tipo_documento AND p.idioma=r.idioma);
