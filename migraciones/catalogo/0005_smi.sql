DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador' OR to_regclass('catalogo.firmante') IS NULL OR to_regclass('catalogo.smi') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
SET LOCAL lock_timeout='3s';
SET LOCAL statement_timeout='30s';
SET LOCAL search_path=pg_catalog,catalogo;

CREATE TABLE catalogo.smi (
 anio smallint PRIMARY KEY CHECK(anio BETWEEN 2026 AND 2040),
 importe_mensual_centimos bigint CHECK(importe_mensual_centimos IS NULL OR importe_mensual_centimos>0),
 importe_anual_centimos bigint GENERATED ALWAYS AS (importe_mensual_centimos*14) STORED,
 version integer NOT NULL DEFAULT 0 CHECK(version>=0),
 modificado_por uuid,
 modificado_en timestamptz
);
INSERT INTO catalogo.smi(anio) SELECT generate_series(2026,2040)::smallint;
CREATE TABLE catalogo.smi_historial (
 anio smallint NOT NULL,
 version integer NOT NULL CHECK(version>0),
 importe_mensual_centimos bigint NOT NULL CHECK(importe_mensual_centimos>0),
 modificado_por uuid NOT NULL,
 modificado_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(anio,version)
);
CREATE FUNCTION catalogo.impedir_cambio_smi_historial() RETURNS trigger
 LANGUAGE plpgsql SET search_path=pg_catalog,catalogo AS $$
 BEGIN RAISE EXCEPTION 'catalogo.version_inmutable'; END $$;
CREATE TRIGGER smi_historial_inmutable BEFORE UPDATE OR DELETE ON catalogo.smi_historial FOR EACH ROW EXECUTE FUNCTION catalogo.impedir_cambio_smi_historial();
ALTER TABLE catalogo.smi ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.smi FORCE ROW LEVEL SECURITY;
ALTER TABLE catalogo.smi_historial ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.smi_historial FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON catalogo.smi TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY propietario ON catalogo.smi_historial TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY lectura_coordinacion ON catalogo.smi FOR SELECT TO etf_app USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY escritura_smi ON catalogo.smi TO etf_app USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))) WITH CHECK(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY lectura_smi_historial ON catalogo.smi_historial FOR SELECT TO etf_app USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
REVOKE ALL ON catalogo.smi,catalogo.smi_historial FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON catalogo.smi,catalogo.smi_historial TO etf_app;
CREATE FUNCTION catalogo.guardar_smi(p_anio smallint,p_importe bigint,p_esperada integer) RETURNS integer
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,catalogo AS $$
 DECLARE actor uuid; actual integer;
 BEGIN
  actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
  IF actor IS NULL OR NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN RAISE EXCEPTION 'catalogo.acceso_denegado' USING ERRCODE='42501'; END IF;
  IF p_anio NOT BETWEEN 2026 AND 2040 OR p_importe<=0 OR p_esperada<0 THEN RAISE EXCEPTION 'catalogo.edicion_invalida'; END IF;
  SELECT version INTO actual FROM catalogo.smi WHERE anio=p_anio FOR UPDATE;
  IF actual IS NULL OR actual<>p_esperada THEN RAISE EXCEPTION 'catalogo.conflicto' USING ERRCODE='40001',DETAIL=coalesce(actual,0)::text; END IF;
  UPDATE catalogo.smi SET importe_mensual_centimos=p_importe,version=version+1,modificado_por=actor,modificado_en=clock_timestamp() WHERE anio=p_anio;
  INSERT INTO catalogo.smi_historial(anio,version,importe_mensual_centimos,modificado_por)
   VALUES(p_anio,p_esperada+1,p_importe,actor);
  RETURN p_esperada+1;
 END $$;
REVOKE ALL ON FUNCTION catalogo.impedir_cambio_smi_historial(),catalogo.guardar_smi(smallint,bigint,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalogo.guardar_smi(smallint,bigint,integer) TO etf_app;
DO $$ BEGIN
 CREATE TRIGGER impedir_truncado_smi BEFORE TRUNCATE ON catalogo.smi FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();
 CREATE TRIGGER impedir_truncado_smi_historial BEFORE TRUNCATE ON catalogo.smi_historial FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();
END $$;
