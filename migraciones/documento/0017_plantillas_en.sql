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
CREATE TEMP TABLE traducciones_plantillas(tipo text,idioma text,version text,previa text,huella text,revision_previa integer) ON COMMIT DROP;
INSERT INTO traducciones_plantillas VALUES
 ('consentimiento/anexo1','en','1.0.2','1.0.1','872df5489ddd2f0a31bd8bc9189ac58f078ed1dbbb65cbe7e19071b7b40e68d4',2),
 ('consentimiento/anexo2','en','1.0.2','1.0.1','f229ee84ecbe853ac8bb7748baee2b931c6f68b11a07a7f926de6f9f76935075',2),
 ('entrada/aceptacion','en','1.1.3','1.1.2','195885257f871009dc52df1f0eff7aee73e0950adfb0c526e171b35d5c329bed',4),
 ('entrada/aceptacion/oficio_spm','en','1.0.2','1.0.1','8a7425114c0ee28e978b2eba0a47860b6f08b2b29ffea6047a4657a81dae39ea',2),
 ('entrada/aceptacion/oficio_sscc','en','1.0.2','1.0.1','bd0c5e0140469db4c22087a36a0f7d354de6d6bf3542fbb13d6adf9197114a22',2),
 ('entrada/no_aceptacion','en','1.0.3','1.0.2','e38028ba097d67ed4e71b8245815f2175e0d54f9ee200359e7d6442896365eb1',3),
 ('entrada/no_aceptacion/lista_espera','en','1.0.3','1.0.2','038af7a949eeb4d8be85aa299470d63287b52d06f75bf11171103fdaf24b2695',3),
 ('finalizacion/anexo11','en','1.0.3','1.0.2','79cbbabc0011a471395c6913a07235ac2528aa199f7d49a7cf0fb2a7c64274b1',3),
 ('finalizacion/anexo12','en','1.0.3','1.0.2','72256cea9f19fd16ad4324cfb443d924ad6e4b96229fba818414275562cd3326',3),
 ('salida/anexo6','en','1.0.2','1.0.1','cfbce655a9415d15e6836ecd8e312d36951d5981166b4355b341f1e22e7dddb6',2),
 ('salida/spm/anexo5','en','1.0.2','1.0.1','6d0dc3aaa2209b0acb1102be73fb4fa8272ea89ca181be6f2c76c0cb253fa339',2),
 ('spaf','en','1.0.3','1.0.2','bdc799c509a5415990e6420217d6b2faf8726b9b0df97c7e454cab637e493e1d',3);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,
 rol_firmante,vigente_desde,comprobacion)
SELECT c.tipo,c.idioma,c.version,'plantillas/'||c.tipo||'/v'||c.version||'.docx',
 decode(c.huella,'hex'),p.rol_firmante,DATE '2026-10-15','motor_docx_1'
FROM traducciones_plantillas c JOIN documento.plantilla p
 ON p.tipo_documento=c.tipo AND p.idioma=c.idioma AND p.version=c.previa
 AND p.ruta_relativa='plantillas/'||c.tipo||'/v'||c.previa||'.docx' AND p.subida_por IS NULL
ON CONFLICT(tipo_documento,idioma,version) DO NOTHING;

-- Sólo se actualizan referencias intactas del paquete inicial; se respetan
-- publicaciones, activaciones y restauraciones de Administración.
UPDATE documento.referencia_activa r SET plantilla_id=n.id,version=r.version+1
FROM traducciones_plantillas c,documento.plantilla anterior,documento.plantilla n
WHERE anterior.id=r.plantilla_id AND anterior.subida_por IS NULL
 AND anterior.tipo_documento=c.tipo AND anterior.idioma=c.idioma AND anterior.version=c.previa
 AND anterior.ruta_relativa='plantillas/'||c.tipo||'/v'||c.previa||'.docx'
 AND r.version=c.revision_previa
 AND n.tipo_documento=c.tipo AND n.idioma=c.idioma AND n.version=c.version
 AND n.ruta_relativa='plantillas/'||c.tipo||'/v'||c.version||'.docx'
 AND n.huella_sha256=decode(c.huella,'hex') AND n.subida_por IS NULL
 AND NOT EXISTS(SELECT 1 FROM documento.activacion_plantilla a WHERE a.tipo_documento=c.tipo AND a.idioma=c.idioma)
 AND NOT EXISTS(SELECT 1 FROM documento.archivo_plantilla a JOIN documento.plantilla p ON p.id=a.anterior_id
                WHERE p.tipo_documento=c.tipo AND p.idioma=c.idioma);
