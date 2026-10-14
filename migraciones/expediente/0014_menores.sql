SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.adultos_revision') IS NULL
 OR to_regclass('expediente.menores_revision') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='expediente.menor'::regclass AND conname='menor_expediente_id_orden_key') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- El orden pertenece a cada revisión. Las identidades retiradas permanecen
-- referenciables por evaluación y no consumen puestos de la colección vigente.
ALTER TABLE expediente.menor DROP CONSTRAINT menor_expediente_id_orden_key;
ALTER TABLE expediente.menor ADD COLUMN datos jsonb NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(datos)='object');
CREATE INDEX menor_expediente ON expediente.menor(expediente_id);
CREATE TABLE expediente.menores_revision (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0),vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id),registrado_en timestamptz NOT NULL,
 UNIQUE(id,expediente_id,equipo_id),UNIQUE(expediente_id,version),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.menor_revision (
 revision_id uuid NOT NULL,menor_id uuid NOT NULL,expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 orden integer NOT NULL CHECK(orden BETWEEN 1 AND 15),sexo text NOT NULL CHECK(sexo IN ('H','M')),
 fecha_nacimiento date CHECK(isfinite(fecha_nacimiento)),nivel_escolarizacion text,
 subprograma text NOT NULL CHECK(subprograma IN ('RIESGO_SOCIAL','REUNIFICACION')),
 centro_proteccion_codigo text,propuesto_atencion_temprana boolean,
 PRIMARY KEY(revision_id,menor_id),UNIQUE(revision_id,orden),
 FOREIGN KEY(revision_id,expediente_id,equipo_id) REFERENCES expediente.menores_revision(id,expediente_id,equipo_id),
 FOREIGN KEY(menor_id,expediente_id,equipo_id) REFERENCES expediente.menor(id,expediente_id,equipo_id)
);
CREATE INDEX menores_historia ON expediente.menores_revision(expediente_id,vigente_desde DESC,version DESC);
-- La primera vigencia procede de la aceptación ya registrada, sin fechas ni
-- atributos inventados. Los campos que aún no se conocen permanecen nulos.
INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 SELECT e.id,e.equipo_id,e.version,e.fecha_aceptacion,e.modificado_por,e.modificado_en
 FROM expediente.expediente e WHERE e.fecha_aceptacion IS NOT NULL;
INSERT INTO expediente.menor_revision(revision_id,menor_id,expediente_id,equipo_id,orden,sexo,subprograma)
 SELECT r.id,m.id,m.expediente_id,m.equipo_id,m.orden,m.sexo,e.subprograma
 FROM expediente.menor m JOIN expediente.expediente e ON e.id=m.expediente_id
 JOIN expediente.menores_revision r ON r.expediente_id=e.id;

-- Tablas de vínculos D4: su edición y sus vigencias corresponden a F2.01c.
CREATE TABLE expediente.marca (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 lista text NOT NULL CHECK(length(lista) BETWEEN 1 AND 64),codigo text NOT NULL CHECK(length(codigo) BETWEEN 1 AND 64),texto_otros text,
 UNIQUE(id,expediente_id,equipo_id),UNIQUE(expediente_id,lista,codigo),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.marca_menor_excluido (
 marca_id uuid NOT NULL,menor_id uuid NOT NULL,expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 PRIMARY KEY(marca_id,menor_id),
 FOREIGN KEY(marca_id,expediente_id,equipo_id) REFERENCES expediente.marca(id,expediente_id,equipo_id),
 FOREIGN KEY(menor_id,expediente_id,equipo_id) REFERENCES expediente.menor(id,expediente_id,equipo_id)
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['menores_revision','menor_revision','marca','marca_menor_excluido'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT,INSERT ON expediente.%I TO etf_app',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia',tabla);
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'',''))
   OR (''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[])))))',tabla);
  EXECUTE format('CREATE POLICY escritura ON expediente.%I FOR INSERT TO etf_app WITH CHECK (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'',''))
   OR (''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[]))))
   AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=%1$I.expediente_id AND e.equipo_id=%1$I.equipo_id AND e.estado IN (''aceptada'',''en_intervencion'')))',tabla);
 END LOOP;
END $$;
-- Aceptar requiere INSERT de Coordinación; no abre UPDATE de fichas.
DROP POLICY coordinacion ON expediente.menor;
CREATE POLICY coordinacion_lectura ON expediente.menor FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY coordinacion_insertar ON expediente.menor FOR INSERT TO etf_app WITH CHECK (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND datos='{}'::jsonb
 AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=menor.expediente_id AND e.equipo_id=menor.equipo_id AND e.estado='aceptada'));
GRANT UPDATE(datos) ON expediente.menor TO etf_app;
CREATE POLICY equipo_insertar ON expediente.menor FOR INSERT TO etf_app WITH CHECK (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))
 AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=menor.expediente_id AND e.equipo_id=menor.equipo_id AND e.estado IN ('aceptada','en_intervencion')));
CREATE POLICY equipo_modificar ON expediente.menor FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))
 AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=menor.expediente_id AND e.equipo_id=menor.equipo_id AND e.estado IN ('aceptada','en_intervencion')));
CREATE VIEW expediente.consulta_menores_vigencia WITH(security_invoker=true) AS
 SELECT id,expediente_id,equipo_id,version,vigente_desde,lead(vigente_desde) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) AS vigente_hasta
 FROM (SELECT DISTINCT ON(expediente_id,vigente_desde) * FROM expediente.menores_revision ORDER BY expediente_id,vigente_desde,version DESC) r;
CREATE VIEW expediente.consulta_menor WITH(security_invoker=true) AS
 SELECT m.menor_id AS id,m.expediente_id,m.equipo_id,m.orden,m.sexo,m.fecha_nacimiento,m.nivel_escolarizacion,m.subprograma,m.centro_proteccion_codigo,m.propuesto_atencion_temprana,
 r.vigente_desde,r.vigente_hasta,r.version FROM expediente.menor_revision m JOIN expediente.consulta_menores_vigencia r ON r.id=m.revision_id;
REVOKE ALL ON expediente.consulta_menores_vigencia,expediente.consulta_menor FROM PUBLIC;
GRANT SELECT ON expediente.consulta_menores_vigencia,expediente.consulta_menor TO etf_app,etf_copia;
-- Mantiene el contrato de aceptación previo. Sólo el INSERT sin ficha ampliada
-- crea la historia inicial; editar menores registra su propia revisión completa.
CREATE FUNCTION expediente.iniciar_historia_menor() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
DECLARE revision uuid; BEGIN
 IF NEW.datos<>'{}'::jsonb THEN RETURN NEW; END IF;
 INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 SELECT id,equipo_id,version,fecha_aceptacion,modificado_por,modificado_en FROM expediente.expediente WHERE id=NEW.expediente_id
 ON CONFLICT(expediente_id,version) DO NOTHING;
 SELECT r.id INTO STRICT revision FROM expediente.menores_revision r JOIN expediente.expediente e ON e.id=r.expediente_id AND e.version=r.version WHERE e.id=NEW.expediente_id;
 INSERT INTO expediente.menor_revision(revision_id,menor_id,expediente_id,equipo_id,orden,sexo,subprograma)
 SELECT revision,NEW.id,NEW.expediente_id,NEW.equipo_id,NEW.orden,NEW.sexo,e.subprograma FROM expediente.expediente e WHERE e.id=NEW.expediente_id;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.iniciar_historia_menor() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER iniciar_historia_menor AFTER INSERT ON expediente.menor FOR EACH ROW EXECUTE FUNCTION expediente.iniciar_historia_menor();
-- La política antigua de Coordinación no debe abrir ediciones en cerrados.
CREATE FUNCTION expediente.proteger_menor() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF current_user='etf_app' AND NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.expediente_id AND e.equipo_id=NEW.equipo_id AND e.estado IN ('aceptada','en_intervencion')) THEN
  RAISE EXCEPTION 'error.estado_no_permite' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_menor() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_menor BEFORE INSERT OR UPDATE ON expediente.menor FOR EACH ROW EXECUTE FUNCTION expediente.proteger_menor();
-- Serializa también las tablas nuevas con cierre y reapertura de F3.02b.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['menores_revision','menor_revision','marca','marca_menor_excluido'] LOOP
  EXECUTE format('CREATE TRIGGER exigir_expediente_abierto BEFORE INSERT OR UPDATE OR DELETE ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.exigir_expediente_abierto()',tabla);
 END LOOP;
END $$;
