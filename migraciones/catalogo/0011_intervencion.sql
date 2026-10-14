SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.intervencion_opcion') IS NOT NULL
 OR to_regclass('catalogo.version_textos') IS NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
ALTER TABLE catalogo.version DROP CONSTRAINT version_ambito_check;
ALTER TABLE catalogo.version ADD CONSTRAINT version_ambito_check CHECK(ambito IN ('listas','equivalencias','plazos','tramos','numeracion','escalas','asignacion_municipios','tipos_documento','intervencion'));
CREATE TABLE catalogo.intervencion_opcion (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 lista text NOT NULL CHECK(lista IN ('fase_intervencion','coordinacion','factor_familia','factor_menor','tipologia')),
 codigo text NOT NULL CHECK(codigo ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 orden integer NOT NULL CHECK(orden>=0),admite_texto boolean NOT NULL,
 archivo_textos text NOT NULL DEFAULT 'catalogos/intervencion-v1.json',
 PRIMARY KEY(version_id,lista,codigo),UNIQUE(version_id,lista,orden)
);
ALTER TABLE catalogo.intervencion_opcion ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.intervencion_opcion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON catalogo.intervencion_opcion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON catalogo.intervencion_opcion TO etf_app,etf_copia;
CREATE POLICY propietario ON catalogo.intervencion_opcion TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY lectura ON catalogo.intervencion_opcion FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['profesional_etf','coordinacion']
 AND EXISTS(SELECT 1 FROM catalogo.version v WHERE v.id=version_id AND v.ambito='intervencion' AND v.estado='publicada'));
CREATE TRIGGER proteger_intervencion BEFORE INSERT OR UPDATE OR DELETE ON catalogo.intervencion_opcion
 FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('intervencion');
CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.intervencion_opcion
 FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();
-- Códigos de la ficha y la Memoria del cliente; las etiquetas viven en idiomas.
WITH nueva AS (
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES('intervencion',1,'2026-01-01','intervencion.catalogos','00000000-0000-4000-8000-000000000001') RETURNING id
)
INSERT INTO catalogo.intervencion_opcion(version_id,lista,codigo,orden,admite_texto) SELECT nueva.id,datos.* FROM nueva CROSS JOIN(VALUES
('fase_intervencion','RECEPCION',0,false),
('fase_intervencion','EVALUACION',1,false),
('fase_intervencion','INTERVENCION',2,false),
('fase_intervencion','CIERRE_DEVOLUCION',3,false),
('fase_intervencion','SEGUIMIENTO',4,false),
('coordinacion','SALUD_MENTAL',0,false),
('coordinacion','SALUD',1,false),
('coordinacion','ATENCION_TEMPRANA',2,false),
('coordinacion','ADICCIONES',3,false),
('coordinacion','EDUCACION',4,false),
('coordinacion','AYUDAS_FAMILIARES',5,false),
('coordinacion','VIOLENCIA_SEXUAL',6,false),
('coordinacion','SPM',7,false),
('coordinacion','ONG',8,false),
('coordinacion','MUJER',9,false),
('coordinacion','DISCAPACIDAD',10,false),
('coordinacion','DEPENDENCIA',11,false),
('coordinacion','FISCALIA',12,false),
('coordinacion','OTROS',13,true),
('factor_menor','VIOLENCIA_GENERO',0,false),
('factor_menor','ADICCIONES',1,false),
('factor_menor','CONDUCTA',2,false),
('factor_menor','ABSENTISMO',3,false),
('factor_menor','VIOLENCIA_FILIOPARENTAL',4,false),
('factor_menor','ROLES_PARENTALES',5,false),
('factor_menor','INFRACTORES',6,false),
('factor_menor','SALUD_MENTAL',7,false),
('factor_menor','AUTOLESION',8,false),
('factor_menor','DISCAPACIDAD',9,false),
('factor_menor','ATENCION_TEMPRANA',10,false),
('tipologia','MALTRATO_FISICO',0,false),
('tipologia','MALTRATO_EMOCIONAL',1,false),
('tipologia','NEGLIGENCIA',2,false),
('tipologia','ABANDONO',3,false),
('tipologia','VIOLENCIA_SEXUAL',4,false),
('tipologia','CONTROL_CONDUCTA',5,false),
('tipologia','MUNCHAUSEN',6,false),
('tipologia','MALTRATO_PRENATAL',7,false),
('tipologia','CORRUPCION',8,false),
('tipologia','MENDICIDAD',9,false),
('tipologia','EXPLOTACION_LABORAL',10,false),
('tipologia','MALTRATO_INSTITUCIONAL',11,false),
('factor_familia','SIN_CONCIENCIA',0,false),
('factor_familia','SIN_MOTIVACION',1,false),
('factor_familia','DESEMPLEO',2,false),
('factor_familia','PRECARIEDAD_LABORAL',3,false),
('factor_familia','ECONOMIA',4,false),
('factor_familia','DESAHUCIO',5,false),
('factor_familia','VIVIENDA_COMPARTIDA',6,false),
('factor_familia','HABITABILIDAD',7,false),
('factor_familia','CONFLICTOS_FAMILIARES',8,false),
('factor_familia','VIOLENCIA_GENERO',9,false),
('factor_familia','FAMILIA_EXTENSA',10,false),
('factor_familia','ADICCIONES',11,false),
('factor_familia','CONFLICTOS_PARENTAL',12,false),
('factor_familia','PROGENITOR_ADOLESCENTE',13,false),
('factor_familia','SALUD_MENTAL',14,false),
('factor_familia','PLANIFICACION_FAMILIAR',15,false),
('factor_familia','DELINCUENCIA',16,false),
('factor_familia','PRISION',17,false),
('factor_familia','AISLAMIENTO',18,false),
('factor_familia','CONFLICTOS_ENTORNO',19,false),
('factor_familia','INMIGRACION_IRREGULAR',20,false),
('factor_familia','DISCAPACIDAD',21,false),
('factor_familia','DESIGUALDAD_GENERO',22,false)) datos(lista,codigo,orden,admite_texto);
INSERT INTO catalogo.publicacion(version_id,publicada_por)
 SELECT id,creada_por FROM catalogo.version WHERE ambito='intervencion';
