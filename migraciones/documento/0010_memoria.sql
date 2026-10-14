SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.tipo_documento') IS NULL OR to_regclass('documento.emision_memoria') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES('memoria',false);
CREATE TABLE documento.emision_memoria (
 id uuid PRIMARY KEY,tipo_documento text NOT NULL DEFAULT 'memoria' CHECK(tipo_documento='memoria'),anio integer NOT NULL CHECK(anio BETWEEN 1 AND 9998),equipo_id uuid REFERENCES catalogo.equipo(id),
 idioma text NOT NULL,clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9_-]{1,128}$'),plantilla_id uuid NOT NULL,
 generada_por uuid NOT NULL REFERENCES identidad.usuario(id),generada_en timestamptz NOT NULL,
 huella_peticion bytea NOT NULL CHECK(octet_length(huella_peticion)=32),huella_datos bytea NOT NULL CHECK(octet_length(huella_datos)=32),
 huella_resultado bytea NOT NULL CHECK(octet_length(huella_resultado)=32),version_motor text NOT NULL,
 ruta_idioma text NOT NULL,version_idioma text NOT NULL,UNIQUE(generada_por,clave),
 FOREIGN KEY(plantilla_id,tipo_documento,idioma) REFERENCES documento.plantilla(id,tipo_documento,idioma)
);
ALTER TABLE documento.emision_memoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.emision_memoria FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.emision_memoria FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT,INSERT ON documento.emision_memoria TO etf_app;
GRANT SELECT ON documento.emision_memoria TO etf_copia;
CREATE POLICY memoria_lectura ON documento.emision_memoria FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
 ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
 ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])))
);
CREATE POLICY memoria_insercion ON documento.emision_memoria FOR INSERT TO etf_app WITH CHECK (
 generada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
 ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])))
);
CREATE INDEX memoria_equipo_anio ON documento.emision_memoria(equipo_id,anio,generada_en);
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('memoria','es','1.0.0','plantillas/memoria/v1.0.0.docx',decode('87466b64fccf8adb3bc451238e52527c29dec2756a9601faf55ae6f155c31ddd','hex'),'aceptacion',DATE '2026-01-01','motor_docx_1');
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) VALUES('memoria','en','1.0.0','plantillas/memoria/v1.0.0.docx',decode('87466b64fccf8adb3bc451238e52527c29dec2756a9601faf55ae6f155c31ddd','hex'),'aceptacion',DATE '2026-01-01','motor_docx_1');
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version) SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento='memoria';
