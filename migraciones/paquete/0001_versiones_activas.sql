DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regprocedure('auditoria.registrar(timestamp with time zone,text,text,uuid,uuid,text)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'paquete') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '60s';
CREATE SCHEMA paquete AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA paquete FROM PUBLIC;
GRANT USAGE ON SCHEMA paquete TO etf_app, etf_copia;

CREATE TABLE paquete.idioma_instalado (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo text NOT NULL CHECK (length(codigo) <= 63 AND codigo ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$'),
    version text NOT NULL CHECK (length(version) <= 32 AND version ~ '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'),
    ruta_relativa text NOT NULL UNIQUE,
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    estado text NOT NULL DEFAULT 'verificada' CHECK (estado = 'verificada'),
    registrado_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    registrado_por uuid NOT NULL CHECK (registrado_por <> '00000000-0000-0000-0000-000000000000'),
    UNIQUE (codigo, version),
    CHECK (ruta_relativa ~ ('^\.versiones/' || codigo || '/' || replace(version,'.','\.') || '/' || huella_sha256 || '/[0-9a-f]{32}/' || codigo || '$'))
);
CREATE TABLE paquete.tema_instalado (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre text NOT NULL CHECK (length(nombre) <= 64 AND nombre ~ '^[a-z0-9][a-z0-9-]{0,63}$'),
    version text NOT NULL CHECK (length(version) <= 32 AND version ~ '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'),
    ruta_relativa text NOT NULL UNIQUE,
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    estado text NOT NULL DEFAULT 'verificada' CHECK (estado = 'verificada'),
    registrado_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    registrado_por uuid NOT NULL CHECK (registrado_por <> '00000000-0000-0000-0000-000000000000'),
    UNIQUE (nombre, version),
    CHECK (ruta_relativa ~ ('^\.versiones/' || nombre || '/' || replace(version,'.','\.') || '/' || huella_sha256 || '/[0-9a-f]{32}/' || nombre || '$'))
);
CREATE TABLE paquete.referencia_activa (
    tipo text NOT NULL CHECK (tipo IN ('idioma','tema')),
    codigo text NOT NULL,
    version_contenido text NOT NULL,
    version bigint NOT NULL CHECK (version > 0 AND version <= 9007199254740991),
    idioma_id uuid REFERENCES paquete.idioma_instalado(id),
    tema_id uuid REFERENCES paquete.tema_instalado(id),
    activada_por uuid NOT NULL CHECK (activada_por <> '00000000-0000-0000-0000-000000000000'),
    activada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tipo,codigo),
    CHECK ((tipo='idioma' AND idioma_id IS NOT NULL AND tema_id IS NULL)
        OR (tipo='tema' AND tema_id IS NOT NULL AND idioma_id IS NULL))
);
CREATE INDEX referencia_idioma ON paquete.referencia_activa(idioma_id);
CREATE INDEX referencia_tema ON paquete.referencia_activa(tema_id);

ALTER TABLE paquete.idioma_instalado ENABLE ROW LEVEL SECURITY;
ALTER TABLE paquete.idioma_instalado FORCE ROW LEVEL SECURITY;
ALTER TABLE paquete.tema_instalado ENABLE ROW LEVEL SECURITY;
ALTER TABLE paquete.tema_instalado FORCE ROW LEVEL SECURITY;
ALTER TABLE paquete.referencia_activa ENABLE ROW LEVEL SECURITY;
ALTER TABLE paquete.referencia_activa FORCE ROW LEVEL SECURITY;
CREATE POLICY idioma_propietario ON paquete.idioma_instalado TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY tema_propietario ON paquete.tema_instalado TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY referencia_propietario ON paquete.referencia_activa TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY idioma_lectura ON paquete.idioma_instalado FOR SELECT TO etf_app
    USING (nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['administracion','coordinacion','equipo','consulta','sistema']::text[]);
CREATE POLICY tema_lectura ON paquete.tema_instalado FOR SELECT TO etf_app
    USING (nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['administracion','coordinacion','equipo','consulta','sistema']::text[]);
CREATE POLICY referencia_lectura ON paquete.referencia_activa FOR SELECT TO etf_app
    USING (nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['administracion','coordinacion','equipo','consulta','sistema']::text[]);
REVOKE ALL ON paquete.idioma_instalado, paquete.tema_instalado, paquete.referencia_activa FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON paquete.idioma_instalado, paquete.tema_instalado, paquete.referencia_activa TO etf_app,etf_copia;

CREATE FUNCTION paquete.inmutable() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog,paquete AS $$
BEGIN RAISE EXCEPTION 'paquete.version_inmutable' USING ERRCODE='42501'; END $$;
REVOKE EXECUTE ON FUNCTION paquete.inmutable() FROM PUBLIC,etf_app;
CREATE TRIGGER idioma_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON paquete.idioma_instalado
    FOR EACH STATEMENT EXECUTE FUNCTION paquete.inmutable();
CREATE TRIGGER tema_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON paquete.tema_instalado
    FOR EACH STATEMENT EXECUTE FUNCTION paquete.inmutable();

-- Proyección mínima para recursos públicos de arranque, también sin sesión.
-- No devuelve actores, preferencias ni contenido pendiente de activación.
CREATE FUNCTION paquete.referencias(p_tipo text)
RETURNS TABLE(tipo text,codigo text,version_contenido text,huella text,ruta text,version bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog,paquete AS $$
    SELECT r.tipo,r.codigo,r.version_contenido,coalesce(i.huella_sha256,t.huella_sha256),
        coalesce(i.ruta_relativa,t.ruta_relativa),r.version
    FROM paquete.referencia_activa r
    LEFT JOIN paquete.idioma_instalado i ON i.id=r.idioma_id
    LEFT JOIN paquete.tema_instalado t ON t.id=r.tema_id
    WHERE r.tipo=p_tipo ORDER BY r.codigo
$$;
REVOKE EXECUTE ON FUNCTION paquete.referencias(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION paquete.referencias(text) TO etf_app;

-- La limpieza consulta incluso versiones antiguas, nunca sólo las activas.
CREATE FUNCTION paquete.ruta_registrada(p_tipo text,p_ruta text) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,paquete AS $$
BEGIN
    -- Espera cualquier confirmación anterior del mismo código. Una lectura
    -- ordinaria podría no ver todavía un COMMIT cuyo resultado se perdió.
    PERFORM pg_advisory_xact_lock(hashtextextended('paquete.'||p_tipo||'.'||split_part(p_ruta,'/',2),0));
    RETURN (p_tipo='idioma' AND EXISTS(SELECT 1 FROM paquete.idioma_instalado WHERE ruta_relativa=p_ruta))
        OR (p_tipo='tema' AND EXISTS(SELECT 1 FROM paquete.tema_instalado WHERE ruta_relativa=p_ruta));
END
$$;
REVOKE EXECUTE ON FUNCTION paquete.ruta_registrada(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION paquete.ruta_registrada(text,text) TO etf_app;

CREATE FUNCTION paquete.activar(p_tipo text,p_codigo text,p_contenido text,p_huella text,p_ruta text,p_version bigint)
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,paquete AS $$
DECLARE previa bigint; nueva bigint; identificador uuid; usuario uuid;
BEGIN
    usuario:=nullif(current_setting('app.usuario_id',true),'')::uuid;
    IF usuario IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
        RAISE EXCEPTION 'paquete.acceso_denegado' USING ERRCODE='42501';
    END IF;
    IF p_tipo IS NULL OR p_tipo NOT IN ('idioma','tema') OR p_codigo IS NULL
       OR p_version IS NULL OR p_version < 0 OR p_version >= 9007199254740991 THEN
        RAISE EXCEPTION 'paquete.archivo_invalido' USING ERRCODE='22023';
    END IF;
    -- El bloqueo asesor cubre también una referencia que todavía no existe.
    PERFORM pg_advisory_xact_lock(hashtextextended('paquete.'||p_tipo||'.'||p_codigo,0));
    SELECT r.version INTO previa FROM paquete.referencia_activa r WHERE r.tipo=p_tipo AND r.codigo=p_codigo FOR UPDATE;
    IF coalesce(previa,0)<>p_version THEN
        RAISE EXCEPTION 'paquete.conflicto' USING ERRCODE='40001';
    END IF;
    nueva:=p_version+1;
    IF p_tipo='idioma' THEN
        INSERT INTO paquete.idioma_instalado(codigo,version,huella_sha256,ruta_relativa,registrado_por)
            VALUES(p_codigo,p_contenido,p_huella,p_ruta,usuario) RETURNING id INTO identificador;
    ELSE
        INSERT INTO paquete.tema_instalado(nombre,version,huella_sha256,ruta_relativa,registrado_por)
            VALUES(p_codigo,p_contenido,p_huella,p_ruta,usuario) RETURNING id INTO identificador;
    END IF;
    INSERT INTO paquete.referencia_activa(tipo,codigo,version_contenido,version,idioma_id,tema_id,activada_por)
        VALUES(p_tipo,p_codigo,p_contenido,nueva,
            CASE WHEN p_tipo='idioma' THEN identificador END,CASE WHEN p_tipo='tema' THEN identificador END,usuario)
        ON CONFLICT(tipo,codigo) DO UPDATE SET version_contenido=EXCLUDED.version_contenido,version=EXCLUDED.version,
            idioma_id=EXCLUDED.idioma_id,tema_id=EXCLUDED.tema_id,activada_por=EXCLUDED.activada_por,activada_en=clock_timestamp();
    -- Usa el mismo registrador persistente que F0.08b. No se puede confirmar
    -- esta función sin confirmar su evento, ni activar si el registro falla.
    PERFORM auditoria.registrar(clock_timestamp(),'paquete.activar','paquete',identificador,NULL,'ok');
    RETURN nueva;
END $$;
REVOKE EXECUTE ON FUNCTION paquete.activar(text,text,text,text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION paquete.activar(text,text,text,text,text,bigint) TO etf_app;
