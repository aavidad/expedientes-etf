SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

-- Comprueba la restricción original antes de ampliarla. La migración aplicada
-- permanece intacta y el migrador envuelve todo el cambio en una transacción.
DO $$
BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.lectura_anexo') IS NULL
 OR NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_constraint
  WHERE conrelid = 'expediente.lectura_anexo'::regclass
   AND conname = 'lectura_anexo_formato_check' AND contype = 'c' AND convalidated
   AND pg_catalog.pg_get_constraintdef(oid) =
    'CHECK ((formato = ANY (ARRAY[''docx''::text, ''pdf''::text])))'
 ) THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

ALTER TABLE expediente.lectura_anexo DROP CONSTRAINT lectura_anexo_formato_check;
ALTER TABLE expediente.lectura_anexo ADD CONSTRAINT lectura_anexo_formato_check
 CHECK (formato IN ('docx', 'odt', 'pdf'));
