SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.salida_servicio') IS NULL
 OR EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='expediente' AND table_name='salida_servicio' AND column_name='servicio_texto')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='expediente.salida_servicio'::regclass AND conname='salida_servicio_servicio_destino_check') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- El destino escrito es un dato del expediente; las etiquetas de catálogo
-- siguen en los paquetes de idioma. El destino se conserva al confirmar.
ALTER TABLE expediente.salida_servicio ADD COLUMN servicio_texto text NOT NULL DEFAULT '';
ALTER TABLE expediente.salida_servicio ALTER COLUMN catalogo_id DROP NOT NULL;
ALTER TABLE expediente.salida_servicio DROP CONSTRAINT salida_servicio_servicio_destino_check;
ALTER TABLE expediente.salida_servicio ADD CONSTRAINT salida_servicio_destino_check CHECK(
 (catalogo_id IS NOT NULL AND servicio_destino ~ '^[A-Z][A-Z0-9_]{0,63}$' AND servicio_texto='')
 OR (catalogo_id IS NULL AND servicio_destino='' AND length(btrim(servicio_texto)) BETWEEN 1 AND 254 AND octet_length(servicio_texto)<=1016)
);
-- No se amplían permisos de edición del destino ni políticas: siguen las de
-- profesional del equipo y Coordinación de expediente/0006. FORCE RLS sigue activo.
REVOKE UPDATE(servicio_texto) ON expediente.salida_servicio FROM etf_app;
