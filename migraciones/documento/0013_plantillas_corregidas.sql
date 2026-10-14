SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('documento.activacion_plantilla') IS NULL
 THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Versiones corregidas del paquete inicial. Los originales conservan sus bytes,
-- metadatos y referencias de emisiones. No se declara un antivirus no ejecutado.
-- Una versión con el mismo número ya publicada por Administración se conserva.
WITH corregidas(tipo,idioma,version,previa,huella) AS (VALUES
 ('cierre/acta_cierre','en','1.0.1','1.0.0','abbed68f592a72649d948d5e018992b132947e19c67abcbae7393d7968834924'),
 ('consentimiento/anexo1','en','1.0.1','1.0.0','99668917d811e75edc86d9aca52eac4deb9d2caeb5aae90875518e8f886c4d9a'),
 ('consentimiento/anexo2','en','1.0.1','1.0.0','5c5fde819e088e22c189bac2db5390eaa48ee28e74cf5268fd24603a5eb8cef2'),
 ('entrada/aceptacion/oficio_spm','en','1.0.1','1.0.0','6bb97c3e1030baa1fa527c98f2b7b316bd2fd2980e7390b6e8a394890274348f'),
 ('entrada/aceptacion/oficio_sscc','en','1.0.1','1.0.0','2cd1b320ccc8b44783fe4bc575ae243a2750125566440a521baee240d35ce8ab'),
 ('entrada/aceptacion','en','1.0.1','1.0.0','14aff4dca73e30ddc00a5f2de3c0559cf2dd233d25fb908a99ecd37d82c8ebf5'),
 ('entrada/aceptacion','en','1.1.1','1.1.0','d8e7ded0107eded691985a2d35c03140c2844b0dd0c00d2bb53e7196059ebc36'),
 ('entrada/no_aceptacion/lista_espera','en','1.0.1','1.0.0','fd4d21103e2e2120e236710141e2c8debf3c9eba598a8a48707927912b6a329b'),
 ('entrada/no_aceptacion','en','1.0.1','1.0.0','5786fc412579bcf88c047345f513e379acc8ef64e9445276a509f4b84c6e73d8'),
 ('finalizacion/anexo11','en','1.0.1','1.0.0','620981be34eeed737b935dc42dc7d226aaab639c95e5c5bcbd6d69d65db569d1'),
 ('finalizacion/anexo12','en','1.0.1','1.0.0','9e1ad958c1e30bed8329d9e6bc11f8fbab01574c40c117583df19a017a8b0d10'),
 ('salida/anexo6','en','1.0.1','1.0.0','20bbcf2caa6ab35e21b18fd0e3e40b1f5f45c32e46fecb2939264b2a3ca874d1'),
 ('salida/spm/anexo5','en','1.0.1','1.0.0','9e1f466f921cfa6ebd69c28a6d58af2c96162c10481f6f6e059892220a3ce9ad'),
 ('spaf','en','1.0.1','1.0.0','034e7caee60a332dacfc98d3c5fa66a181395246a8923fd889ea424bff4f4725'),
 ('cierre/acta_cierre','es','1.0.1','1.0.0','aa712fde5e82005a5a890a59a1a627a8c3c35806a98a576a49a43e2e53d91338'),
 ('consentimiento/anexo1','es','1.0.1','1.0.0','eff5708adada86d3fdd5bb9be77adbc8da76283959480b0a4c0a303fd6b10ba1'),
 ('consentimiento/anexo2','es','1.0.1','1.0.0','7c6c72cc144996531dd7afaa2deab36666831be02cf242aeacd171a84d5ff226'),
 ('entrada/aceptacion/oficio_spm','es','1.0.1','1.0.0','c24748cec51059ad883a84ef99615182fc623e46b5518657a9d12adc22b77ac5'),
 ('entrada/aceptacion/oficio_sscc','es','1.0.1','1.0.0','caefc8e0b24ddb37e0f879db3fd89d9cc347152e51796c9f5644d6ba12a07996'),
 ('entrada/aceptacion','es','1.0.1','1.0.0','3e6b0ea51666dbfb0129b30baf43dc669a2c08e195288be8fbe30d9b45bbe3fd'),
 ('entrada/aceptacion','es','1.1.1','1.1.0','af2cbfce796d9c904aef7a637383ea3ab704c511132109f3cf5d6ecc709857ea'),
 ('entrada/no_aceptacion/lista_espera','es','1.0.1','1.0.0','871afaa3a2560af7467a3382fc3207726dedb440ed5eafb2dc4b85fc2bb75239'),
 ('entrada/no_aceptacion','es','1.0.1','1.0.0','6a02eabab7aa8b07e1b088d89692f8592aca81eecc738a75768b7049f81bbcea'),
 ('finalizacion/anexo11','es','1.0.1','1.0.0','6aac0902264618c4e83c77a0c48a381873eb2b6685d8bd6f819f9ceabdb9fe39'),
 ('finalizacion/anexo12','es','1.0.1','1.0.0','579d5be159b625f3999b8d1d72573dbddc3f3ee9dafd02d1341033ce5ca4f420'),
 ('salida/anexo6','es','1.0.1','1.0.0','4619f9e4bf6a137558d077aa4960d463e6fbcd1913bb76a1956da559c78a1735'),
 ('salida/spm/anexo5','es','1.0.1','1.0.0','31afd6e2bc49703d55a016c70e69bfc4818e404760da89b517928353dbe73209'),
 ('spaf','es','1.0.1','1.0.0','d626067e89447a0f0ad643f24861c13277c17c941996a7919f19ef0ca08bea10')
)
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,
 rol_firmante,vigente_desde,comprobacion)
SELECT c.tipo,c.idioma,c.version,'plantillas/'||c.tipo||'/v'||c.version||'.docx',
 decode(c.huella,'hex'),p.rol_firmante,DATE '2026-10-14','motor_docx_1'
FROM corregidas c JOIN documento.plantilla p
 ON p.tipo_documento=c.tipo AND p.idioma=c.idioma AND p.version=c.previa
 AND p.ruta_relativa='plantillas/'||c.tipo||'/v'||c.previa||'.docx'
 AND p.subida_por IS NULL
ON CONFLICT(tipo_documento,idioma,version) DO NOTHING;
DO $$ BEGIN
 IF (SELECT count(*) FROM documento.plantilla p
     JOIN documento.plantilla n ON n.tipo_documento=p.tipo_documento AND n.idioma=p.idioma
      AND n.version=CASE WHEN p.version='1.1.0' THEN '1.1.1' ELSE '1.0.1' END
     WHERE p.version IN ('1.0.0','1.1.0') AND p.subida_por IS NULL
      AND p.idioma IN ('es','en')
      AND p.ruta_relativa='plantillas/'||p.tipo_documento||'/v'||p.version||'.docx') <> 28 THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Sólo se sustituyen referencias del arranque. Una publicación, activación o
-- restauración administrativa conserva la decisión y el historial de su autor.
UPDATE documento.referencia_activa r SET plantilla_id=n.id,version=r.version+1
FROM documento.plantilla anterior,documento.plantilla n
WHERE anterior.id=r.plantilla_id AND anterior.subida_por IS NULL
 AND anterior.ruta_relativa='plantillas/'||anterior.tipo_documento||'/v'||anterior.version||'.docx'
 AND anterior.version IN ('1.0.0','1.1.0')
 AND r.version=CASE WHEN r.tipo_documento='entrada/aceptacion' THEN 2 ELSE 1 END
 AND n.tipo_documento=r.tipo_documento AND n.idioma=r.idioma
 AND n.version=CASE WHEN anterior.version='1.1.0' THEN '1.1.1' ELSE '1.0.1' END
 AND n.ruta_relativa='plantillas/'||n.tipo_documento||'/v'||n.version||'.docx'
 AND NOT EXISTS(SELECT 1 FROM documento.activacion_plantilla a
                WHERE a.tipo_documento=r.tipo_documento AND a.idioma=r.idioma)
 AND NOT EXISTS(SELECT 1 FROM documento.archivo_plantilla a
                JOIN documento.plantilla p ON p.id=a.anterior_id
                WHERE p.tipo_documento=r.tipo_documento AND p.idioma=r.idioma);
