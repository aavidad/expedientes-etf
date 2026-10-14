-- F0.10 incorporará perfiles y asignaciones; la FK a catálogo de equipos
-- se añadirá en otra migración cuando F0.12 esté integrada.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('plataforma.migracion_aplicada') IS NULL
       OR to_regclass('auditoria.evento') IS NULL
       OR EXISTS (SELECT FROM pg_namespace WHERE nspname = 'identidad') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
CREATE SCHEMA identidad AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA identidad FROM PUBLIC;
GRANT USAGE ON SCHEMA identidad TO etf_app, etf_copia;
CREATE TABLE identidad.usuario (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    object_guid_ad uuid NOT NULL UNIQUE CHECK (object_guid_ad <> '00000000-0000-0000-0000-000000000000'),
    cuenta_ad text UNIQUE CHECK (cuenta_ad ~ '^[a-z0-9._-]{1,128}$'),
    cuenta_ad_anterior text CHECK (cuenta_ad_anterior ~ '^[a-z0-9._-]{1,128}$'),
    nombre_mostrado text NOT NULL CHECK (length(nombre_mostrado) <= 256),
    estado text NOT NULL DEFAULT 'pendiente_alta' CHECK (estado IN ('pendiente_alta','activo','baja')),
    idioma_preferido text,
    tema_preferido text,
    primer_acceso_en timestamptz NOT NULL,
    ultimo_acceso_en timestamptz NOT NULL CHECK (ultimo_acceso_en >= primer_acceso_en),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    CHECK (cuenta_ad IS NOT NULL OR cuenta_ad_anterior IS NOT NULL)
);
CREATE TABLE identidad.sesion (
    id_huella bytea PRIMARY KEY CHECK (octet_length(id_huella) = 32),
    usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
    creada_en timestamptz NOT NULL,
    ultima_actividad_en timestamptz NOT NULL CHECK (ultima_actividad_en >= creada_en),
    caduca_en timestamptz NOT NULL CHECK (caduca_en > creada_en)
);
CREATE INDEX sesion_usuario ON identidad.sesion(usuario_id);
CREATE INDEX sesion_caducidad ON identidad.sesion(caduca_en);
ALTER TABLE identidad.usuario ENABLE ROW LEVEL SECURITY;
ALTER TABLE identidad.usuario FORCE ROW LEVEL SECURITY;
ALTER TABLE identidad.sesion ENABLE ROW LEVEL SECURITY;
ALTER TABLE identidad.sesion FORCE ROW LEVEL SECURITY;
CREATE POLICY usuario_propietario ON identidad.usuario TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY sesion_propietario ON identidad.sesion TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY usuario_propio ON identidad.usuario FOR SELECT TO etf_app
    USING (id::text = nullif(current_setting('app.usuario_id', true), ''));
REVOKE ALL ON identidad.usuario, identidad.sesion FROM PUBLIC, etf_app, etf_copia;
GRANT SELECT ON identidad.usuario TO etf_app, etf_copia;
GRANT SELECT ON identidad.sesion TO etf_copia;

-- Sólo el adaptador de identidad llama a estas funciones tras validar AD.
-- El login nunca se usa para localizar o transferir una identidad.
CREATE FUNCTION identidad.registrar_acceso(guid uuid, cuenta text, nombre text, ahora timestamptz)
RETURNS SETOF identidad.usuario LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, identidad AS $$
DECLARE persona identidad.usuario; anterior uuid;
BEGIN
    -- Serializar cambios de login evita que dos accesos intercambien una cuenta
    -- mientras se libera. No se cambian estado, GUID ni permisos del anterior.
    PERFORM pg_advisory_xact_lock(hashtextextended('identidad.registrar_acceso', 0));
    UPDATE identidad.usuario SET cuenta_ad_anterior = cuenta_ad, cuenta_ad = NULL,
        version = version + 1 WHERE cuenta_ad = cuenta AND object_guid_ad <> guid RETURNING id INTO anterior;
    INSERT INTO identidad.usuario(object_guid_ad, cuenta_ad, nombre_mostrado, primer_acceso_en, ultimo_acceso_en)
    VALUES (guid, cuenta, nombre, ahora, ahora)
    ON CONFLICT (object_guid_ad) DO UPDATE SET
        cuenta_ad_anterior = CASE WHEN usuario.cuenta_ad IS DISTINCT FROM EXCLUDED.cuenta_ad
            THEN coalesce(usuario.cuenta_ad, usuario.cuenta_ad_anterior) ELSE usuario.cuenta_ad_anterior END,
        cuenta_ad = EXCLUDED.cuenta_ad,
        nombre_mostrado = EXCLUDED.nombre_mostrado, ultimo_acceso_en = EXCLUDED.ultimo_acceso_en,
        version = usuario.version + 1
    RETURNING * INTO persona;
    PERFORM set_config('app.usuario_id', persona.id::text, true);
    IF anterior IS NOT NULL THEN
        PERFORM auditoria.registrar(ahora, 'identidad.cuenta_reutilizada', 'identidad.usuario', anterior, NULL, 'ok');
    END IF;
    PERFORM auditoria.registrar(ahora, 'identidad.usuario_acceso', 'identidad.usuario', persona.id, NULL, 'ok');
    RETURN NEXT persona;
END $$;
CREATE FUNCTION identidad.crear_sesion(huella bytea, persona uuid, creada timestamptz, actividad timestamptz, caduca timestamptz)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
BEGIN
    IF NOT EXISTS (SELECT FROM identidad.usuario WHERE id = persona AND estado <> 'baja') THEN
        RAISE EXCEPTION 'identidad.usuario_no_disponible';
    END IF;
    INSERT INTO identidad.sesion VALUES (huella, persona, creada, actividad, caduca);
    PERFORM set_config('app.usuario_id', persona::text, true);
    PERFORM auditoria.registrar(creada, 'identidad.sesion_entrar', 'identidad.usuario', persona, NULL, 'ok');
END $$;
CREATE FUNCTION identidad.bloquear_sesion(huella bytea)
RETURNS SETOF identidad.sesion LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE ses identidad.sesion;
BEGIN
    SELECT * INTO ses FROM identidad.sesion WHERE id_huella = huella FOR UPDATE;
    IF FOUND THEN
        PERFORM set_config('app.usuario_id', ses.usuario_id::text, true);
        PERFORM set_config('app.sesion_huella', encode(huella, 'hex'), true);
        RETURN NEXT ses;
    END IF;
END $$;
CREATE FUNCTION identidad.tocar_sesion(huella bytea, actividad timestamptz)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
BEGIN
    IF encode(huella, 'hex') IS DISTINCT FROM current_setting('app.sesion_huella', true) THEN
        RAISE EXCEPTION 'identidad.sesion_no_valida';
    END IF;
    UPDATE identidad.sesion SET ultima_actividad_en = actividad WHERE id_huella = huella;
    PERFORM auditoria.registrar(actividad, 'identidad.sesion_consultar', 'identidad.usuario',
        nullif(current_setting('app.usuario_id', true), '')::uuid, NULL, 'ok');
END $$;
CREATE FUNCTION identidad.eliminar_sesion(huella bytea)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE persona uuid;
BEGIN
    DELETE FROM identidad.sesion WHERE id_huella = huella RETURNING usuario_id INTO persona;
    PERFORM set_config('app.usuario_id', coalesce(persona::text, ''), true);
    PERFORM auditoria.registrar(clock_timestamp(), 'identidad.sesion_salir', 'identidad.usuario', persona, NULL, 'ok');
END $$;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA identidad FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.registrar_acceso(uuid,text,text,timestamptz),
    identidad.crear_sesion(bytea,uuid,timestamptz,timestamptz,timestamptz),
    identidad.bloquear_sesion(bytea), identidad.tocar_sesion(bytea,timestamptz),
    identidad.eliminar_sesion(bytea) TO etf_app;
