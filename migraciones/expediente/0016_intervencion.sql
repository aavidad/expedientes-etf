SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.marca_menor_excluido') IS NULL
 OR to_regclass('expediente.intervencion_revision') IS NOT NULL
 OR to_regclass('catalogo.intervencion_opcion') IS NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
CREATE TABLE expediente.intervencion_revision (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0),vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 fase text, listas_conocidas text[] NOT NULL CHECK(listas_conocidas <@ ARRAY['coordinacion','factor_familia','factor_menor','tipologia']),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id),registrado_en timestamptz NOT NULL,
 UNIQUE(id,expediente_id,equipo_id),UNIQUE(expediente_id,version),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.intervencion_marca (
 revision_id uuid NOT NULL,marca_id uuid NOT NULL,expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 texto_otros text NOT NULL DEFAULT '' CHECK(length(texto_otros)<=4000),
 PRIMARY KEY(revision_id,marca_id),UNIQUE(revision_id,marca_id,expediente_id,equipo_id),
 FOREIGN KEY(revision_id,expediente_id,equipo_id) REFERENCES expediente.intervencion_revision(id,expediente_id,equipo_id),
 FOREIGN KEY(marca_id,expediente_id,equipo_id) REFERENCES expediente.marca(id,expediente_id,equipo_id)
);
CREATE TABLE expediente.intervencion_exclusion (
 revision_id uuid NOT NULL,marca_id uuid NOT NULL,menor_id uuid NOT NULL,expediente_id uuid NOT NULL,equipo_id uuid NOT NULL,
 PRIMARY KEY(revision_id,marca_id,menor_id),
 FOREIGN KEY(revision_id,marca_id,expediente_id,equipo_id) REFERENCES expediente.intervencion_marca(revision_id,marca_id,expediente_id,equipo_id),
 FOREIGN KEY(menor_id,expediente_id,equipo_id) REFERENCES expediente.menor(id,expediente_id,equipo_id)
);
CREATE INDEX intervencion_historia ON expediente.intervencion_revision(expediente_id,vigente_desde DESC,version DESC);
CREATE INDEX intervencion_marca_pertenencia ON expediente.intervencion_marca(expediente_id,equipo_id);
CREATE INDEX intervencion_exclusion_menor ON expediente.intervencion_exclusion(menor_id,expediente_id,equipo_id);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['intervencion_revision','intervencion_marca','intervencion_exclusion'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT,INSERT ON expediente.%I TO etf_app',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia',tabla);
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING(
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (
   ''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) OR
   (''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[])))))',tabla);
  EXECUTE format('CREATE POLICY escritura ON expediente.%I FOR INSERT TO etf_app WITH CHECK(
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'',''))
   AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[]))
   AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=%1$I.expediente_id AND e.equipo_id=%1$I.equipo_id AND e.estado IN (''aceptada'',''en_intervencion'')))',tabla);
  EXECUTE format('CREATE TRIGGER exigir_expediente_abierto BEFORE INSERT OR UPDATE OR DELETE ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.exigir_expediente_abierto()',tabla);
 END LOOP;
END $$;
CREATE VIEW expediente.consulta_intervencion_vigencia WITH(security_invoker=true) AS
 WITH efectivas AS (
  SELECT DISTINCT ON(expediente_id,vigente_desde) * FROM expediente.intervencion_revision ORDER BY expediente_id,vigente_desde,version DESC
 ), cambios AS (
  SELECT r.*,CASE WHEN fase IS DISTINCT FROM lag(fase) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) THEN 1 ELSE 0 END AS cambio FROM efectivas r
 ), tramos AS (
  SELECT r.*,sum(cambio) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) AS tramo FROM cambios r
 )
 SELECT id,expediente_id,equipo_id,version,fase,listas_conocidas,vigente_desde,
 lead(vigente_desde) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) AS vigente_hasta,
 CASE WHEN fase IS NOT NULL THEN min(vigente_desde) OVER(PARTITION BY expediente_id,tramo) END AS fase_desde
 FROM tramos;
-- La proyección compartida no expone explicaciones ni otros textos libres.
CREATE VIEW expediente.consulta_intervencion WITH(security_invoker=true) AS
 SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.vigente_hasta,r.fase_desde,
 jsonb_build_object('fase',r.fase,'listas',(
  SELECT jsonb_object_agg(lista,CASE WHEN lista=ANY(r.listas_conocidas) THEN coalesce((
   SELECT jsonb_agg(jsonb_build_object('codigo',m.codigo,'menores_excluidos',coalesce((
    SELECT jsonb_agg(x.menor_id ORDER BY x.menor_id) FROM expediente.intervencion_exclusion x WHERE x.revision_id=r.id AND x.marca_id=m.id),'[]'::jsonb)) ORDER BY m.codigo)
   FROM expediente.intervencion_marca rm JOIN expediente.marca m ON m.id=rm.marca_id WHERE rm.revision_id=r.id AND m.lista=listas.lista),'[]'::jsonb) ELSE 'null'::jsonb END)
  FROM unnest(ARRAY['coordinacion','factor_familia','factor_menor','tipologia']) listas(lista))) AS datos
 FROM expediente.consulta_intervencion_vigencia r;
REVOKE ALL ON expediente.consulta_intervencion_vigencia,expediente.consulta_intervencion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.consulta_intervencion_vigencia,expediente.consulta_intervencion TO etf_app,etf_copia;
