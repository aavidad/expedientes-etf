package expediente

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

const columnasSeguimiento = `id::text,expediente_id::text,equipo_id::text,episodio_id::text,version,estado,to_char(fecha_cese,'YYYY-MM-DD'),to_char(fecha_fin,'YYYY-MM-DD'),coalesce(to_char(fecha_terminacion,'YYYY-MM-DD'),''),coalesce(motivo,''),regla_id::text,coalesce(motivo_reapertura,''),to_char(tramo_desde,'YYYY-MM-DD')`

//etf:tecnico
func leerSeguimiento(row pgx.Row) (*d.Seguimiento, error) {
	var s d.Seguimiento
	err := row.Scan(&s.ID, &s.ExpedienteID, &s.EquipoID, &s.EpisodioID, &s.Version, &s.Estado, &s.FechaCese, &s.FechaFin, &s.FechaTerminacion, &s.Motivo, &s.ReglaID, &s.MotivoReapertura, &s.TramoDesde)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

//etf:tecnico
func (*Repositorio) ObtenerSeguimiento(ctx context.Context, id string, bloquear bool) (*d.Seguimiento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	sufijo := ""
	if bloquear {
		sufijo = " FOR UPDATE"
	}
	return leerSeguimiento(tx.QueryRow(ctx, `SELECT `+columnasSeguimiento+` FROM expediente.seguimiento WHERE expediente_id=$1`+sufijo, id))
}

//etf:tecnico
func (*Repositorio) RepetirSeguimiento(ctx context.Context, u, c, r, h string, t time.Time) (d.Seguimiento, bool, error) {
	return repeticion[d.Seguimiento](ctx, u, c, r, h, t)
}

//etf:tecnico
func (*Repositorio) RecordarSeguimiento(ctx context.Context, u, c, r, h string, p d.Seguimiento, t time.Time) error {
	return recordar(ctx, u, c, r, h, p, t)
}

//etf:tecnico
func (*Repositorio) IniciarSeguimiento(ctx context.Context, id, equipo, episodio string, p d.PeticionSeguimiento, regla d.ReglaSeguimiento, u string, t time.Time) (d.Seguimiento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Seguimiento{}, err
	}
	fin, err := d.FinSeguimiento(p.Fecha, regla.Meses)
	if err != nil {
		return d.Seguimiento{}, err
	}
	r, err := leerSeguimiento(tx.QueryRow(ctx, `INSERT INTO expediente.seguimiento(expediente_id,equipo_id,episodio_id,version,estado,fecha_cese,tramo_desde,fecha_fin,regla_id,creado_por,creado_en,modificado_por,modificado_en)
 VALUES($1,$2,$3,1,'activo',$4::date,$4::date,$5::date,$6,$7,$8,$7,$8) RETURNING `+columnasSeguimiento, id, equipo, episodio, p.Fecha, fin, regla.ID, u, t))
	if err != nil {
		return d.Seguimiento{}, err
	}
	if r == nil {
		return d.Seguimiento{}, d.ErrEstado
	}
	return *r, nil
}

//etf:tecnico
func (*Repositorio) CambiarSeguimiento(ctx context.Context, s d.Seguimiento, accion string, p d.PeticionSeguimiento, regla d.ReglaSeguimiento, u string, t time.Time) (d.Seguimiento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Seguimiento{}, err
	}
	estado, fecha, motivo := "terminado", p.Fecha, p.Motivo
	fin, reglaID := s.FechaFin, s.ReglaID
	if accion == "reabrir" {
		estado, fecha, motivo = "activo", "", ""
		fin, err = d.FinSeguimiento(d.FechaCivil(t), regla.Meses)
		if err != nil {
			return d.Seguimiento{}, err
		}
		reglaID = regla.ID
	}
	r, err := leerSeguimiento(tx.QueryRow(ctx, `UPDATE expediente.seguimiento SET version=version+1,estado=$2,fecha_fin=$10::date,regla_id=$11,tramo_desde=CASE WHEN $8='reabrir' THEN ($6::timestamptz AT TIME ZONE 'Europe/Madrid')::date ELSE tramo_desde END,fecha_terminacion=nullif($3,'')::date,motivo=nullif($4,''),modificado_por=$5,modificado_en=$6,motivo_reapertura=CASE WHEN $8='reabrir' THEN $9 ELSE motivo_reapertura END
 WHERE id=$1 AND version=$7 RETURNING `+columnasSeguimiento, s.ID, estado, fecha, motivo, u, t, s.Version, accion, p.Motivo, fin, reglaID))
	if err != nil {
		return d.Seguimiento{}, err
	}
	if r == nil {
		return d.Seguimiento{}, d.ErrEstado
	}
	return *r, nil
}

//etf:tecnico
func (*Repositorio) AvisosSeguimiento(ctx context.Context, hoy string, dias int) ([]d.AvisoSeguimiento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT expediente_id::text,id::text,to_char(fecha_fin-$2::integer,'YYYY-MM-DD'),to_char(fecha_fin,'YYYY-MM-DD')
 FROM expediente.seguimiento WHERE estado='activo' AND fecha_fin-$2::integer<=$1::date AND $1::date<fecha_fin ORDER BY fecha_fin,id`, hoy, dias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := []d.AvisoSeguimiento{}
	for rows.Next() {
		var a d.AvisoSeguimiento
		a.Clave = "expediente.fin_seguimiento_proximo"
		if err = rows.Scan(&a.ExpedienteID, &a.SeguimientoID, &a.Desde, &a.Hasta); err != nil {
			return nil, err
		}
		r = append(r, a)
	}
	return r, rows.Err()
}

//etf:tecnico
func (*Repositorio) VencerSeguimientos(ctx context.Context) ([]string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT expediente_id::text FROM expediente.vencer_seguimientos()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
