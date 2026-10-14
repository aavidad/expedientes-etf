package expediente

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"strings"
)

// La proyección elimina nombres y texto libre antes de salir de PostgreSQL.
// La propuesta de municipio se usa sólo si es única; nunca asigna un equipo.
func (*Repositorio) ConsultarBandeja(ctx context.Context, p dominio.FiltroBandeja, codigoMunicipio string) (dominio.ResultadoBandeja, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.ResultadoBandeja{}, err
	}
	patron := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(p.Busqueda)
	var contenido []byte
	err = e.QueryRow(ctx, `WITH base AS MATERIALIZED (
 SELECT e.id::text,e.numero,e.estado,e.fecha_recepcion,e.fecha_aceptacion,e.origen='carga_inicial' AS cargado_para_revision,
 coalesce(e.equipo_id,e.equipo_carga_id)::text AS equipo_id,e.municipio_id::text,coalesce(e.subprograma,CASE WHEN d.contenido->'contenido'->>'tipo'='anexo3' THEN 'RIESGO_SOCIAL' ELSE '' END) AS subprograma,
 CASE WHEN jsonb_array_length(coalesce(nullif(d.contenido->'propuesta_municipio'->'Candidatos','null'::jsonb),'[]'::jsonb))=1
 THEN d.contenido->'propuesta_municipio'->'Candidatos'->0->>'Codigo' ELSE '' END AS municipio_propuesto,
 coalesce((SELECT c->>'valor' FROM jsonb_array_elements(d.contenido->'contenido'->'campos') c
 WHERE c->>'codigo'='familia.contacto.apellidos' LIMIT 1),'') AS apellidos,
 coalesce((SELECT string_agg(c->>'valor',' ') FROM jsonb_array_elements(d.contenido->'contenido'->'campos') c
 WHERE c->>'codigo' IN ('familia.contacto.nombre','familia.contacto.apellidos')),'') AS buscar
 FROM expediente.expediente e JOIN expediente.derivacion_datos d ON d.expediente_id=e.id
 WHERE ($1='' OR coalesce(e.equipo_id,e.equipo_carga_id)::text=$1)
 AND ($2='' OR e.fecha_recepcion >= (nullif($2,'')::date::timestamp AT TIME ZONE 'Europe/Madrid'))
 AND ($3='' OR e.fecha_recepcion < ((nullif($3,'')::date+1)::timestamp AT TIME ZONE 'Europe/Madrid'))
 ), filtradas AS MATERIALIZED (
 SELECT * FROM base WHERE ($4='' OR municipio_id=$4 OR (municipio_id IS NULL AND municipio_propuesto=$5))
 AND ($6='' OR buscar ILIKE '%'||$6||'%' OR coalesce(numero,'') ILIKE '%'||$6||'%')
 ), pagina AS (
 SELECT jsonb_build_object('id',id,'numero',coalesce(numero,''),'estado',estado,'fecha_recepcion',fecha_recepcion,
 'cargado_para_revision',cargado_para_revision,'fecha_aceptacion',coalesce(fecha_aceptacion::text,''),'equipo_id',coalesce(equipo_id,''),
 'municipio_id',coalesce(municipio_id,''),'municipio_propuesto',municipio_propuesto,'subprograma',coalesce(subprograma,''),
 'iniciales',coalesce((SELECT string_agg(upper(left(palabra,1))||'.',' ' ORDER BY orden)
 FROM regexp_split_to_table(trim(apellidos),'\s+') WITH ORDINALITY AS partes(palabra,orden) WHERE palabra<>''),'')) AS fila
 FROM filtradas WHERE ($7='' OR estado=$7)
 ORDER BY CASE WHEN $7='lista_espera' THEN fecha_recepcion END ASC,
 CASE WHEN $7='aceptada' THEN fecha_aceptacion END DESC,fecha_recepcion DESC,id
 LIMIT $8 OFFSET $9
 ) SELECT jsonb_build_object('filas',coalesce((SELECT jsonb_agg(fila) FROM pagina),'[]'::jsonb),
 'total',(SELECT count(*) FROM filtradas WHERE $7='' OR estado=$7),
 'resumen',(SELECT coalesce(jsonb_object_agg(estado,n),'{}'::jsonb) FROM (SELECT estado,count(*) n FROM filtradas GROUP BY estado) r))`,
		p.EquipoID, p.Desde, p.Hasta, p.MunicipioID, codigoMunicipio, patron, p.Estado, p.Limite, p.Desplazamiento).Scan(&contenido)
	if err != nil {
		return dominio.ResultadoBandeja{}, err
	}
	var resultado dominio.ResultadoBandeja
	err = json.Unmarshal(contenido, &resultado)
	return resultado, err
}
