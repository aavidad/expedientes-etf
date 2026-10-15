SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.estado_cambio') IS NULL
 OR to_regclass('expediente.espera_memoria') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Proyección sin nombres, contactos ni contenido del anexo. El equipo es el
-- confirmado para la espera, no una asignación definitiva del expediente.
CREATE TABLE expediente.espera_memoria (
 expediente_id uuid PRIMARY KEY REFERENCES expediente.expediente(id),
 equipo_id uuid REFERENCES catalogo.equipo(id),
 tipo_anexo text NOT NULL,
 recepcion timestamptz,
 desde date CHECK(isfinite(desde)),
 hasta date CHECK(isfinite(hasta) AND hasta>=desde),
 registrado_en timestamptz,
 terminado_en timestamptz,
 CHECK((desde IS NULL)=(registrado_en IS NULL)),
 CHECK(desde IS NULL OR recepcion IS NOT NULL),
 CHECK(hasta IS NULL OR desde IS NOT NULL),
 CHECK((hasta IS NULL)=(terminado_en IS NULL)),
 CHECK(terminado_en IS NULL OR terminado_en>=registrado_en)
);
CREATE INDEX espera_memoria_equipo ON expediente.espera_memoria(equipo_id,desde);
ALTER TABLE expediente.espera_memoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.espera_memoria FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.espera_memoria FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.espera_memoria TO etf_app,etf_copia;
CREATE POLICY propietario ON expediente.espera_memoria TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY copia ON expediente.espera_memoria FOR SELECT TO etf_copia USING(true);
CREATE POLICY lectura ON expediente.espera_memoria FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 OR (desde IS NOT NULL AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));

-- Sólo durante esta migración se leen las derivaciones previas bajo RLS.
CREATE POLICY resp1_migrador ON expediente.expediente FOR SELECT TO etf_propietario USING(session_user='etf_migrador');
CREATE POLICY resp1_migrador ON expediente.derivacion_datos FOR SELECT TO etf_propietario USING(session_user='etf_migrador');
CREATE POLICY resp1_migrador ON expediente.estado_cambio FOR SELECT TO etf_propietario USING(session_user='etf_migrador');
-- Conserva los datos de ubicación confirmados sin releer el anexo desde un
-- trigger de propietario. Las filas sin fecha de espera no se sirven al equipo.
INSERT INTO expediente.espera_memoria(expediente_id,equipo_id,tipo_anexo)
SELECT e.id,CASE WHEN e.origen='traslado' THEN e.traslado_equipo_id
 ELSE nullif(d.contenido->'ubicacion_confirmada'->>'equipo_id','')::uuid END,
 coalesce(d.contenido->'contenido'->>'tipo','')
FROM expediente.expediente e JOIN expediente.derivacion_datos d ON d.expediente_id=e.id;
UPDATE expediente.espera_memoria m SET recepcion=e.fecha_recepcion,desde=e.fecha_lista_espera,
 hasta=CASE fin.estado_nuevo WHEN 'aceptada' THEN e.fecha_aceptacion
 WHEN 'no_aceptada' THEN (fin.cambiado_en AT TIME ZONE 'Europe/Madrid')::date END,
 registrado_en=inicio.cambiado_en,terminado_en=fin.cambiado_en
FROM expediente.expediente e
JOIN LATERAL(SELECT cambiado_en FROM expediente.estado_cambio c
 WHERE c.expediente_id=e.id AND c.estado_nuevo='lista_espera' ORDER BY cambiado_en,id LIMIT 1) inicio ON true
LEFT JOIN LATERAL(SELECT cambiado_en,estado_nuevo FROM expediente.estado_cambio c
 WHERE c.expediente_id=e.id AND c.estado_anterior='lista_espera'
 AND c.estado_nuevo IN ('aceptada','no_aceptada') ORDER BY cambiado_en,id LIMIT 1) fin ON true
WHERE m.expediente_id=e.id AND e.fecha_lista_espera IS NOT NULL AND m.tipo_anexo='anexo3';
DROP POLICY resp1_migrador ON expediente.estado_cambio;
DROP POLICY resp1_migrador ON expediente.derivacion_datos;
DROP POLICY resp1_migrador ON expediente.expediente;

-- Sólo se conserva tipo y equipo; las fechas se fijan al entrar en espera.
CREATE FUNCTION expediente.preparar_espera_memoria() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
BEGIN
 INSERT INTO expediente.espera_memoria(expediente_id,equipo_id,tipo_anexo)
 VALUES(NEW.expediente_id,nullif(NEW.contenido->'ubicacion_confirmada'->>'equipo_id','')::uuid,
 coalesce(NEW.contenido->'contenido'->>'tipo',''))
 ON CONFLICT(expediente_id) DO UPDATE SET equipo_id=EXCLUDED.equipo_id,tipo_anexo=EXCLUDED.tipo_anexo
 WHERE espera_memoria.desde IS NULL;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.preparar_espera_memoria() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER preparar_espera_memoria AFTER INSERT OR UPDATE OF contenido ON expediente.derivacion_datos
 FOR EACH ROW EXECUTE FUNCTION expediente.preparar_espera_memoria();

CREATE FUNCTION expediente.registrar_espera_memoria() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE tipo text; equipo uuid;
BEGIN
 IF NEW.estado=OLD.estado THEN RETURN NEW; END IF;
 IF NEW.estado='lista_espera' THEN
  IF nullif(current_setting('app.usuario_id',true),'') IS NULL
   OR NOT 'coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado';
  END IF;
  SELECT tipo_anexo,equipo_id INTO tipo,equipo FROM expediente.espera_memoria WHERE expediente_id=NEW.id;
  IF tipo IS DISTINCT FROM 'anexo3' THEN RAISE EXCEPTION 'expediente.espera_solo_comunitarios'; END IF;
  IF NEW.origen='traslado' THEN equipo:=NEW.traslado_equipo_id; END IF;
  UPDATE expediente.espera_memoria SET equipo_id=equipo,recepcion=NEW.fecha_recepcion,
   desde=NEW.fecha_lista_espera,registrado_en=NEW.modificado_en WHERE expediente_id=NEW.id AND desde IS NULL;
 ELSIF OLD.estado='lista_espera' AND NEW.estado IN ('aceptada','no_aceptada') THEN
  UPDATE expediente.espera_memoria SET
   hasta=CASE NEW.estado WHEN 'aceptada' THEN NEW.fecha_aceptacion ELSE (NEW.modificado_en AT TIME ZONE 'Europe/Madrid')::date END,
   terminado_en=NEW.modificado_en WHERE expediente_id=NEW.id AND hasta IS NULL;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.registrar_espera_memoria() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER registrar_espera_memoria AFTER UPDATE OF estado ON expediente.expediente
 FOR EACH ROW EXECUTE FUNCTION expediente.registrar_espera_memoria();
