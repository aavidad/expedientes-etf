package expediente

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

// Las iniciales se calculan en PostgreSQL: el listado nunca carga nombres en Go.
const proyeccionEquipo = `WITH datos AS (
 SELECT e.*,coalesce((SELECT c->>'valor' FROM jsonb_array_elements(dd.contenido->'contenido'->'campos') c
 WHERE c->>'codigo'='familia.contacto.apellidos' LIMIT 1),'') AS apellidos
 FROM expediente.expediente e JOIN expediente.derivacion_datos dd ON dd.expediente_id=e.id
 WHERE e.numero IS NOT NULL AND
 (coalesce(nullif(current_setting('app.equipos',true),''),'{}')='{}' OR e.equipo_id=ANY(current_setting('app.equipos',true)::uuid[]))
), resumen AS (
 SELECT e.id::text,e.equipo_id::text,e.numero,
 coalesce((SELECT string_agg(upper(left(p,1))||'.',' ' ORDER BY orden)
 FROM regexp_split_to_table(trim(e.apellidos),'\s+') WITH ORDINALITY AS palabras(p,orden) WHERE p<>''),'') AS iniciales,
 e.apellidos,e.estado,e.subprograma,e.fase_intervencion,e.fecha_recepcion,e.fecha_aceptacion,
 (SELECT count(*) FROM expediente.menor m WHERE m.expediente_id=e.id)::integer AS menores,e.version,
 EXISTS(SELECT 1 FROM expediente.propuesta_cierre p WHERE p.expediente_id=e.id AND p.estado='pendiente') AS cierre_pendiente,
 EXISTS(SELECT 1 FROM expediente.salida_spm s WHERE s.expediente_id=e.id AND s.fecha_derivacion IS NOT NULL AND s.respuesta='a_la_espera') AS spm_pendiente
 FROM datos e
)
 SELECT id,equipo_id,numero,iniciales,estado,subprograma,fase_intervencion,fecha_recepcion,fecha_aceptacion,menores,version,cierre_pendiente,spm_pendiente FROM resumen `

func leerResumen(row pgx.Row) (d.ResumenEquipo, error) {
	var r d.ResumenEquipo
	var fecha time.Time
	var pendiente, spm bool
	err := row.Scan(&r.ID, &r.EquipoID, &r.Numero, &r.Familia, &r.Estado, &r.Subprograma, &r.Fase, &r.FechaRecepcion, &fecha, &r.Menores, &r.Version, &pendiente, &spm)
	r.FechaAceptacion = fecha.Format("2006-01-02")
	r.Avisos = []string{}
	if pendiente {
		r.Avisos = append(r.Avisos, "equipo.aviso.cierre_pendiente")
	}
	if spm {
		r.Avisos = append(r.Avisos, "equipo.aviso.spm_pendiente")
	}
	return r, err
}

func (*Repositorio) ListarEquipo(ctx context.Context, f d.FiltroEquipo) (d.PaginaEquipo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.PaginaEquipo{}, err
	}
	filas, err := tx.Query(ctx, proyeccionEquipo+` WHERE ($1='' OR strpos(lower(numero),lower($1))>0 OR strpos(lower(apellidos),lower($1))>0 OR strpos(lower(iniciales),lower($1))>0)
 AND ($2='' OR estado=$2) AND ($3='' OR subprograma=$3) AND ($4='' OR fase_intervencion=$4)
 ORDER BY (cierre_pendiente OR spm_pendiente) DESC,numero,id LIMIT $5 OFFSET $6`, f.Busqueda, f.Estado, f.Subprograma, f.Fase, f.Limite+1, f.Desplazamiento)
	if err != nil {
		return d.PaginaEquipo{}, err
	}
	defer filas.Close()
	p := d.PaginaEquipo{Filas: []d.ResumenEquipo{}}
	for filas.Next() {
		r, err := leerResumen(filas)
		if err != nil {
			return d.PaginaEquipo{}, err
		}
		p.Filas = append(p.Filas, r)
	}
	if len(p.Filas) > f.Limite {
		p.HayMas = true
		p.Filas = p.Filas[:f.Limite]
	}
	return p, filas.Err()
}

func (*Repositorio) FichaEquipo(ctx context.Context, id string) (d.FichaEquipo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.FichaEquipo{}, err
	}
	r, err := leerResumen(tx.QueryRow(ctx, proyeccionEquipo+` WHERE id=$1`, id)) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.FichaEquipo{}, d.ErrNoEncontrada
	}
	if err != nil {
		return d.FichaEquipo{}, err
	}
	f := d.FichaEquipo{ResumenEquipo: r, Hitos: []d.SucesoExpediente{}}
	err = tx.QueryRow(ctx, `SELECT coalesce((SELECT c->>'valor' FROM jsonb_array_elements(contenido->'contenido'->'campos') c WHERE c->>'codigo'='familia.contacto.apellidos' LIMIT 1),'') FROM expediente.derivacion_datos WHERE expediente_id=$1`, id).Scan(&f.Familia) //etf:tecnico
	if err != nil {
		return d.FichaEquipo{}, err
	}
	filas, err := tx.Query(ctx, `SELECT id::text,tipo,fecha,confirmado_por::text FROM expediente.hito WHERE expediente_id=$1 UNION ALL SELECT salida_id::text AS id,tipo,fecha,confirmado_por::text FROM expediente.salida_servicio_hito WHERE expediente_id=$1 ORDER BY fecha,id`, id) //etf:tecnico
	if err != nil {
		return d.FichaEquipo{}, err
	}
	defer filas.Close()
	for filas.Next() {
		var h d.SucesoExpediente
		h.Tipo = "hito"
		if err := filas.Scan(&h.ID, &h.Codigo, &h.Fecha, &h.Autor); err != nil {
			return d.FichaEquipo{}, err
		}
		f.Hitos = append(f.Hitos, h)
	}
	if err := filas.Err(); err != nil {
		return d.FichaEquipo{}, err
	}
	filas.Close()
	f.SalidasServicio = []d.SalidaServicio{}
	salidas, err := tx.Query(ctx, `SELECT id::text,servicio_destino,motivo,estado,version,fecha,servicio_texto FROM expediente.salida_servicio WHERE expediente_id=$1 ORDER BY creada_en DESC,id LIMIT 100`, id) //etf:tecnico
	if err != nil {
		return d.FichaEquipo{}, err
	}
	defer salidas.Close()
	for salidas.Next() {
		var s d.SalidaServicio
		var dia *time.Time
		if err := salidas.Scan(&s.ID, &s.Servicio, &s.Motivo, &s.Estado, &s.Version, &dia, &s.ServicioTexto); err != nil {
			return d.FichaEquipo{}, err
		}
		if dia != nil {
			s.Fecha = dia.Format("2006-01-02")
		}
		f.SalidasServicio = append(f.SalidasServicio, s)
	}
	return f, salidas.Err()
}
