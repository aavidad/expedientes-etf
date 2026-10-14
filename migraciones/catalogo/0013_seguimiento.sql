SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.parametro') IS NULL OR to_regclass('catalogo.lista_opcion') IS NULL OR to_regclass('catalogo.fase_duracion') IS NULL
 OR EXISTS(SELECT 1 FROM catalogo.version WHERE ambito IN ('plazos','listas') AND estado='publicada'
  AND vigente_desde>=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date)
 OR to_regprocedure('catalogo.fin_seguimiento(date,uuid,date)') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- Se exige que las versiones previas hayan comenzado antes de hoy. Así las
-- reglas nuevas estarán vigentes al instalar, sin alterar contenido publicado.
-- Se publica una versión nueva de cada ámbito, sin modificar contenido previo.
-- Otros plazos, opciones y referencias se conservan íntegros.
DO $$ DECLARE ambito_nuevo text; anterior catalogo.version; nueva uuid; desde date; BEGIN
 FOREACH ambito_nuevo IN ARRAY ARRAY['plazos','listas'] LOOP
  SELECT * INTO anterior FROM catalogo.version WHERE ambito=ambito_nuevo AND estado='publicada' ORDER BY numero DESC LIMIT 1;
  desde:=CASE WHEN anterior.id IS NULL THEN DATE '2026-01-01' ELSE (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date END;
  INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
  VALUES(ambito_nuevo,coalesce(anterior.numero,0)+1,desde,'catalogo.seguimiento_inicial','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
  IF ambito_nuevo='plazos' THEN
   INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
   INSERT INTO catalogo.fase_duracion SELECT nueva,subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=anterior.id;
   INSERT INTO catalogo.parametro VALUES(nueva,'seguimiento_tras_cese',12,'meses'),(nueva,'aviso_fin_seguimiento',30,'dias') ON CONFLICT DO NOTHING;
  ELSE
   INSERT INTO catalogo.lista_opcion SELECT nueva,lista,codigo,orden,admite_texto,activa FROM catalogo.lista_opcion WHERE version_id=anterior.id;
   INSERT INTO catalogo.lista_opcion SELECT nueva,'fin_seguimiento','CAMBIO_SITUACION',coalesce(max(orden),0)+1,false,true FROM catalogo.lista_opcion WHERE version_id=nueva AND lista='fin_seguimiento' ON CONFLICT(version_id,lista,codigo) DO NOTHING;
  END IF;
  INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM catalogo.version_textos WHERE version_id=anterior.id;
  PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
 END LOOP;
END $$;
-- Puerto SQL de sólo lectura para la segunda barrera del seguimiento. Se usa
-- una regla publicada y vigente en la fecha del acto, sin abrir escritura.
CREATE FUNCTION catalogo.fin_seguimiento(desde date,regla uuid,fecha_acto date) RETURNS date
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,catalogo AS $$ DECLARE meses integer; fin date; BEGIN
 SELECT p.valor_entero INTO meses FROM catalogo.version v JOIN catalogo.parametro p ON p.version_id=v.id
 WHERE v.id=regla AND v.ambito='plazos' AND v.estado='publicada'
 AND daterange(v.vigente_desde,v.vigente_hasta,'[)') @> fecha_acto
 AND p.clave='seguimiento_tras_cese' AND p.unidad='meses' AND p.valor_entero BETWEEN 1 AND 1200;
 IF meses IS NULL OR desde IS NULL OR NOT isfinite(desde) THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 fin:=(desde+make_interval(months=>meses))::date;
 IF extract(year FROM fin)>9999 THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 RETURN fin;
END $$;
REVOKE EXECUTE ON FUNCTION catalogo.fin_seguimiento(date,uuid,date) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION catalogo.fin_seguimiento(date,uuid,date) TO etf_app;
