SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('identidad.usuario') IS NULL OR to_regclass('catalogo.equipo') IS NULL
       OR to_regclass('identidad.usuario_perfil') IS NOT NULL THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
SET LOCAL search_path = pg_catalog, identidad, catalogo;
CREATE TABLE identidad.usuario_perfil (
    usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
    perfil text NOT NULL CHECK (perfil IN ('coordinacion','profesional_etf','consulta','administracion')),
    desde timestamptz NOT NULL CHECK (isfinite(desde)),
    hasta timestamptz CHECK (isfinite(hasta) AND hasta >= desde),
    concedido_por uuid NOT NULL REFERENCES identidad.usuario(id),
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    EXCLUDE USING gist (usuario_id WITH =, perfil WITH =, tstzrange(desde,hasta,'[)') WITH &&)
);
CREATE TABLE identidad.usuario_equipo (
    usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
    equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),
    puesto text NOT NULL CHECK (puesto IN ('ts','es','ps')),
    desde timestamptz NOT NULL CHECK (isfinite(desde)),
    hasta timestamptz CHECK (isfinite(hasta) AND hasta >= desde),
    asignado_por uuid NOT NULL REFERENCES identidad.usuario(id),
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    EXCLUDE USING gist (usuario_id WITH =, tstzrange(desde,hasta,'[)') WITH &&)
);
CREATE UNIQUE INDEX usuario_equipo_unico ON identidad.usuario_equipo(usuario_id) WHERE hasta IS NULL;
CREATE TABLE identidad.alta_administrador_inicial (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    usuario_id uuid NOT NULL UNIQUE REFERENCES identidad.usuario(id),
    object_guid_ad uuid NOT NULL UNIQUE,
    realizado_en timestamptz NOT NULL
);

-- El permiso vigente se comprueba también en los datos del servidor. La función
-- no depende de las políticas de etf_app, para evitar recursión al leer perfiles.
CREATE FUNCTION identidad.es_administrador() RETURNS boolean
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
    SELECT coalesce('administracion' = ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
      AND EXISTS (SELECT FROM identidad.usuario u JOIN identidad.usuario_perfil p ON p.usuario_id=u.id
        WHERE u.id::text=nullif(current_setting('app.usuario_id',true),'') AND u.estado='activo'
          AND p.perfil='administracion' AND p.desde <= clock_timestamp()
          AND (p.hasta IS NULL OR clock_timestamp() < p.hasta))
$$;
REVOKE EXECUTE ON FUNCTION identidad.es_administrador() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.es_administrador() TO etf_app;
CREATE POLICY usuario_administracion ON identidad.usuario FOR SELECT TO etf_app USING (identidad.es_administrador());
DO $$ DECLARE tabla text; BEGIN
    FOREACH tabla IN ARRAY ARRAY['usuario_perfil','usuario_equipo','alta_administrador_inicial'] LOOP
        EXECUTE format('ALTER TABLE identidad.%I ENABLE ROW LEVEL SECURITY',tabla);
        EXECUTE format('ALTER TABLE identidad.%I FORCE ROW LEVEL SECURITY',tabla);
        EXECUTE format('CREATE POLICY propietario ON identidad.%I TO etf_propietario USING (true) WITH CHECK (true)',tabla);
        EXECUTE format('REVOKE ALL ON identidad.%I FROM PUBLIC,etf_app,etf_copia',tabla);
        EXECUTE format('GRANT SELECT ON identidad.%I TO etf_copia',tabla);
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY['usuario_perfil','usuario_equipo'] LOOP
        EXECUTE format('CREATE POLICY lectura ON identidad.%I FOR SELECT TO etf_app USING (usuario_id::text = nullif(current_setting(''app.usuario_id'',true),'''') OR identidad.es_administrador())',tabla);
        EXECUTE format('GRANT SELECT ON identidad.%I TO etf_app',tabla);
    END LOOP;
END $$;

CREATE FUNCTION identidad.conceder_administrador_inicial(persona uuid, guid uuid, cuenta text, ahora timestamptz)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
    IF persona::text IS DISTINCT FROM current_setting('app.usuario_id',true)
       OR NOT EXISTS (SELECT FROM identidad.usuario WHERE id=persona AND object_guid_ad=guid AND cuenta_ad=cuenta) THEN
        RAISE EXCEPTION 'identidad.acceso_denegado';
    END IF;
    IF EXISTS (SELECT FROM identidad.alta_administrador_inicial) THEN RETURN; END IF;
    IF NOT EXISTS (SELECT FROM identidad.usuario WHERE id=persona AND estado='pendiente_alta') THEN RETURN; END IF;
    INSERT INTO identidad.alta_administrador_inicial VALUES (true,persona,guid,ahora);
    INSERT INTO identidad.usuario_perfil VALUES (persona,'administracion',ahora,NULL,persona);
    UPDATE identidad.usuario SET estado='activo',version=version+1 WHERE id=persona;
    PERFORM set_config('app.perfiles','administracion',true);
    PERFORM auditoria.registrar(ahora,'identidad.administrador_inicial','identidad.usuario',persona,NULL,'ok');
END $$;
REVOKE EXECUTE ON FUNCTION identidad.conceder_administrador_inicial(uuid,uuid,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.conceder_administrador_inicial(uuid,uuid,text,timestamptz) TO etf_app;

-- El cambio sustituye la configuración futura conservando todas las filas
-- retiradas. Los intervalos ya vigentes nunca se pueden conceder retroactivamente.
CREATE FUNCTION identidad.cambiar_perfiles(persona uuid, esperada bigint, nuevo_estado text, concesiones jsonb, equipo jsonb)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE actor uuid; ahora timestamptz; actual bigint; concesion record; profesional jsonb;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
    IF NOT identidad.es_administrador() THEN RAISE EXCEPTION 'identidad.acceso_denegado'; END IF;
    ahora := clock_timestamp();
    actor := nullif(current_setting('app.usuario_id',true),'')::uuid;
    SELECT version INTO actual FROM identidad.usuario WHERE id=persona FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION 'identidad.usuario_no_disponible'; END IF;
    IF actual <> esperada THEN RAISE EXCEPTION 'identidad.conflicto'; END IF;
    IF nuevo_estado NOT IN ('activo','baja') OR jsonb_typeof(concesiones) IS DISTINCT FROM 'array'
       OR jsonb_array_length(concesiones)>4 THEN RAISE EXCEPTION 'identidad.perfiles_invalidos'; END IF;
    IF EXISTS (SELECT FROM jsonb_to_recordset(concesiones) AS x(perfil text,desde timestamptz,hasta timestamptz)
       WHERE perfil IS NULL OR perfil NOT IN ('coordinacion','profesional_etf','consulta','administracion')
         OR desde IS NULL OR NOT isfinite(desde) OR (hasta IS NOT NULL AND (NOT isfinite(hasta) OR hasta <= greatest(desde,ahora))))
       OR (SELECT count(*)<>count(DISTINCT x->>'perfil') FROM jsonb_array_elements(concesiones) x) THEN
       RAISE EXCEPTION 'identidad.perfiles_invalidos';
    END IF;
    SELECT x INTO profesional FROM jsonb_array_elements(concesiones) x WHERE x->>'perfil'='profesional_etf';
    equipo := nullif(equipo,'null'::jsonb);
    IF (profesional IS NULL) <> (equipo IS NULL)
       OR (nuevo_estado='baja' AND (jsonb_array_length(concesiones)>0 OR equipo IS NOT NULL)) THEN
       RAISE EXCEPTION 'identidad.perfiles_invalidos';
    END IF;
    IF equipo IS NOT NULL AND (jsonb_typeof(equipo) IS DISTINCT FROM 'object' OR equipo->>'equipo_id' IS NULL
       OR equipo->>'puesto' IS NULL OR equipo->>'puesto' NOT IN ('ts','es','ps')
       OR (equipo->>'desde')::timestamptz IS DISTINCT FROM (profesional->>'desde')::timestamptz
       OR (equipo->>'hasta')::timestamptz IS DISTINCT FROM (profesional->>'hasta')::timestamptz) THEN
       RAISE EXCEPTION 'identidad.perfiles_invalidos';
    END IF;
    -- Sólo se conserva una adscripción idéntica y su concesión profesional.
    IF NOT EXISTS (SELECT FROM identidad.usuario_equipo e WHERE e.usuario_id=persona AND (e.hasta IS NULL OR e.hasta>ahora)
       AND e.equipo_id=(equipo->>'equipo_id')::uuid AND e.puesto=equipo->>'puesto'
       AND e.desde=(equipo->>'desde')::timestamptz AND e.hasta IS NOT DISTINCT FROM (equipo->>'hasta')::timestamptz) THEN
       UPDATE identidad.usuario_equipo SET hasta=greatest(ahora,desde) WHERE usuario_id=persona AND (hasta IS NULL OR hasta>ahora);
       UPDATE identidad.usuario_perfil SET hasta=greatest(ahora,desde) WHERE usuario_id=persona AND perfil='profesional_etf' AND (hasta IS NULL OR hasta>ahora);
       IF equipo IS NOT NULL THEN
           INSERT INTO identidad.usuario_equipo VALUES (persona,(equipo->>'equipo_id')::uuid,equipo->>'puesto',greatest(ahora,(equipo->>'desde')::timestamptz),(equipo->>'hasta')::timestamptz,actor);
       END IF;
    END IF;
    UPDATE identidad.usuario_perfil p SET hasta=greatest(ahora,p.desde)
      WHERE p.usuario_id=persona AND (p.hasta IS NULL OR p.hasta>ahora)
        AND NOT EXISTS (SELECT FROM jsonb_to_recordset(concesiones) x(perfil text,desde timestamptz,hasta timestamptz)
          WHERE x.perfil=p.perfil AND x.desde=p.desde AND x.hasta IS NOT DISTINCT FROM p.hasta);
    FOR concesion IN SELECT * FROM jsonb_to_recordset(concesiones) x(perfil text,desde timestamptz,hasta timestamptz) LOOP
      IF NOT EXISTS (SELECT FROM identidad.usuario_perfil v WHERE v.usuario_id=persona AND v.perfil=concesion.perfil
        AND v.desde=concesion.desde AND v.hasta IS NOT DISTINCT FROM concesion.hasta AND (v.hasta IS NULL OR v.hasta>ahora)) THEN
        INSERT INTO identidad.usuario_perfil VALUES (persona,concesion.perfil,greatest(ahora,concesion.desde),concesion.hasta,actor);
      END IF;
    END LOOP;
    UPDATE identidad.usuario SET estado=nuevo_estado,version=version+1 WHERE id=persona;
    PERFORM auditoria.registrar(ahora,'identidad.perfiles_cambiar','identidad.usuario',persona,NULL,'ok');
END $$;
REVOKE EXECUTE ON FUNCTION identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb) TO etf_app;
