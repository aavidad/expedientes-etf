SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('documento.archivo_plantilla') IS NULL
 OR to_regclass('documento.activacion_plantilla') IS NOT NULL THEN
 RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
ALTER TABLE documento.plantilla ADD COLUMN antivirus text NOT NULL DEFAULT 'no_consta' CHECK (antivirus IN ('no_consta','limpio'));
CREATE TABLE documento.activacion_plantilla (
 tipo_documento text NOT NULL,
 idioma text NOT NULL,
 revision bigint NOT NULL CHECK (revision>0),
 plantilla_id uuid NOT NULL,
 activada_por uuid NOT NULL REFERENCES identidad.usuario(id),
 activada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tipo_documento,idioma,revision),
 FOREIGN KEY(plantilla_id,tipo_documento,idioma) REFERENCES documento.plantilla(id,tipo_documento,idioma)
);
CREATE INDEX activacion_por_plantilla ON documento.activacion_plantilla(plantilla_id,activada_en DESC);
ALTER TABLE documento.activacion_plantilla ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.activacion_plantilla FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.activacion_plantilla FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON documento.activacion_plantilla TO etf_app,etf_copia;
CREATE POLICY activacion_propietario ON documento.activacion_plantilla TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY activacion_administracion ON documento.activacion_plantilla FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE TRIGGER activacion_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON documento.activacion_plantilla
 FOR EACH STATEMENT EXECUTE FUNCTION documento.inmutable();
CREATE OR REPLACE FUNCTION documento.publicar_plantilla(p_id uuid,p_tipo text,p_idioma text,p_contenido text,p_ruta text,p_huella text,p_previa bigint)
RETURNS TABLE(aplicada boolean,revision bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,documento AS $$
DECLARE usuario uuid; previa bigint; anterior uuid; rol text;
BEGIN
    usuario:=nullif(current_setting('app.usuario_id',true),'')::uuid;
    IF usuario IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
        RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501';
    END IF;
    IF p_id IS NULL OR p_id='00000000-0000-0000-0000-000000000000'::uuid
       OR p_tipo IS NULL OR p_idioma IS NULL OR p_contenido IS NULL OR p_ruta IS NULL
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_previa IS NULL OR p_previa<1 OR p_previa>=9007199254740991
       OR p_ruta<>'plantillas/'||p_tipo||'/.versiones/'||p_id::text||'/v'||p_contenido||'.docx' THEN
        RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('documento.plantilla:'||p_tipo||':'||p_idioma,0));
    SELECT r.version,r.plantilla_id,p.rol_firmante INTO previa,anterior,rol
      FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
      WHERE r.tipo_documento=p_tipo AND r.idioma=p_idioma FOR UPDATE OF r;
    IF previa IS NULL THEN
        RAISE EXCEPTION 'documento.plantilla_no_disponible' USING ERRCODE='22023';
    END IF;
    IF previa<>p_previa THEN RETURN QUERY SELECT false,previa; RETURN; END IF;
    INSERT INTO documento.plantilla(id,tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,subida_por,comprobacion,antivirus)
      VALUES(p_id,p_tipo,p_idioma,p_contenido,p_ruta,decode(p_huella,'hex'),rol,current_date,usuario,'motor_docx_1','limpio');
    INSERT INTO documento.activacion_plantilla(tipo_documento,idioma,revision,plantilla_id,activada_por)
 VALUES(p_tipo,p_idioma,previa+1,p_id,usuario);
    UPDATE documento.referencia_activa SET plantilla_id=p_id,version=previa+1 WHERE tipo_documento=p_tipo AND idioma=p_idioma;
    PERFORM auditoria.registrar(clock_timestamp(),'documento.plantilla_activar','documento',p_id,NULL,'ok');
    RETURN QUERY SELECT true,previa+1;
END $$;
CREATE FUNCTION documento.guardar_version_plantilla(p_id uuid,p_tipo text,p_idioma text,p_contenido text,p_ruta text,p_huella text,p_previa bigint)
RETURNS TABLE(aplicada boolean,revision bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,documento AS $$
DECLARE usuario uuid; previa bigint; anterior uuid; rol text;
BEGIN
    usuario:=nullif(current_setting('app.usuario_id',true),'')::uuid;
    IF usuario IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
        RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501';
    END IF;
    IF p_id IS NULL OR p_id='00000000-0000-0000-0000-000000000000'::uuid
       OR p_tipo IS NULL OR p_idioma IS NULL OR p_contenido IS NULL OR p_ruta IS NULL
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_previa IS NULL OR p_previa<1 OR p_previa>=9007199254740991
       OR p_ruta<>'plantillas/'||p_tipo||'/.versiones/'||p_id::text||'/v'||p_contenido||'.docx' THEN
        RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended('documento.plantilla:'||p_tipo||':'||p_idioma,0));
    SELECT r.version,r.plantilla_id,p.rol_firmante INTO previa,anterior,rol
      FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
      WHERE r.tipo_documento=p_tipo AND r.idioma=p_idioma FOR UPDATE OF r;
    IF previa IS NULL THEN
        RAISE EXCEPTION 'documento.plantilla_no_disponible' USING ERRCODE='22023';
    END IF;
    IF previa<>p_previa THEN RETURN QUERY SELECT false,previa; RETURN; END IF;
    INSERT INTO documento.plantilla(id,tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,subida_por,comprobacion,antivirus)
      VALUES(p_id,p_tipo,p_idioma,p_contenido,p_ruta,decode(p_huella,'hex'),rol,current_date,usuario,'motor_docx_1','limpio');

    PERFORM auditoria.registrar(clock_timestamp(),'documento.plantilla_subir','documento',p_id,NULL,'ok');
    RETURN QUERY SELECT true,previa;
END $$;
REVOKE EXECUTE ON FUNCTION documento.guardar_version_plantilla(uuid,text,text,text,text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION documento.guardar_version_plantilla(uuid,text,text,text,text,text,bigint) TO etf_app;
CREATE FUNCTION documento.activar_version_plantilla(p_id uuid,p_previa bigint)
RETURNS TABLE(aplicada boolean,revision bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,documento AS $$
DECLARE usuario uuid; previa bigint; tipo text; lengua text; anterior uuid;
BEGIN
 usuario:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 IF usuario IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
 RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501'; END IF;
 IF p_id IS NULL OR p_previa IS NULL OR p_previa<1 OR p_previa>=9007199254740991 THEN
 RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='22023'; END IF;
 SELECT tipo_documento,idioma INTO tipo,lengua FROM documento.plantilla WHERE id=p_id;
 IF tipo IS NULL THEN RAISE EXCEPTION 'documento.plantilla_no_disponible' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('documento.plantilla:'||tipo||':'||lengua,0));
 SELECT r.version,r.plantilla_id INTO previa,anterior FROM documento.referencia_activa r
 WHERE r.tipo_documento=tipo AND r.idioma=lengua FOR UPDATE;
 IF previa IS NULL THEN RAISE EXCEPTION 'documento.plantilla_no_disponible' USING ERRCODE='22023'; END IF;
 IF previa<>p_previa THEN RETURN QUERY SELECT false,previa; RETURN; END IF;
 IF anterior=p_id THEN RETURN QUERY SELECT true,previa; RETURN; END IF;
 UPDATE documento.referencia_activa SET plantilla_id=p_id,version=previa+1 WHERE tipo_documento=tipo AND idioma=lengua;
 INSERT INTO documento.activacion_plantilla(tipo_documento,idioma,revision,plantilla_id,activada_por)
 VALUES(tipo,lengua,previa+1,p_id,usuario);
 PERFORM auditoria.registrar(clock_timestamp(),'documento.plantilla_activar','documento',p_id,NULL,'ok');
 RETURN QUERY SELECT true,previa+1;
END $$;
REVOKE EXECUTE ON FUNCTION documento.activar_version_plantilla(uuid,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION documento.activar_version_plantilla(uuid,bigint) TO etf_app;
