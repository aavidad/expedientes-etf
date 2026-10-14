package expediente

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// Las vistas de F2.01 mantienen vigencias y RLS. La proyección no devuelve
// nombres, contactos ni narrativas. Una consulta sirve también al recuento.
func (*Repositorio) DatosAvisos(ctx context.Context, id, equipo, corte string) ([]dominio.DatosAvisos, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT jsonb_build_object(
 'id',e.id,'equipo_id',e.equipo_id,'estado',e.estado,'subprograma',e.subprograma,
 'fecha_aceptacion',to_char(e.fecha_aceptacion,'YYYY-MM-DD'),
 'consentimiento',(SELECT to_char(h.fecha,'YYYY-MM-DD') FROM expediente.hito h WHERE h.expediente_id=e.id AND h.tipo='consentimiento' AND h.fecha<=$3::date),
 'atendido_anio',EXISTS(SELECT 1 FROM expediente.episodio p JOIN expediente.hito h ON h.expediente_id=p.expediente_id AND h.tipo='consentimiento'
 WHERE p.expediente_id=e.id AND p.apertura_fecha<=$3::date AND coalesce(p.cierre_fecha,$3::date)>=date_trunc('year',$3::date)::date AND h.fecha<=$3::date),
 'familia',coalesce(f.valor,'{}'::jsonb),
 'adultos_conocidos',EXISTS(SELECT 1 FROM expediente.consulta_adultos_vigencia a WHERE a.expediente_id=e.id AND a.vigente_desde<=$3::date AND (a.vigente_hasta IS NULL OR $3::date<a.vigente_hasta)),
 'adultos',coalesce((SELECT jsonb_agg(jsonb_build_object('id',a.id,'papel',a.papel,'sexo',a.sexo,'fecha_nacimiento',a.fecha_nacimiento,
 'edad_declarada',a.edad_declarada,'nivel_educativo',a.nivel_educativo,'situacion_laboral',a.situacion_laboral,'es_tutor',a.es_tutor) ORDER BY a.orden)
 FROM expediente.consulta_adulto a WHERE a.expediente_id=e.id AND a.vigente_desde<=$3::date AND (a.vigente_hasta IS NULL OR $3::date<a.vigente_hasta)),'[]'::jsonb),
 'menores',coalesce((SELECT jsonb_agg(jsonb_build_object('id',m.id,'orden',m.orden,'sexo',m.sexo,'fecha_nacimiento',m.fecha_nacimiento,
 'subprograma',m.subprograma,'propuesto_atencion_temprana',m.propuesto_atencion_temprana) ORDER BY m.orden)
 FROM expediente.consulta_menor m WHERE m.expediente_id=e.id AND m.vigente_desde<=$3::date AND (m.vigente_hasta IS NULL OR $3::date<m.vigente_hasta)),'[]'::jsonb),
 'intervencion',i.datos,'fase_desde',to_char(i.fase_desde,'YYYY-MM-DD'))
 FROM expediente.expediente e
 LEFT JOIN expediente.consulta_familia_vigencia f ON f.expediente_id=e.id AND f.vigente_desde<=$3::date AND (f.vigente_hasta IS NULL OR $3::date<f.vigente_hasta)
 LEFT JOIN expediente.consulta_intervencion i ON i.expediente_id=e.id AND i.vigente_desde<=$3::date AND (i.vigente_hasta IS NULL OR $3::date<i.vigente_hasta)
 WHERE ($1='' OR e.id=nullif($1,'')::uuid) AND ($2='' OR e.equipo_id=nullif($2,'')::uuid)
 AND e.fecha_aceptacion IS NOT NULL AND e.fecha_aceptacion<=$3::date ORDER BY e.id`, id, equipo, corte) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := []dominio.DatosAvisos{}
	for rows.Next() {
		var b []byte
		var d dominio.DatosAvisos
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &d); err != nil {
			return nil, err
		}
		r = append(r, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if id != "" && len(r) == 0 {
		return nil, dominio.ErrNoEncontrada
	}
	return r, nil
}
