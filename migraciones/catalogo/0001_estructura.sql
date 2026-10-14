DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('plataforma.migracion_aplicada') IS NULL
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalogo')
       OR EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gist') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;

CREATE SCHEMA catalogo AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA catalogo FROM PUBLIC;
GRANT USAGE ON SCHEMA catalogo TO etf_app, etf_copia;
CREATE EXTENSION btree_gist WITH SCHEMA catalogo;
SET LOCAL search_path = pg_catalog, catalogo;

CREATE TABLE catalogo.version (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ambito text NOT NULL CHECK (ambito IN ('listas','equivalencias','plazos','tramos','numeracion','escalas','asignacion_municipios','tipos_documento')),
    numero integer NOT NULL CHECK (numero > 0),
    vigente_desde date NOT NULL CHECK (isfinite(vigente_desde)),
    vigente_hasta date CHECK (isfinite(vigente_hasta) AND vigente_hasta > vigente_desde),
    estado text NOT NULL DEFAULT 'borrador' CHECK (estado IN ('borrador','publicada','retirada')),
    motivo text NOT NULL CHECK (motivo ~ '^[a-z][a-z0-9_.]*$'),
    creada_por uuid NOT NULL,
    publicada_por uuid,
    publicada_en timestamptz,
    UNIQUE (ambito, numero),
    CHECK ((estado = 'publicada') = (publicada_por IS NOT NULL AND publicada_en IS NOT NULL)),
    EXCLUDE USING gist (ambito WITH =, daterange(vigente_desde, vigente_hasta, '[)') WITH &&) WHERE (estado = 'publicada')
);

CREATE TABLE catalogo.equipo (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo text NOT NULL UNIQUE CHECK (codigo ~ '^(0[1-9]|1[0-2])$'),
    clave_texto text NOT NULL UNIQUE CHECK (clave_texto = 'equipo.' || codigo),
    activo_desde date NOT NULL CHECK (isfinite(activo_desde)),
    activo_hasta date CHECK (isfinite(activo_hasta) AND activo_hasta > activo_desde)
);
CREATE TABLE catalogo.centro_ss (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo text NOT NULL UNIQUE CHECK (codigo ~ '^[A-Z][A-Z0-9_]*$'),
    clave_texto text NOT NULL UNIQUE CHECK (clave_texto = 'centro.' || codigo),
    activo_desde date NOT NULL CHECK (isfinite(activo_desde)),
    activo_hasta date CHECK (isfinite(activo_hasta) AND activo_hasta > activo_desde)
);
CREATE TABLE catalogo.municipio (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ine text NOT NULL UNIQUE CHECK (ine ~ '^18[0-9]{3}$|^ELA-18[0-9]{3}-[A-Z0-9-]+$'),
    clave_texto text NOT NULL UNIQUE CHECK (clave_texto = 'municipio.' || ine),
    tipo text NOT NULL CHECK (tipo IN ('municipio','ela')),
    matriz_id uuid REFERENCES catalogo.municipio(id),
    ambito_diputacion boolean NOT NULL,
    CHECK ((tipo = 'ela') = (matriz_id IS NOT NULL)),
    CHECK (matriz_id IS DISTINCT FROM id)
);
CREATE TABLE catalogo.municipio_asignacion (
    version_id uuid NOT NULL REFERENCES catalogo.version(id),
    municipio_id uuid NOT NULL REFERENCES catalogo.municipio(id),
    centro_ss_id uuid REFERENCES catalogo.centro_ss(id),
    equipo_id uuid REFERENCES catalogo.equipo(id),
    zts text CHECK (zts ~ '^[A-Z0-9_]+$'),
    PRIMARY KEY (version_id, municipio_id)
);
CREATE INDEX municipio_asignacion_equipo ON catalogo.municipio_asignacion(equipo_id);
CREATE INDEX municipio_asignacion_centro ON catalogo.municipio_asignacion(centro_ss_id);
CREATE INDEX municipio_matriz ON catalogo.municipio(matriz_id);
CREATE TABLE catalogo.lista_opcion (
    version_id uuid NOT NULL REFERENCES catalogo.version(id),
    lista text NOT NULL CHECK (lista ~ '^[a-z][a-z0-9_]*$'),
    codigo text NOT NULL CHECK (codigo ~ '^[A-Z][A-Z0-9_]*$'),
    orden integer NOT NULL CHECK (orden >= 0),
    admite_texto boolean NOT NULL DEFAULT false,
    activa boolean NOT NULL DEFAULT true,
    PRIMARY KEY (version_id, lista, codigo),
    UNIQUE (version_id, lista, orden)
);

-- Historia de publicación y de los límites anteriores, sólo de inserción.
-- No sustituye la auditoría encadenada del módulo auditoria (F0.14).
CREATE TABLE catalogo.publicacion (
    version_id uuid PRIMARY KEY REFERENCES catalogo.version(id),
    anterior_id uuid REFERENCES catalogo.version(id),
    anterior_hasta date,
    publicada_por uuid NOT NULL,
    publicada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE FUNCTION catalogo.proteger_version() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, catalogo AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.estado <> 'borrador' THEN RAISE EXCEPTION 'catalogo.publicacion_controlada'; END IF;
        RETURN NEW;
    END IF;
    IF OLD.estado = 'publicada' OR (TG_OP = 'UPDATE' AND NEW.estado = 'publicada') THEN
        -- Sólo el disparador de historia de publicación puede cerrar un límite
        -- o publicar el borrador. Nunca puede cambiar el contenido publicado.
        IF TG_OP = 'UPDATE' AND pg_trigger_depth() = 2 AND current_user = 'etf_propietario' THEN
            IF OLD.estado = 'borrador' AND NEW.estado = 'publicada'
               AND (to_jsonb(NEW) - 'estado' - 'publicada_por' - 'publicada_en') =
                   (to_jsonb(OLD) - 'estado' - 'publicada_por' - 'publicada_en') THEN RETURN NEW; END IF;
            IF OLD.estado = 'publicada' AND NEW.estado = 'publicada'
               AND (to_jsonb(NEW) - 'vigente_hasta') = (to_jsonb(OLD) - 'vigente_hasta')
               AND NEW.vigente_hasta IS NOT NULL
               AND (OLD.vigente_hasta IS NULL OR NEW.vigente_hasta < OLD.vigente_hasta) THEN RETURN NEW; END IF;
        END IF;
        RAISE EXCEPTION 'catalogo.version_inmutable';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER proteger_version BEFORE INSERT OR UPDATE OR DELETE ON catalogo.version
FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_version();

CREATE FUNCTION catalogo.proteger_regla() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, catalogo AS $$
DECLARE v catalogo.version;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        SELECT * INTO v FROM catalogo.version WHERE id = OLD.version_id FOR UPDATE;
        IF v.estado <> 'borrador' THEN RAISE EXCEPTION 'catalogo.version_inmutable'; END IF;
    END IF;
    IF TG_OP <> 'DELETE' THEN
        SELECT * INTO v FROM catalogo.version WHERE id = NEW.version_id FOR UPDATE;
        IF v.estado <> 'borrador' THEN RAISE EXCEPTION 'catalogo.version_inmutable'; END IF;
        IF v.ambito <> TG_ARGV[0] THEN RAISE EXCEPTION 'catalogo.ambito_invalido'; END IF;
        RETURN NEW;
    END IF;
    RETURN OLD;
END $$;
CREATE TRIGGER proteger_asignacion BEFORE INSERT OR UPDATE OR DELETE ON catalogo.municipio_asignacion
FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('asignacion_municipios');
CREATE TRIGGER proteger_opcion BEFORE INSERT OR UPDATE OR DELETE ON catalogo.lista_opcion
FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('listas');

CREATE FUNCTION catalogo.impedir_cambio() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, catalogo AS $$
BEGIN RAISE EXCEPTION 'catalogo.contenido_inmutable'; END $$;
CREATE TRIGGER publicacion_inmutable BEFORE UPDATE OR DELETE ON catalogo.publicacion
FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();
-- Los identificadores geográficos ya publicados no se reescriben. Las nuevas
-- asignaciones se expresan con una nueva versión, sin alterar las antiguas.
CREATE TRIGGER equipo_inmutable BEFORE UPDATE OR DELETE ON catalogo.equipo
FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();
CREATE TRIGGER centro_inmutable BEFORE UPDATE OR DELETE ON catalogo.centro_ss
FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();
CREATE TRIGGER municipio_inmutable BEFORE UPDATE OR DELETE ON catalogo.municipio
FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio();

CREATE FUNCTION catalogo.aplicar_publicacion() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, catalogo AS $$
DECLARE nueva catalogo.version; anterior catalogo.version;
BEGIN
    SELECT * INTO nueva FROM catalogo.version WHERE id = NEW.version_id;
    IF NOT FOUND THEN RAISE EXCEPTION 'catalogo.version_no_disponible'; END IF;
    PERFORM pg_advisory_xact_lock(697012, hashtext(nueva.ambito));
    SELECT * INTO nueva FROM catalogo.version WHERE id = NEW.version_id FOR UPDATE;
    IF nueva.estado <> 'borrador' THEN RAISE EXCEPTION 'catalogo.version_inmutable'; END IF;
    IF EXISTS (SELECT 1 FROM catalogo.version WHERE ambito = nueva.ambito AND estado = 'publicada'
               AND vigente_desde >= nueva.vigente_desde) THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
    SELECT * INTO anterior FROM catalogo.version WHERE ambito = nueva.ambito AND estado = 'publicada'
       AND (vigente_hasta IS NULL OR vigente_hasta > nueva.vigente_desde) FOR UPDATE;
    NEW.anterior_id := anterior.id;
    NEW.anterior_hasta := anterior.vigente_hasta;
    -- BEFORE no tiene aún historia persistida; las actualizaciones anidadas
    -- pasan por proteger_version a profundidad dos en esta misma transacción.
    IF anterior.id IS NOT NULL THEN
        UPDATE catalogo.version SET vigente_hasta = nueva.vigente_desde WHERE id = anterior.id;
    END IF;
    UPDATE catalogo.version SET estado = 'publicada', publicada_por = NEW.publicada_por,
       publicada_en = NEW.publicada_en WHERE id = nueva.id;
    RETURN NEW;
END $$;
CREATE TRIGGER aplicar_publicacion BEFORE INSERT ON catalogo.publicacion
FOR EACH ROW EXECUTE FUNCTION catalogo.aplicar_publicacion();

CREATE FUNCTION catalogo.publicar(p_version uuid, p_actor uuid) RETURNS void
LANGUAGE sql SET search_path = pg_catalog, catalogo AS $$
    INSERT INTO catalogo.publicacion(version_id, publicada_por) VALUES (p_version, p_actor);
$$;
REVOKE EXECUTE ON FUNCTION catalogo.proteger_version(), catalogo.proteger_regla(),
    catalogo.impedir_cambio(), catalogo.aplicar_publicacion(), catalogo.publicar(uuid, uuid)
    FROM PUBLIC, etf_app, etf_copia;
GRANT EXECUTE ON FUNCTION catalogo.publicar(uuid, uuid) TO etf_propietario;

DO $$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['version','equipo','centro_ss','municipio','municipio_asignacion','lista_opcion','publicacion'] LOOP
        EXECUTE format('ALTER TABLE catalogo.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE catalogo.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format('CREATE POLICY propietario ON catalogo.%I TO etf_propietario USING (true) WITH CHECK (true)', tabla);
        EXECUTE format('CREATE POLICY lectura ON catalogo.%I FOR SELECT TO etf_app USING (
            nullif(current_setting(''app.usuario_id'', true), '''') IS NOT NULL AND
            string_to_array(current_setting(''app.perfiles'', true), '','') &&
            ARRAY[''coordinacion'',''equipo'',''consulta'',''administracion'',''sistema''])', tabla);
        EXECUTE format('REVOKE ALL ON catalogo.%I FROM PUBLIC, etf_app, etf_copia', tabla);
        EXECUTE format('GRANT SELECT ON catalogo.%I TO etf_app, etf_copia', tabla);
    END LOOP;
END $$;
-- El rol Consulta sólo recibe contenido publicado, necesario para sus cifras.
DROP POLICY lectura ON catalogo.version;
CREATE POLICY lectura ON catalogo.version FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id', true), '') IS NOT NULL AND
    ('administracion' = ANY(string_to_array(current_setting('app.perfiles', true), ','))
     OR (estado = 'publicada' AND string_to_array(current_setting('app.perfiles', true), ',') && ARRAY['coordinacion','equipo','consulta','sistema']))
);
DROP POLICY lectura ON catalogo.lista_opcion;
CREATE POLICY lectura ON catalogo.lista_opcion FOR SELECT TO etf_app USING (
    EXISTS (SELECT 1 FROM catalogo.version v WHERE v.id = version_id)
);
DROP POLICY lectura ON catalogo.municipio_asignacion;
CREATE POLICY lectura ON catalogo.municipio_asignacion FOR SELECT TO etf_app USING (
    EXISTS (SELECT 1 FROM catalogo.version v WHERE v.id = version_id)
);
DROP POLICY lectura ON catalogo.publicacion;
CREATE POLICY lectura ON catalogo.publicacion FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id', true), '') IS NOT NULL AND
    'administracion' = ANY(string_to_array(current_setting('app.perfiles', true), ','))
);
DO $$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['version','equipo','centro_ss','municipio','municipio_asignacion','lista_opcion','publicacion'] LOOP
        EXECUTE format('CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.%I FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio()', tabla);
    END LOOP;
END $$;
