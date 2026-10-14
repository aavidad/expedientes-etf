SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.dato_estadistico_vigencia') IS NULL
 OR to_regclass('expediente.api_estadistica_expedientes') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Proyecciones del módulo dueño. El invocador conserva la RLS de cada tabla;
-- no se publican nombres, narrativas ni el JSON de los anexos.
CREATE VIEW expediente.api_estadistica_expedientes WITH (security_invoker=true,security_barrier=true) AS
 SELECT e.id,e.equipo_id,e.numero,e.fecha_recepcion,e.municipio_id,
 CASE d.contenido->'contenido'->>'tipo' WHEN 'anexo3' THEN 'SSSSCC' WHEN 'anexo4' THEN 'SPM' ELSE '' END AS servicio
 FROM expediente.expediente e LEFT JOIN expediente.derivacion_datos d ON d.expediente_id=e.id
 WHERE e.equipo_id IS NOT NULL;
CREATE VIEW expediente.api_estadistica_episodios WITH (security_invoker=true,security_barrier=true) AS
 SELECT p.id,p.expediente_id,p.equipo_id,p.apertura_fecha,p.creado_en,
 to_jsonb(p)->>'reapertura_de_id' AS reapertura_de_id,
 (to_jsonb(p)->>'cierre_fecha')::date AS cierre_fecha,
 to_jsonb(p)->>'motivo_cierre' AS motivo_cierre,
 (to_jsonb(p)->>'cerrado_en')::timestamptz AS cerrado_en
 FROM expediente.episodio p;
CREATE VIEW expediente.api_estadistica_hitos WITH (security_invoker=true,security_barrier=true) AS
 SELECT id,expediente_id,equipo_id,episodio_id,tipo,fecha,confirmado_en FROM expediente.hito;
CREATE VIEW expediente.api_estadistica_estados WITH (security_invoker=true,security_barrier=true) AS
 SELECT id,expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_en FROM expediente.estado_cambio;
CREATE VIEW expediente.api_estadistica_vigencias WITH (security_invoker=true,security_barrier=true) AS
 SELECT id,expediente_id,equipo_id,atributo,valor,vigente_desde,registrado_en
 FROM expediente.dato_estadistico_vigencia WHERE entidad_tipo='expediente' AND atributo='subprograma';
REVOKE ALL ON expediente.api_estadistica_expedientes,expediente.api_estadistica_episodios,
 expediente.api_estadistica_hitos,expediente.api_estadistica_estados,expediente.api_estadistica_vigencias FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.api_estadistica_expedientes,expediente.api_estadistica_episodios,
 expediente.api_estadistica_hitos,expediente.api_estadistica_estados,expediente.api_estadistica_vigencias TO etf_app,etf_copia;
CREATE INDEX estadistica_episodio_conocido ON expediente.episodio(expediente_id,creado_en);
CREATE INDEX estadistica_hito_conocido ON expediente.hito(expediente_id,confirmado_en);
CREATE INDEX estadistica_vigencia_conocida ON expediente.dato_estadistico_vigencia(expediente_id,registrado_en,vigente_desde);

-- Las revisiones de familia ya excluyen contactos y narrativas. Esta proyección
-- enumera las dimensiones para impedir que una ampliación exponga más campos.
CREATE VIEW expediente.api_estadistica_familias WITH(security_invoker=true,security_barrier=true) AS
 SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,
 jsonb_build_object('composicion_familiar',valor->'composicion_familiar',
 'familia_numerosa',valor->'familia_numerosa','situacion_economica',valor->'situacion_economica',
 'ingresos_anuales_centimos',valor->'ingresos_anuales_centimos','tramo_smi',valor->'tramo_smi') AS valor
 FROM expediente.familia_revision;
CREATE VIEW expediente.api_estadistica_adultos WITH(security_invoker=true,security_barrier=true) AS
 SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 coalesce(jsonb_agg(jsonb_build_object('papel',a.papel,'sexo',a.sexo,
 'fecha_nacimiento',a.fecha_nacimiento,'nivel_educativo',a.nivel_educativo,
 'situacion_laboral',a.situacion_laboral) ORDER BY a.orden) FILTER(WHERE a.revision_id IS NOT NULL),'[]') AS valor
 FROM expediente.adultos_revision r LEFT JOIN expediente.adulto_revision a ON a.revision_id=r.id
 GROUP BY r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en;
REVOKE ALL ON expediente.api_estadistica_familias,expediente.api_estadistica_adultos FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.api_estadistica_familias,expediente.api_estadistica_adultos TO etf_app,etf_copia;
