SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$
DECLARE restriccion text;
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('documento.referencia_activa') IS NULL
       OR to_regclass('documento.archivo_plantilla') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM documento.tipo_documento WHERE codigo='entrada/no_aceptacion') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
    SELECT conname INTO STRICT restriccion FROM pg_constraint
      WHERE conrelid='documento.plantilla'::regclass AND contype='c'
        AND pg_get_constraintdef(oid) LIKE '%ruta_relativa%';
    EXECUTE format('ALTER TABLE documento.plantilla DROP CONSTRAINT %I',restriccion);
END $$;
ALTER TABLE documento.plantilla ADD CONSTRAINT plantilla_ruta_publicada CHECK (
    ruta_relativa='plantillas/'||tipo_documento||'/v'||version||'.docx'
    OR ruta_relativa='plantillas/'||tipo_documento||'/.versiones/'||id::text||'/v'||version||'.docx');

-- El archivo es historia de sólo adición: nunca modifica contenido, versión,
-- huella ni estado de las plantillas que ya pueden figurar en emisiones.
CREATE TABLE documento.archivo_plantilla (
    anterior_id uuid PRIMARY KEY REFERENCES documento.plantilla(id),
    nueva_id uuid NOT NULL UNIQUE REFERENCES documento.plantilla(id),
    archivada_por uuid NOT NULL REFERENCES identidad.usuario(id),
    archivada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (anterior_id<>nueva_id)
);
ALTER TABLE documento.archivo_plantilla ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.archivo_plantilla FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.archivo_plantilla FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON documento.archivo_plantilla TO etf_app,etf_copia;
CREATE POLICY archivo_propietario ON documento.archivo_plantilla TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY archivo_administracion ON documento.archivo_plantilla FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
    AND 'administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE TRIGGER archivo_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON documento.archivo_plantilla
    FOR EACH STATEMENT EXECUTE FUNCTION documento.inmutable();

CREATE FUNCTION documento.publicar_plantilla(p_id uuid,p_tipo text,p_idioma text,p_contenido text,p_ruta text,p_huella text,p_previa bigint)
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
    INSERT INTO documento.plantilla(id,tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,subida_por,comprobacion)
      VALUES(p_id,p_tipo,p_idioma,p_contenido,p_ruta,decode(p_huella,'hex'),rol,current_date,usuario,'motor_docx_1');
    INSERT INTO documento.archivo_plantilla(anterior_id,nueva_id,archivada_por) VALUES(anterior,p_id,usuario);
    UPDATE documento.referencia_activa SET plantilla_id=p_id,version=previa+1 WHERE tipo_documento=p_tipo AND idioma=p_idioma;
    PERFORM auditoria.registrar(clock_timestamp(),'documento.plantilla_activar','documento',p_id,NULL,'ok');
    RETURN QUERY SELECT true,previa+1;
END $$;
REVOKE EXECUTE ON FUNCTION documento.publicar_plantilla(uuid,text,text,text,text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION documento.publicar_plantilla(uuid,text,text,text,text,text,bigint) TO etf_app;
