-- Ejecutada en una transacción por cmd/migrar, con el rol propietario.
DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('plataforma.migracion_aplicada') IS NULL
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auditoria') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '60s';

CREATE SCHEMA auditoria AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA auditoria FROM PUBLIC;
GRANT USAGE ON SCHEMA auditoria TO etf_app, etf_copia;

CREATE TABLE auditoria.evento (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ocurrido_en timestamptz NOT NULL CHECK (ocurrido_en >= '0001-01-01 00:00:00+00'::timestamptz
                                           AND ocurrido_en < '10000-01-01 00:00:00+00'::timestamptz),
    registrado_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    usuario_id uuid CHECK (usuario_id <> '00000000-0000-0000-0000-000000000000'),
    actor_sistema boolean NOT NULL,
    perfiles text[] NOT NULL,
    accion text NOT NULL CHECK (accion ~ '^[a-z][a-z0-9_.]{0,127}$' AND accion !~ '\.\.|\.$'),
    entidad text NOT NULL CHECK (entidad ~ '^[a-z][a-z0-9_.]{0,127}$' AND entidad !~ '\.\.|\.$'),
    entidad_id uuid CHECK (entidad_id <> '00000000-0000-0000-0000-000000000000'),
    expediente_id uuid CHECK (expediente_id <> '00000000-0000-0000-0000-000000000000'),
    resultado text NOT NULL CHECK (resultado IN ('ok', 'denegado', 'error')),
    huella_anterior bytea NOT NULL CHECK (octet_length(huella_anterior) = 32),
    huella bytea NOT NULL CHECK (octet_length(huella) = 32),
    CHECK (actor_sistema = (usuario_id IS NULL)),
    CHECK (cardinality(perfiles) <= 16 AND array_position(perfiles, NULL) IS NULL)
);
CREATE INDEX evento_registrado ON auditoria.evento (registrado_en, id);

-- El punto cero representa una cadena todavía vacía; no hay UUID artificial.
CREATE TABLE auditoria.sello_diario (
    dia date PRIMARY KEY,
    ultimo_evento_id bigint NOT NULL CHECK (ultimo_evento_id >= 0),
    ultima_huella bytea NOT NULL CHECK (octet_length(ultima_huella) = 32),
    CHECK ((ultimo_evento_id = 0) = (ultima_huella = decode(repeat('00', 32), 'hex')))
);

ALTER TABLE auditoria.evento ENABLE ROW LEVEL SECURITY;
ALTER TABLE auditoria.evento FORCE ROW LEVEL SECURITY;
ALTER TABLE auditoria.sello_diario ENABLE ROW LEVEL SECURITY;
ALTER TABLE auditoria.sello_diario FORCE ROW LEVEL SECURITY;
CREATE POLICY evento_propietario ON auditoria.evento TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY sello_propietario ON auditoria.sello_diario TO etf_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON auditoria.evento, auditoria.sello_diario FROM PUBLIC, etf_app, etf_copia;
REVOKE ALL ON SEQUENCE auditoria.evento_id_seq FROM PUBLIC, etf_app, etf_copia;
GRANT SELECT ON auditoria.evento, auditoria.sello_diario TO etf_copia;

CREATE FUNCTION auditoria.rechazar_mutacion() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, auditoria AS $$
BEGIN
    RAISE EXCEPTION 'auditoria.solo_insercion' USING ERRCODE = '42501';
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.rechazar_mutacion() FROM PUBLIC, etf_app;
CREATE TRIGGER evento_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON auditoria.evento
    FOR EACH STATEMENT EXECUTE FUNCTION auditoria.rechazar_mutacion();
CREATE TRIGGER sello_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON auditoria.sello_diario
    FOR EACH STATEMENT EXECUTE FUNCTION auditoria.rechazar_mutacion();

-- int8send produce los ocho bytes big endian de la forma canónica v1.
CREATE FUNCTION auditoria.texto_binario(p_texto text) RETURNS bytea
LANGUAGE sql IMMUTABLE STRICT SET search_path = pg_catalog, auditoria AS $$
    SELECT int8send(octet_length(convert_to(p_texto, 'UTF8'))::bigint) || convert_to(p_texto, 'UTF8')
$$;
REVOKE EXECUTE ON FUNCTION auditoria.texto_binario(text) FROM PUBLIC, etf_app;

CREATE FUNCTION auditoria.calcular_huella(p_evento auditoria.evento) RETURNS bytea
LANGUAGE plpgsql STABLE STRICT SET search_path = pg_catalog, auditoria AS $$
DECLARE contenido bytea; perfil text;
BEGIN
    contenido := p_evento.huella_anterior || auditoria.texto_binario('etf.auditoria.v1')
        || int8send(p_evento.id)
        || auditoria.texto_binario(to_char(p_evento.ocurrido_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
        || auditoria.texto_binario(CASE WHEN p_evento.actor_sistema THEN 'sistema' ELSE p_evento.usuario_id::text END)
        || int8send(cardinality(p_evento.perfiles)::bigint);
    FOREACH perfil IN ARRAY p_evento.perfiles LOOP
        contenido := contenido || auditoria.texto_binario(perfil);
    END LOOP;
    RETURN sha256(contenido || auditoria.texto_binario(p_evento.accion)
        || auditoria.texto_binario(p_evento.entidad)
        || auditoria.texto_binario(coalesce(p_evento.entidad_id::text, ''))
        || auditoria.texto_binario(coalesce(p_evento.expediente_id::text, ''))
        || auditoria.texto_binario(p_evento.resultado));
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.calcular_huella(auditoria.evento) FROM PUBLIC, etf_app;

-- Actor y perfiles proceden del contexto de la sesión fijado por el servidor.
-- Un contexto vacío identifica tareas automáticas o accesos aún sin identidad.
CREATE FUNCTION auditoria.registrar(p_ocurrido_en timestamptz, p_accion text, p_entidad text,
    p_entidad_id uuid, p_expediente_id uuid, p_resultado text) RETURNS auditoria.evento
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, auditoria AS $$
DECLARE e auditoria.evento; usuario text; contexto_perfiles text;
BEGIN
    usuario := coalesce(current_setting('app.usuario_id', true), '');
    contexto_perfiles := coalesce(current_setting('app.perfiles', true), '');
    IF (usuario = '' AND contexto_perfiles <> '') OR
       (usuario <> '' AND (usuario !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
         OR usuario = '00000000-0000-0000-0000-000000000000')) THEN
        RAISE EXCEPTION 'auditoria.usuario_invalido' USING ERRCODE = '22023';
    END IF;
    e.perfiles := CASE WHEN contexto_perfiles = '' THEN ARRAY[]::text[] ELSE string_to_array(contexto_perfiles, ',') END;
    IF cardinality(e.perfiles) > 16 OR EXISTS (
        SELECT 1 FROM unnest(e.perfiles) p WHERE p !~ '^[a-z][a-z0-9_.]{0,127}$' OR p ~ '\.\.|\.$'
    ) OR (SELECT count(*) <> count(DISTINCT p) FROM unnest(e.perfiles) p) THEN
        RAISE EXCEPTION 'auditoria.perfiles_invalidos' USING ERRCODE = '22023';
    END IF;
    IF p_ocurrido_en IS NULL OR NOT isfinite(p_ocurrido_en)
       OR p_ocurrido_en < '0001-01-01 00:00:00+00'::timestamptz
       OR p_ocurrido_en >= '10000-01-01 00:00:00+00'::timestamptz THEN
        RAISE EXCEPTION 'auditoria.fecha_invalida' USING ERRCODE = '22023';
    END IF;
    IF p_accion IS NULL OR p_entidad IS NULL OR p_accion !~ '^[a-z][a-z0-9_.]{0,127}$'
       OR p_entidad !~ '^[a-z][a-z0-9_.]{0,127}$' OR p_accion ~ '\.\.|\.$' OR p_entidad ~ '\.\.|\.$' THEN
        RAISE EXCEPTION 'auditoria.codigo_invalido' USING ERRCODE = '22023';
    END IF;
    IF p_resultado IS NULL OR p_resultado NOT IN ('ok', 'denegado', 'error') THEN
        RAISE EXCEPTION 'auditoria.resultado_invalido' USING ERRCODE = '22023';
    END IF;
    IF p_entidad_id = '00000000-0000-0000-0000-000000000000'::uuid
       OR p_expediente_id = '00000000-0000-0000-0000-000000000000'::uuid THEN
        RAISE EXCEPTION 'auditoria.identificador_invalido' USING ERRCODE = '22023';
    END IF;
    -- El ID se reserva DESPUÉS del bloqueo. Los saltos por rollback son válidos.
    PERFORM pg_advisory_xact_lock(697008);
    SELECT huella INTO e.huella_anterior FROM auditoria.evento ORDER BY id DESC LIMIT 1;
    e.huella_anterior := coalesce(e.huella_anterior, decode(repeat('00', 32), 'hex'));
    e.id := nextval('auditoria.evento_id_seq'::regclass);
    e.ocurrido_en := p_ocurrido_en;
    e.registrado_en := clock_timestamp();
    e.actor_sistema := usuario = '';
    e.usuario_id := nullif(usuario, '')::uuid;
    e.accion := p_accion; e.entidad := p_entidad;
    e.entidad_id := p_entidad_id; e.expediente_id := p_expediente_id;
    e.resultado := p_resultado;
    e.huella := auditoria.calcular_huella(e);
    INSERT INTO auditoria.evento OVERRIDING SYSTEM VALUE SELECT (e).*;
    RETURN e;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.registrar(timestamptz, text, text, uuid, uuid, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.registrar(timestamptz, text, text, uuid, uuid, text) TO etf_app;

CREATE FUNCTION auditoria.exigir_administracion(p_permitir_sistema boolean) RETURNS void
LANGUAGE plpgsql SET search_path = pg_catalog, auditoria AS $$
DECLARE usuario text := coalesce(current_setting('app.usuario_id', true), '');
        perfiles text := coalesce(current_setting('app.perfiles', true), '');
BEGIN
    IF p_permitir_sistema AND usuario = '' AND perfiles = '' THEN RETURN; END IF;
    IF usuario = '' OR usuario !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
       OR usuario = '00000000-0000-0000-0000-000000000000'
       OR NOT ('administracion' = ANY(string_to_array(perfiles, ','))) THEN
        RAISE EXCEPTION 'auditoria.permiso_denegado' USING ERRCODE = '42501';
    END IF;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.exigir_administracion(boolean) FROM PUBLIC, etf_app;

-- NULL significa tramo válido; un bigint identifica el primer eslabón roto.
-- Comprueba también sellos internos: el punto cero señala un sello vacío roto.
CREATE FUNCTION auditoria.verificar(p_desde bigint, p_hasta bigint) RETURNS bigint
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, auditoria AS $$
DECLARE e auditoria.evento; previa bytea; sello auditoria.sello_diario;
BEGIN
    PERFORM auditoria.exigir_administracion(true);
    IF p_desde IS NULL OR p_hasta IS NULL OR p_desde < 1 OR p_hasta < p_desde THEN
        RAISE EXCEPTION 'auditoria.tramo_invalido' USING ERRCODE = '22023';
    END IF;
    PERFORM pg_advisory_xact_lock(697008);
    SELECT huella INTO previa FROM auditoria.evento WHERE id < p_desde ORDER BY id DESC LIMIT 1;
    previa := coalesce(previa, decode(repeat('00', 32), 'hex'));
    FOR e IN SELECT * FROM auditoria.evento WHERE id BETWEEN p_desde AND p_hasta ORDER BY id LOOP
        IF e.huella_anterior <> previa OR e.huella <> auditoria.calcular_huella(e) THEN RETURN e.id; END IF;
        previa := e.huella;
    END LOOP;
    FOR sello IN SELECT * FROM auditoria.sello_diario ORDER BY dia LOOP
        IF sello.ultimo_evento_id <> coalesce((SELECT max(id) FROM auditoria.evento
            WHERE registrado_en < (sello.dia + 1)::timestamp AT TIME ZONE 'UTC'), 0) THEN
            RETURN sello.ultimo_evento_id;
        END IF;
        IF sello.ultimo_evento_id = 0 THEN
            IF EXISTS (SELECT 1 FROM auditoria.evento WHERE registrado_en < (sello.dia + 1)::timestamp AT TIME ZONE 'UTC') THEN RETURN 0; END IF;
        ELSIF NOT EXISTS (SELECT 1 FROM auditoria.evento WHERE id = sello.ultimo_evento_id AND huella = sello.ultima_huella) THEN
            RETURN sello.ultimo_evento_id;
        END IF;
    END LOOP;
    RETURN NULL;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.verificar(bigint, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.verificar(bigint, bigint) TO etf_app;

CREATE FUNCTION auditoria.exportar(p_desde bigint, p_hasta bigint) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, auditoria AS $$
DECLARE anterior jsonb; final jsonb; eventos jsonb; documento jsonb; roto bigint;
BEGIN
    PERFORM auditoria.exigir_administracion(false);
    roto := auditoria.verificar(p_desde, p_hasta);
    IF roto IS NOT NULL THEN RAISE EXCEPTION 'auditoria.cadena_alterada' USING ERRCODE = '22000'; END IF;
    SELECT jsonb_build_object('id', id, 'huella', encode(huella, 'hex')) INTO anterior
        FROM auditoria.evento WHERE id < p_desde ORDER BY id DESC LIMIT 1;
    anterior := coalesce(anterior, jsonb_build_object('id', 0, 'huella', repeat('00', 32)));
    SELECT jsonb_build_object('id', id, 'huella', encode(huella, 'hex')) INTO final
        FROM auditoria.evento WHERE id BETWEEN p_desde AND p_hasta ORDER BY id DESC LIMIT 1;
    final := coalesce(final, anterior);
    SELECT coalesce(jsonb_agg(jsonb_build_object(
        'id', id, 'ocurrido_en', to_char(ocurrido_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'usuario_id', CASE WHEN actor_sistema THEN 'sistema' ELSE usuario_id::text END,
        'perfiles', perfiles, 'accion', accion, 'entidad', entidad,
        'entidad_id', coalesce(entidad_id::text, ''), 'expediente_id', coalesce(expediente_id::text, ''),
        'resultado', resultado, 'huella_anterior', encode(huella_anterior, 'hex'), 'huella', encode(huella, 'hex')
    ) ORDER BY id), '[]'::jsonb) INTO eventos FROM auditoria.evento WHERE id BETWEEN p_desde AND p_hasta;
    documento := jsonb_build_object('version', 'etf.auditoria.v1', 'algoritmo', 'sha256',
        'canonizacion', 'etf.auditoria.binaria.v1', 'orden', 'id_ascendente',
        'anterior', anterior, 'final', final, 'eventos', eventos);
    IF octet_length(convert_to(documento::text, 'UTF8')) > 67108864 THEN
        RAISE EXCEPTION 'auditoria.exportacion_demasiado_grande' USING ERRCODE = '54000';
    END IF;
    -- El adaptador confirma esta transacción antes de devolver el documento.
    PERFORM auditoria.registrar(clock_timestamp(), 'auditoria.exportar', 'auditoria', NULL, NULL, 'ok');
    RETURN documento;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.exportar(bigint, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.exportar(bigint, bigint) TO etf_app;

-- La fecha corresponde al día UTC de INSERCIÓN, independiente de ocurrido_en.
-- Sólo se cierran días terminados. Un sello publicado jamás se sustituye.
CREATE FUNCTION auditoria.sellar_diario(p_dia date) RETURNS auditoria.sello_diario
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, auditoria AS $$
DECLARE s auditoria.sello_diario; roto bigint;
BEGIN
    PERFORM auditoria.exigir_administracion(true);
    IF p_dia IS NULL OR NOT isfinite(p_dia) OR p_dia >= (clock_timestamp() AT TIME ZONE 'UTC')::date THEN
        RAISE EXCEPTION 'auditoria.dia_invalido' USING ERRCODE = '22023';
    END IF;
    PERFORM pg_advisory_xact_lock(697008);
    roto := auditoria.verificar(1, 9223372036854775807);
    IF roto IS NOT NULL THEN RAISE EXCEPTION 'auditoria.cadena_alterada' USING ERRCODE = '22000'; END IF;
    SELECT * INTO s FROM auditoria.sello_diario WHERE dia = p_dia;
    IF FOUND THEN RETURN s; END IF;
    s.dia := p_dia;
    SELECT id, huella INTO s.ultimo_evento_id, s.ultima_huella FROM auditoria.evento
        WHERE registrado_en < (p_dia + 1)::timestamp AT TIME ZONE 'UTC' ORDER BY id DESC LIMIT 1;
    s.ultimo_evento_id := coalesce(s.ultimo_evento_id, 0);
    s.ultima_huella := coalesce(s.ultima_huella, decode(repeat('00', 32), 'hex'));
    INSERT INTO auditoria.sello_diario SELECT (s).*;
    RETURN s;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.sellar_diario(date) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.sellar_diario(date) TO etf_app;
