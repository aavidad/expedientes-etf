SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regprocedure('identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb)') IS NULL
       OR to_regclass('identidad.alta_administrador_inicial') IS NULL THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;
-- Conserva las políticas de lectura y la ausencia de DML directo para etf_app.
-- Sólo cambia el contrato de escritura; el alta inicial sigue siendo separada.
DROP FUNCTION identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb);
CREATE FUNCTION identidad.cambiar_perfiles(persona uuid, esperada bigint, nuevo_estado text, concesiones jsonb, equipo jsonb)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, identidad AS $$
DECLARE actor uuid; ahora timestamptz; actual bigint; concesion record; profesional jsonb;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0));
    IF NOT identidad.es_administrador() THEN RAISE EXCEPTION 'identidad.acceso_denegado'; END IF;
    ahora := clock_timestamp();
    actor := nullif(current_setting('app.usuario_id',true),'')::uuid;
    -- Un rechazo se confirma con su auditoría; lanzar una excepción aquí
    -- revertiría también el apunte. El llamante debe comprobar el booleano.
    IF persona = actor THEN
        PERFORM auditoria.registrar(ahora,'identidad.perfiles_cambiar','identidad.usuario',persona,NULL,'denegado');
        RETURN false;
    END IF;
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
    RETURN true;
END $$;
REVOKE EXECUTE ON FUNCTION identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION identidad.cambiar_perfiles(uuid,bigint,text,jsonb,jsonb) TO etf_app;
