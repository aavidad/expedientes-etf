SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.smi') IS NULL OR to_regclass('catalogo.motivo_cierre') IS NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Importe inicial de 2026: datos del Programa, sin sustituir una edición.
-- La columna anual se calcula en la tabla con las catorce pagas existentes.
UPDATE catalogo.smi SET importe_mensual_centimos=122100
 WHERE anio=2026 AND version=0 AND importe_mensual_centimos IS NULL;

-- D5: nueva versión de las equivalencias iniciales. Se conservan los cierres
-- anteriores y cualquier publicación decidida por Administración.
DO $$ DECLARE anterior uuid; nueva uuid; BEGIN
 SELECT id INTO anterior FROM catalogo.version
 WHERE ambito='equivalencias' AND estado='publicada' AND numero=1
 AND motivo='cierre.d5' AND creada_por='00000000-0000-4000-8000-000000000001';
 IF anterior IS NOT NULL AND NOT EXISTS(SELECT 1 FROM catalogo.version WHERE ambito='equivalencias' AND numero>1) THEN
  INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
  VALUES('equivalencias',2,DATE '2026-10-14','cierre.d5_confirmada','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
  INSERT INTO catalogo.motivo_cierre(version_id,codigo,orden,acta,spaf,admite_texto,sin_consentimiento)
  SELECT nueva,codigo,orden,acta,
   CASE codigo WHEN 'DECLARACION_RIESGO' THEN 'DECLARACION_RIESGO' WHEN 'OTROS' THEN 'OTROS' ELSE spaf END,
   admite_texto,sin_consentimiento FROM catalogo.motivo_cierre WHERE version_id=anterior;
  PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
 END IF;
END $$;
