DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('catalogo.version') IS NULL
       OR to_regclass('paquete.idioma_instalado') IS NULL
       OR to_regclass('paquete.referencia_activa') IS NULL
       OR to_regprocedure('paquete.activar_catalogo(text,text,text,text,bigint)') IS NOT NULL THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '60s';

-- Los idiomas instalados conservan semver y su ruta de paquete. Las
-- instantáneas de textos de catálogo usan la huella como versión inmutable.
ALTER TABLE paquete.idioma_instalado
    DROP CONSTRAINT idioma_instalado_version_check,
    DROP CONSTRAINT idioma_instalado_check,
    ADD CONSTRAINT idioma_instalado_version_check CHECK (
        (length(version) <= 32 AND version ~ '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$')
        OR version ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT idioma_instalado_check CHECK (
        (length(version) <= 32 AND version ~ '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
            AND ruta_relativa ~ ('^\.versiones/' || codigo || '/' || replace(version,'.','\.') || '/' || huella_sha256 || '/[0-9a-f]{32}/' || codigo || '$'))
        OR (version ~ '^[0-9a-f]{64}$' AND huella_sha256 = version
            AND ruta_relativa = codigo || '/catalogos/.versiones/' || version || '.json'));

ALTER TABLE paquete.referencia_activa
    DROP CONSTRAINT referencia_activa_tipo_check,
    DROP CONSTRAINT referencia_activa_check,
    ADD CONSTRAINT referencia_activa_tipo_check CHECK (tipo IN ('idioma','tema','catalogo')),
    ADD CONSTRAINT referencia_activa_check CHECK (
        (tipo IN ('idioma','catalogo') AND idioma_id IS NOT NULL AND tema_id IS NULL)
        OR (tipo = 'tema' AND tema_id IS NOT NULL AND idioma_id IS NULL));

-- Se conservan las tablas, los UUID, la RLS, los permisos de sólo lectura y
-- los disparadores de inmutabilidad de paquete/0001.
CREATE FUNCTION paquete.activar_catalogo(p_codigo text,p_contenido text,p_ruta text,p_huella text,p_esperada bigint)
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,paquete AS $$
DECLARE actual bigint; actor uuid; identificador uuid;
BEGIN
    actor := nullif(current_setting('app.usuario_id',true),'')::uuid;
    IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
        RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501';
    END IF;
    IF p_codigo IS NULL OR p_codigo NOT IN ('es','en')
       OR p_esperada IS NULL OR p_esperada < 0 OR p_esperada >= 9007199254740991
       OR p_contenido IS NULL OR p_contenido !~ '^[0-9a-f]{64}$'
       OR p_huella IS DISTINCT FROM p_contenido
       OR p_ruta IS DISTINCT FROM (p_codigo || '/catalogos/.versiones/' || p_contenido || '.json') THEN
        RAISE EXCEPTION 'paquete.publicacion_invalida' USING ERRCODE='22023';
    END IF;
    -- Mismo bloqueo que el puerto de publicación y limpieza de catálogos.
    PERFORM pg_advisory_xact_lock(697014,hashtext(p_codigo));
    SELECT r.version INTO actual FROM paquete.referencia_activa r
        WHERE r.tipo='catalogo' AND r.codigo=p_codigo FOR UPDATE;
    actual := coalesce(actual,0);
    IF actual <> p_esperada THEN
        RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=actual::text;
    END IF;
    INSERT INTO paquete.idioma_instalado(codigo,version,ruta_relativa,huella_sha256,registrado_por)
        VALUES(p_codigo,p_contenido,p_ruta,p_huella,actor)
        ON CONFLICT(codigo,version) DO NOTHING;
    SELECT i.id INTO identificador FROM paquete.idioma_instalado i
        WHERE i.codigo=p_codigo AND i.version=p_contenido;
    INSERT INTO paquete.referencia_activa(tipo,codigo,version_contenido,version,idioma_id,activada_por)
        VALUES('catalogo',p_codigo,p_contenido,actual+1,identificador,actor)
        ON CONFLICT(tipo,codigo) DO UPDATE SET version_contenido=EXCLUDED.version_contenido,
            version=EXCLUDED.version,idioma_id=EXCLUDED.idioma_id,
            activada_por=EXCLUDED.activada_por,activada_en=EXCLUDED.activada_en;
    PERFORM auditoria.registrar(clock_timestamp(),'paquete.activar_catalogo','paquete',identificador,NULL,'ok');
    RETURN actual+1;
END $$;
REVOKE EXECUTE ON FUNCTION paquete.activar_catalogo(text,text,text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION paquete.activar_catalogo(text,text,text,text,bigint) TO etf_app;
