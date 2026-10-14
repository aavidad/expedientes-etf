SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.hito') IS NULL
 OR EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='expediente'
  AND table_name='expediente' AND column_name='fecha_no_aceptacion') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

ALTER TABLE expediente.expediente
 ADD COLUMN fecha_no_aceptacion date CHECK (isfinite(fecha_no_aceptacion)),
 ADD COLUMN causa_no_aceptacion text CHECK (length(btrim(causa_no_aceptacion))>0 AND octet_length(causa_no_aceptacion)<=20000),
 ADD COLUMN fecha_lista_espera date CHECK (isfinite(fecha_lista_espera)),
 DROP CONSTRAINT expediente_estado_check,
 DROP CONSTRAINT expediente_aceptacion_completa,
 ADD CONSTRAINT expediente_estado_check CHECK (estado IN ('derivacion_recibida','lista_espera','no_aceptada','aceptada')),
 ADD CONSTRAINT expediente_aceptacion_completa CHECK (
  (estado IN ('derivacion_recibida','lista_espera','no_aceptada') AND numero IS NULL AND equipo_id IS NULL AND municipio_id IS NULL
   AND fecha_aceptacion IS NULL AND anio_numero IS NULL AND correlativo IS NULL AND regla_numeracion_id IS NULL AND centro_ss_id IS NULL
   AND profesional_referencia_puesto IS NULL AND profesional_referencia_id IS NULL AND descripcion_situacion IS NULL AND fase_intervencion IS NULL)
  OR (estado='aceptada' AND numero IS NOT NULL AND equipo_id IS NOT NULL AND municipio_id IS NOT NULL AND centro_ss_id IS NOT NULL
   AND fecha_aceptacion IS NOT NULL AND anio_numero IS NOT NULL AND anio_numero=extract(year FROM fecha_aceptacion)
   AND correlativo IS NOT NULL AND regla_numeracion_id IS NOT NULL AND subprograma IS NOT NULL
   AND profesional_referencia_puesto IS NOT NULL AND descripcion_situacion IS NOT NULL AND fase_intervencion IS NOT NULL
   AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
 ),
 ADD CONSTRAINT expediente_decision_completa CHECK (
  (estado='no_aceptada' AND fecha_no_aceptacion IS NOT NULL AND causa_no_aceptacion IS NOT NULL AND subprograma IS NOT NULL
   AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
  OR (estado<>'no_aceptada' AND fecha_no_aceptacion IS NULL AND causa_no_aceptacion IS NULL
   AND (estado='aceptada' OR subprograma IS NULL))
 ),
 ADD CONSTRAINT expediente_espera_completa CHECK (
  estado<>'lista_espera' OR (fecha_lista_espera IS NOT NULL AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
 );
CREATE INDEX expediente_lista_espera ON expediente.expediente(fecha_recepcion,id) WHERE estado='lista_espera';
GRANT UPDATE(fecha_no_aceptacion,causa_no_aceptacion,fecha_lista_espera) ON expediente.expediente TO etf_app;

-- Antes de aceptar no existe equipo ni episodio. Los hitos siguen separados de las emisiones.
ALTER TABLE expediente.hito
 ALTER COLUMN equipo_id DROP NOT NULL, ALTER COLUMN episodio_id DROP NOT NULL,
 DROP CONSTRAINT hito_tipo_check,
 ADD CONSTRAINT hito_tipo_check CHECK (tipo IN ('aceptacion','no_aceptacion','lista_espera')),
 ADD CONSTRAINT hito_derivacion FOREIGN KEY(expediente_id) REFERENCES expediente.expediente(id),
 ADD CONSTRAINT hito_asignacion CHECK (
  (tipo='aceptacion' AND equipo_id IS NOT NULL AND episodio_id IS NOT NULL)
  OR (tipo IN ('no_aceptacion','lista_espera') AND equipo_id IS NULL AND episodio_id IS NULL)
 );
ALTER TABLE expediente.estado_cambio
 ALTER COLUMN equipo_id DROP NOT NULL,
 DROP CONSTRAINT estado_cambio_estado_anterior_check,
 DROP CONSTRAINT estado_cambio_estado_nuevo_check,
 ADD CONSTRAINT estado_cambio_derivacion FOREIGN KEY(expediente_id) REFERENCES expediente.expediente(id),
 ADD CONSTRAINT estado_cambio_transicion CHECK (
  (estado_anterior='derivacion_recibida' AND estado_nuevo IN ('lista_espera','no_aceptada','aceptada'))
  OR (estado_anterior='lista_espera' AND estado_nuevo IN ('no_aceptada','aceptada'))
 ),
 ADD CONSTRAINT estado_cambio_asignacion CHECK ((estado_nuevo='aceptada' AND equipo_id IS NOT NULL)
  OR (estado_nuevo IN ('lista_espera','no_aceptada') AND equipo_id IS NULL));

-- Segunda barrera: una no aceptación es final y sólo Anexo 3 puede esperar.
CREATE FUNCTION expediente.proteger_decision() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 IF TG_OP='UPDATE' AND OLD.estado='no_aceptada' THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 IF NEW.estado='lista_espera' OR NEW.estado='no_aceptada' THEN
  IF NOT EXISTS(SELECT 1 FROM expediente.derivacion_datos d WHERE d.expediente_id=NEW.id
   AND ((NEW.estado='lista_espera' AND d.contenido->'contenido'->>'tipo'='anexo3')
    OR (NEW.estado='no_aceptada' AND ((d.contenido->'contenido'->>'tipo'='anexo3' AND NEW.subprograma='RIESGO_SOCIAL')
     OR d.contenido->'contenido'->>'tipo'='anexo4')))) THEN
   RAISE EXCEPTION 'expediente.decision_origen_invalido';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_decision() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_decision BEFORE INSERT OR UPDATE ON expediente.expediente
 FOR EACH ROW EXECUTE FUNCTION expediente.proteger_decision();
