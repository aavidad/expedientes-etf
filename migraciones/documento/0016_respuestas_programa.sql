SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.activacion_plantilla') IS NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Las plantillas y emisiones anteriores conservan sus bytes y referencias.
-- Una publicación administrativa con el mismo número nunca se sustituye.
CREATE TEMP TABLE respuestas_plantillas(tipo text,idioma text,version text,previa text,huella text) ON COMMIT DROP;
INSERT INTO respuestas_plantillas VALUES
 ('cierre/acta_cierre','es','1.0.2','1.0.1','f3e4dcacd5f71d6787445749041539ec658ae479ae8b5039748777d3130ff210'),
 ('entrada/aceptacion','es','1.1.2','1.1.1','14d720c9d14ae4eb035d149064cdd1bf165c2ae61fe6a4567d84d427e3944fd2'),
 ('entrada/no_aceptacion','es','1.0.2','1.0.1','d42fa1698b175dfd3071808cebc7901f4eec0d10fc07edf0fe57b81576e6e30f'),
 ('entrada/no_aceptacion/lista_espera','es','1.0.2','1.0.1','a82aba9d8d7e05fa9d0b3bad62cf3f1acc23e43ae74450f6b99149c0f6b8b9a6'),
 ('spaf','es','1.0.2','1.0.1','469ec95d495277329b149343f87c0cc34b9a20798e1e0048d409aca8d2f78635'),
 ('cierre/acta_cierre','en','1.0.2','1.0.1','62e06b88b18065352f820b79e46f8c260b1c958530953a7096b1ace712d11932'),
 ('entrada/aceptacion','en','1.1.2','1.1.1','8a5099189c5e51878530502cd7fa798a8284b71a5d4bb96a03e6089a91198bce'),
 ('entrada/no_aceptacion','en','1.0.2','1.0.1','0a175698e1075287de010dd73d7f59c770cca39a4f9c26592634b862934c2a04'),
 ('entrada/no_aceptacion/lista_espera','en','1.0.2','1.0.1','ce2ba15ac5e0de393f59fb0a37dd7fd2764db45f2947c70f43e1d3a8fd5af09b'),
 ('spaf','en','1.0.2','1.0.1','1a0e25233dfd2402454e52724a1d6c6bfea2a5c68a05f52ca3a18faec3df535e');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,
 rol_firmante,vigente_desde,comprobacion)
SELECT c.tipo,c.idioma,c.version,'plantillas/'||c.tipo||'/v'||c.version||'.docx',
 decode(c.huella,'hex'),p.rol_firmante,DATE '2026-10-14','motor_docx_1'
FROM respuestas_plantillas c JOIN documento.plantilla p
 ON p.tipo_documento=c.tipo AND p.idioma=c.idioma AND p.version=c.previa
 AND p.ruta_relativa='plantillas/'||c.tipo||'/v'||c.previa||'.docx' AND p.subida_por IS NULL
ON CONFLICT(tipo_documento,idioma,version) DO NOTHING;

-- Sólo se actualizan referencias intactas del paquete inicial; se respetan
-- publicaciones, activaciones y restauraciones de Administración.
UPDATE documento.referencia_activa r SET plantilla_id=n.id,version=r.version+1
FROM respuestas_plantillas c,documento.plantilla anterior,documento.plantilla n
WHERE anterior.id=r.plantilla_id AND anterior.subida_por IS NULL
 AND anterior.tipo_documento=c.tipo AND anterior.idioma=c.idioma AND anterior.version=c.previa
 AND anterior.ruta_relativa='plantillas/'||c.tipo||'/v'||c.previa||'.docx'
 AND r.version=CASE WHEN r.tipo_documento='entrada/aceptacion' THEN 3 ELSE 2 END
 AND n.tipo_documento=c.tipo AND n.idioma=c.idioma AND n.version=c.version
 AND n.ruta_relativa='plantillas/'||c.tipo||'/v'||c.version||'.docx'
 AND n.huella_sha256=decode(c.huella,'hex') AND n.subida_por IS NULL
 AND NOT EXISTS(SELECT 1 FROM documento.activacion_plantilla a WHERE a.tipo_documento=c.tipo AND a.idioma=c.idioma)
 AND NOT EXISTS(SELECT 1 FROM documento.archivo_plantilla a JOIN documento.plantilla p ON p.id=a.anterior_id
                WHERE p.tipo_documento=c.tipo AND p.idioma=c.idioma);
