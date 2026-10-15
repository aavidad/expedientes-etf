SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regprocedure('auditoria.historial_expediente(uuid)') IS NULL
 OR to_regprocedure('auditoria.delimitar_exportacion(timestamptz,timestamptz)') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

CREATE INDEX evento_ocurrido ON auditoria.evento(ocurrido_en,id);

-- El tramo incluye los eslabones intermedios aunque no cumplan los filtros.
-- Los identificadores crecen; ocurrido_en puede tener fechas retrospectivas.
CREATE FUNCTION auditoria.delimitar_exportacion(p_desde timestamptz,p_hasta timestamptz)
 RETURNS TABLE(anterior_id bigint,anterior_huella text,final_id bigint,final_huella text)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,auditoria AS $$
DECLARE primero bigint; ultimo bigint; cantidad bigint;
BEGIN
 PERFORM auditoria.exigir_administracion(false);
 IF p_desde IS NULL OR p_hasta IS NULL OR NOT isfinite(p_desde) OR NOT isfinite(p_hasta)
 OR p_hasta<=p_desde OR p_hasta-p_desde>interval '31 days' THEN
  RAISE EXCEPTION 'auditoria.fechas_invalidas' USING ERRCODE='22023';
 END IF;
 SELECT min(e.id),max(e.id) INTO primero,ultimo FROM auditoria.evento e
 WHERE e.ocurrido_en>=p_desde AND e.ocurrido_en<p_hasta;
 IF primero IS NULL THEN
  RETURN QUERY SELECT coalesce(e.id,0),coalesce(encode(e.huella,'hex'),repeat('00',32)),
                      coalesce(e.id,0),coalesce(encode(e.huella,'hex'),repeat('00',32))
    FROM (SELECT 1) n LEFT JOIN LATERAL(SELECT id,huella FROM auditoria.evento ORDER BY id DESC LIMIT 1)e ON true;
  RETURN;
 END IF;
 SELECT count(*) INTO cantidad FROM(SELECT id FROM auditoria.evento WHERE id BETWEEN primero AND ultimo LIMIT 100001)e;
 IF cantidad>100000 THEN RAISE EXCEPTION 'auditoria.exportacion_demasiado_grande' USING ERRCODE='54000'; END IF;
 RETURN QUERY SELECT coalesce(p.id,0),coalesce(encode(p.huella,'hex'),repeat('00',32)),u.id,encode(u.huella,'hex')
 FROM auditoria.evento u LEFT JOIN LATERAL(SELECT id,huella FROM auditoria.evento WHERE id<primero ORDER BY id DESC LIMIT 1)p ON true
 WHERE u.id=ultimo;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.delimitar_exportacion(timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.delimitar_exportacion(timestamptz,timestamptz) TO etf_app;

CREATE FUNCTION auditoria.pagina_exportacion(p_despues bigint,p_final bigint)
 RETURNS TABLE(id bigint,ocurrido_en timestamptz,usuario_id text,perfiles text[],accion text,entidad text,
 entidad_id text,expediente_id text,resultado text,huella_anterior text,huella text)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog,auditoria AS $$
BEGIN
 PERFORM auditoria.exigir_administracion(false);
 IF p_despues IS NULL OR p_final IS NULL OR p_despues<0 OR p_final<p_despues THEN
  RAISE EXCEPTION 'auditoria.tramo_invalido' USING ERRCODE='22023';
 END IF;
 RETURN QUERY SELECT e.id,e.ocurrido_en,CASE WHEN e.actor_sistema THEN 'sistema' ELSE e.usuario_id::text END,
 e.perfiles,e.accion,e.entidad,coalesce(e.entidad_id::text,''),coalesce(e.expediente_id::text,''),e.resultado,
 encode(e.huella_anterior,'hex'),encode(e.huella,'hex') FROM auditoria.evento e
 WHERE e.id>p_despues AND e.id<=p_final ORDER BY e.id LIMIT 256;
END $$;
REVOKE EXECUTE ON FUNCTION auditoria.pagina_exportacion(bigint,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auditoria.pagina_exportacion(bigint,bigint) TO etf_app;
