SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.fase_duracion') IS NOT NULL
 OR to_regclass('catalogo.parametro') IS NULL
 OR EXISTS(SELECT 1 FROM catalogo.version WHERE ambito='plazos' AND estado='publicada'
 AND vigente_desde>=current_date) THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
CREATE TABLE catalogo.fase_duracion (
 version_id uuid NOT NULL REFERENCES catalogo.version(id),
 subprograma text NOT NULL CHECK(subprograma IN ('RIESGO_SOCIAL','REUNIFICACION')),
 fase text NOT NULL CHECK(fase IN ('RECEPCION','EVALUACION','INTERVENCION','CIERRE_DEVOLUCION','SEGUIMIENTO')),
 dias_orientativos integer NOT NULL CHECK(dias_orientativos>=0),
 dias_ampliable integer CHECK(dias_ampliable>=dias_orientativos),
 PRIMARY KEY(version_id,subprograma,fase)
);
ALTER TABLE catalogo.fase_duracion ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.fase_duracion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON catalogo.fase_duracion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON catalogo.fase_duracion TO etf_app,etf_copia;
CREATE POLICY propietario ON catalogo.fase_duracion TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY lectura ON catalogo.fase_duracion FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['administracion','coordinacion','profesional_etf','sistema']);
CREATE TRIGGER proteger_fase BEFORE INSERT OR UPDATE OR DELETE ON catalogo.fase_duracion
 FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('plazos');
CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.fase_duracion
 FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();
-- Conserva los parámetros publicados si la instalación ya los había configurado.
DO $$ DECLARE anterior catalogo.version; nueva uuid; BEGIN
 SELECT * INTO anterior FROM catalogo.version WHERE ambito='plazos' AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES('plazos',coalesce(anterior.numero,0)+1,
 CASE WHEN anterior.id IS NULL THEN DATE '2026-01-01' ELSE current_date END,
 'catalogo.avisos_iniciales','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
 INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
 INSERT INTO catalogo.parametro VALUES(nueva,'plazo_consentimiento',30,'dias_naturales') ON CONFLICT DO NOTHING;
 -- Duraciones orientativas en días: un mes orientativo equivale a 30 días.
 INSERT INTO catalogo.fase_duracion SELECT nueva,datos.* FROM (VALUES
 ('RIESGO_SOCIAL','RECEPCION',30,NULL::integer),('REUNIFICACION','RECEPCION',30,NULL::integer),
 ('RIESGO_SOCIAL','EVALUACION',90,NULL::integer),('REUNIFICACION','EVALUACION',45,NULL::integer),
 ('RIESGO_SOCIAL','INTERVENCION',450,540),('REUNIFICACION','INTERVENCION',180,360),
 ('RIESGO_SOCIAL','CIERRE_DEVOLUCION',45,NULL::integer),('REUNIFICACION','CIERRE_DEVOLUCION',30,NULL::integer),
 ('RIESGO_SOCIAL','SEGUIMIENTO',120,NULL::integer),('REUNIFICACION','SEGUIMIENTO',120,NULL::integer)
 ) datos(subprograma,fase,dias_orientativos,dias_ampliable);
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM catalogo.version_textos WHERE version_id=anterior.id;
 PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
END $$;
CREATE OR REPLACE FUNCTION catalogo.editar(p_ambito text,p_esperada integer,p_desde date,p_cambio jsonb) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,catalogo AS $$
DECLARE anterior catalogo.version; nueva uuid; actor uuid;
BEGIN
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501'; END IF;
 IF p_ambito NOT IN ('listas','plazos','tramos') OR p_esperada<0 THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
 PERFORM pg_advisory_xact_lock(697012,hashtext(p_ambito));
 SELECT * INTO anterior FROM catalogo.version WHERE ambito=p_ambito AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 IF coalesce(anterior.numero,0)<>p_esperada THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=coalesce(anterior.numero,0)::text; END IF;
 IF anterior.id IS NOT NULL AND p_desde<=anterior.vigente_desde THEN RAISE EXCEPTION 'catalogo.vigencia_invalida'; END IF;
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES(p_ambito,p_esperada+1,p_desde,'catalogo.edicion',actor) RETURNING id INTO nueva;
 IF p_ambito='listas' THEN
  INSERT INTO catalogo.lista_opcion SELECT nueva,lista,codigo,orden,admite_texto,activa FROM catalogo.lista_opcion WHERE version_id=anterior.id;
  INSERT INTO catalogo.lista_opcion VALUES(nueva,p_cambio->>'lista',p_cambio->>'codigo',(p_cambio->>'orden')::integer,(p_cambio->>'admite_texto')::boolean,(p_cambio->>'activa')::boolean)
   ON CONFLICT(version_id,lista,codigo) DO UPDATE SET orden=EXCLUDED.orden,admite_texto=EXCLUDED.admite_texto,activa=EXCLUDED.activa;
 ELSIF p_ambito='plazos' THEN
  INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
  INSERT INTO catalogo.fase_duracion SELECT nueva,subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=anterior.id;
  IF p_cambio ? 'fase' THEN
   INSERT INTO catalogo.fase_duracion VALUES(nueva,p_cambio->>'subprograma',p_cambio->>'fase',(p_cambio->>'dias_orientativos')::integer,(p_cambio->>'dias_ampliable')::integer)
   ON CONFLICT(version_id,subprograma,fase) DO UPDATE SET dias_orientativos=EXCLUDED.dias_orientativos,dias_ampliable=EXCLUDED.dias_ampliable;
  ELSE
  INSERT INTO catalogo.parametro VALUES(nueva,p_cambio->>'clave',(p_cambio->>'valor')::integer,p_cambio->>'unidad') ON CONFLICT(version_id,clave) DO UPDATE SET valor_entero=EXCLUDED.valor_entero,unidad=EXCLUDED.unidad;
  END IF;
 ELSE
  INSERT INTO catalogo.tramo SELECT nueva,tipo,orden,desde,hasta,unidad FROM catalogo.tramo WHERE version_id=anterior.id;
  INSERT INTO catalogo.tramo VALUES(nueva,p_cambio->>'tipo',(p_cambio->>'orden')::integer,(p_cambio->>'desde')::numeric,(p_cambio->>'hasta')::numeric,p_cambio->>'unidad') ON CONFLICT(version_id,tipo,orden) DO UPDATE SET desde=EXCLUDED.desde,hasta=EXCLUDED.hasta,unidad=EXCLUDED.unidad;
 END IF;
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM paquete.referencia_activa WHERE tipo='catalogo';
 IF p_ambito='listas' AND (SELECT count(*) FROM catalogo.version_textos WHERE version_id=nueva)<>2 THEN RAISE EXCEPTION 'paquete.publicacion_invalida'; END IF;
 PERFORM catalogo.publicar(nueva,actor);
 RETURN nueva;
END $$;
REVOKE ALL ON FUNCTION catalogo.editar(text,integer,date,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalogo.editar(text,integer,date,jsonb) TO etf_app;
