SET LOCAL lock_timeout = '3s';
SET LOCAL statement_timeout = '30s';
SET LOCAL search_path = pg_catalog, catalogo;
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('catalogo.smi') IS NULL OR to_regclass('catalogo.motivo_cierre') IS NOT NULL
 OR EXISTS (SELECT 1 FROM catalogo.version WHERE ambito = 'equivalencias') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

CREATE TABLE catalogo.motivo_cierre (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 codigo text NOT NULL CHECK (codigo ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 orden integer NOT NULL CHECK (orden >= 0),
 acta text NOT NULL CHECK (acta ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 spaf text NOT NULL CHECK (spaf ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 admite_texto boolean NOT NULL DEFAULT false,
 sin_consentimiento boolean NOT NULL DEFAULT false,
 PRIMARY KEY (version_id, codigo),
 UNIQUE (version_id, orden)
);
ALTER TABLE catalogo.motivo_cierre ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.motivo_cierre FORCE ROW LEVEL SECURITY;
REVOKE ALL ON catalogo.motivo_cierre FROM PUBLIC, etf_app, etf_copia;
GRANT SELECT ON catalogo.motivo_cierre TO etf_app, etf_copia;
CREATE POLICY propietario ON catalogo.motivo_cierre TO etf_propietario
 USING (true) WITH CHECK (true);
CREATE POLICY lectura_cierre ON catalogo.motivo_cierre FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id', true), '') IS NOT NULL
 AND string_to_array(current_setting('app.perfiles', true), ',') && ARRAY['coordinacion','profesional_etf']
 AND EXISTS (SELECT 1 FROM catalogo.version v WHERE v.id = version_id
             AND v.ambito = 'equivalencias' AND v.estado = 'publicada')
);
-- Los lectores de documentos usan el perfil vigente, «profesional_etf».
-- Las versiones anteriores sólo reconocían el perfil «equipo».
CREATE POLICY lectura_version_cierre ON catalogo.version FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id', true), '') IS NOT NULL
 AND 'profesional_etf' = ANY(string_to_array(current_setting('app.perfiles', true), ','))
 AND estado = 'publicada'
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['equipo','centro_ss','municipio','firmante','firmante_textos'] LOOP
  EXECUTE format('CREATE POLICY lectura_profesional_cierre ON catalogo.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'', true), '''') IS NOT NULL
   AND ''profesional_etf'' = ANY(string_to_array(current_setting(''app.perfiles'', true), '','')))', tabla);
 END LOOP;
END $$;
CREATE POLICY lectura_asignacion_cierre ON catalogo.municipio_asignacion FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id', true), '') IS NOT NULL
 AND 'profesional_etf' = ANY(string_to_array(current_setting('app.perfiles', true), ','))
 AND EXISTS (SELECT 1 FROM catalogo.version v WHERE v.id = version_id AND v.estado = 'publicada')
);
CREATE TRIGGER proteger_motivo_cierre BEFORE INSERT OR UPDATE OR DELETE ON catalogo.motivo_cierre
 FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('equivalencias');
CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.motivo_cierre
 FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();

-- Equivalencias provisionales D5; sólo OTROS admite explicación libre.
WITH nueva AS (
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES ('equivalencias',1,'2026-01-01','cierre.d5','00000000-0000-4000-8000-000000000001') RETURNING id
)
INSERT INTO catalogo.motivo_cierre(version_id,codigo,orden,acta,spaf,admite_texto,sin_consentimiento)
 SELECT nueva.id, datos.* FROM nueva CROSS JOIN (VALUES
 ('OBJ_TOTAL',0,'OBJ_TOTAL','OBJETIVOS',false,false),
 ('OBJ_PARCIAL',1,'OBJ_PARCIAL','OBJETIVOS',false,false),
 ('SIN_ADHERENCIA',2,'OTROS','NO_COLABORACION',false,false),
 ('ABANDONO',3,'NO_ACEPTACION','NO_COLABORACION',false,false),
 ('TRASLADO',4,'OTROS','TRASLADO',false,false),
 ('MAYORIA_EDAD',5,'OTROS','MAYORIA_EDAD',false,false),
 ('INGRESO_CENTRO',6,'OTROS','MEDIDA_PROTECTORA',false,false),
 ('DECLARACION_RIESGO',7,'OTROS','NO_CONSECUCION',false,false),
 ('MEDIDA_PROTECTORA',8,'OTROS','MEDIDA_PROTECTORA',false,false),
 ('OTROS',9,'OTROS','NO_CONSECUCION',true,false),
 ('SIN_CONSENTIMIENTO',10,'NO_ACEPTACION','NO_COLABORACION',false,true)
 ) AS datos(codigo,orden,acta,spaf,admite_texto,sin_consentimiento);
SELECT catalogo.publicar(id,'00000000-0000-4000-8000-000000000001')
 FROM catalogo.version WHERE ambito = 'equivalencias';
