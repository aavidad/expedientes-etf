SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('catalogo.parametro') IS NULL OR to_regclass('catalogo.fase_duracion') IS NULL
 OR EXISTS(SELECT 1 FROM catalogo.parametro WHERE clave='umbral_privacidad_consulta') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- Conserva todas las versiones y referencias de idioma. Si la última versión
-- empieza hoy o en el futuro, la nueva empieza el día siguiente a esa versión.
-- Hasta entonces la aplicación aplica el umbral inicial seguro de tres casos.
DO $$ DECLARE anterior catalogo.version; nueva uuid; desde date; BEGIN
 SELECT * INTO anterior FROM catalogo.version WHERE ambito='plazos' AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 desde:=greatest((CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date,anterior.vigente_desde+1);
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES('plazos',coalesce(anterior.numero,0)+1,desde,'catalogo.privacidad_consulta','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
 INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
 INSERT INTO catalogo.fase_duracion SELECT nueva,subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=anterior.id;
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM catalogo.version_textos WHERE version_id=anterior.id;
 INSERT INTO catalogo.parametro VALUES(nueva,'umbral_privacidad_consulta',3,'casos');
 PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
END $$;
ALTER TABLE catalogo.parametro ADD CONSTRAINT umbral_privacidad_seguro
 CHECK(clave<>'umbral_privacidad_consulta' OR (valor_entero>=3 AND unidad='casos'));

-- Consulta sólo puede leer la regla de privacidad publicada. El resto de
-- parámetros sigue protegido por las políticas y perfiles ya existentes.
CREATE POLICY privacidad_consulta ON catalogo.parametro FOR SELECT TO etf_app USING (
 clave='umbral_privacidad_consulta'
 AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'consulta'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND EXISTS(SELECT 1 FROM catalogo.version v WHERE v.id=version_id AND v.estado='publicada')
);
