SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.carga_inicial') IS NULL
 OR EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='expediente'
  AND table_name='carga_inicial' AND column_name='huella_fuente') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- La reserva carga/0001 amplía la procedencia del lote del módulo expediente.
-- No crea otro propietario de los datos ni almacena la hoja o su nombre.
ALTER TABLE expediente.carga_inicial
 ADD COLUMN huella_fuente bytea,
 ADD COLUMN formato_fuente text,
 ADD COLUMN bytes_fuente integer,
 ADD COLUMN antivirus text,
 ADD CONSTRAINT carga_hoja_procedencia CHECK (
  (origen='manual' AND huella_fuente IS NULL AND formato_fuente IS NULL
   AND bytes_fuente IS NULL AND antivirus IS NULL)
  OR (origen='hoja' AND huella_fuente IS NOT NULL AND octet_length(huella_fuente)=32
   AND formato_fuente IS NOT NULL AND formato_fuente IN ('csv','xlsx')
   AND bytes_fuente IS NOT NULL AND bytes_fuente BETWEEN 1 AND 10000000
   AND antivirus IS NOT NULL AND antivirus='limpio'));

-- RLS y permisos de lectura de expediente/0018 se conservan. Sólo se amplía
-- INSERT; la procedencia confirmada no admite UPDATE, DELETE ni TRUNCATE.
GRANT INSERT(origen,huella_fuente,formato_fuente,bytes_fuente,antivirus)
 ON expediente.carga_inicial TO etf_app;
