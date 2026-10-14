package expediente

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

func (*Repositorio) RepetirActoCierre(ctx context.Context, u, c, r, h string, t time.Time) (d.ResultadoCierre, bool, error) {
	return repeticion[d.ResultadoCierre](ctx, u, c, r, h, t)
}
func (*Repositorio) RecordarActoCierre(ctx context.Context, u, c, r, h string, p d.ResultadoCierre, t time.Time) error {
	return recordar(ctx, u, c, r, h, p, t)
}
func (*Repositorio) BloquearCierre(ctx context.Context, id string) (d.SituacionCierre, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.SituacionCierre{}, err
	}
	var s d.SituacionCierre
	err = tx.QueryRow(ctx, `SELECT estado,version,coalesce(equipo_id::text,'') FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&s.Estado, &s.Version, &s.EquipoID) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return s, d.ErrNoEncontrada
	}
	if err != nil {
		return s, err
	}
	err = tx.QueryRow(ctx, `SELECT id::text,to_char(apertura_fecha,'YYYY-MM-DD'),coalesce(to_char(cierre_fecha,'YYYY-MM-DD'),'') FROM expediente.episodio e WHERE expediente_id=$1 AND NOT EXISTS(SELECT 1 FROM expediente.episodio hijo WHERE hijo.reapertura_de_id=e.id)`, id).Scan(&s.EpisodioID, &s.AperturaFecha, &s.UltimoCierre) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return s, d.ErrEstado
	}
	if err != nil {
		return s, err
	}
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=$1 AND tipo='consentimiento')`, id).Scan(&s.Consentimiento) //etf:tecnico
	if err != nil {
		return s, err
	}
	err = tx.QueryRow(ctx, `SELECT id::text,propuesta_por::text,motivo_finalizacion,motivo_acta,motivo_spaf FROM expediente.propuesta_cierre WHERE expediente_id=$1 AND estado='pendiente'`, id).Scan(&s.PropuestaID, &s.PropuestaPor, &s.Motivo, &s.MotivoActa, &s.MotivoSPAF) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return s, nil
	}
	return s, err
}

//etf:tecnico
func (*Repositorio) ConfirmarCierre(ctx context.Context, id string, s d.SituacionCierre, p d.PeticionCierre, u string, t time.Time) (d.ResultadoCierre, error) {
	r := d.ResultadoCierre{ExpedienteID: id, EpisodioID: s.EpisodioID, Estado: d.Cerrada, Fecha: p.Fecha, Version: s.Version + 1}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var tag pgconn.CommandTag
	if s.PropuestaID != "" {
		tag, err = tx.Exec(ctx, `UPDATE expediente.propuesta_cierre SET estado='confirmada',resuelta_por=$2,resuelta_en=$3 WHERE id=$1 AND estado='pendiente'`, s.PropuestaID, u, t)
		if err != nil {
			return r, err
		}
		if tag.RowsAffected() != 1 {
			return r, d.ErrEstado
		}
	}
	tag, err = tx.Exec(ctx, `UPDATE expediente.episodio SET cierre_fecha=$2::date,motivo_cierre=$3,motivo_acta=$4,motivo_acta_texto=$5,motivo_spaf=$6,cerrado_por=$7,cerrado_en=$8,propuesta_cierre_id=nullif($9,'')::uuid,forzado_id=nullif($10,'')::uuid,cierre_historico=($9='') WHERE id=$1 AND cierre_fecha IS NULL`, s.EpisodioID, p.Fecha, s.Motivo, s.MotivoActa, p.MotivoActaTexto, s.MotivoSPAF, u, t, s.PropuestaID, p.ForzadoID)
	if err != nil {
		return r, err
	}
	if tag.RowsAffected() != 1 {
		return r, d.ErrEstado
	}
	if err = cambiarEstadoCierre(ctx, id, s, d.Cerrada, u, t); err != nil {
		return r, err
	}
	return r, nil
}

//etf:tecnico
func (*Repositorio) Reabrir(ctx context.Context, id string, s d.SituacionCierre, p d.PeticionReapertura, estado, u string, t time.Time) (d.ResultadoCierre, error) {
	r := d.ResultadoCierre{ExpedienteID: id, Estado: estado, Fecha: p.Fecha, Version: s.Version + 1}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	if err = cambiarEstadoCierre(ctx, id, s, estado, u, t); err != nil {
		return r, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,reapertura_de_id,motivo_reapertura,creado_por,creado_en) VALUES($1,$2,$3::date,$4,$5,$6,$7) RETURNING id::text`, id, s.EquipoID, p.Fecha, s.EpisodioID, p.Motivo, u, t).Scan(&r.EpisodioID)
	return r, err
}

//etf:tecnico
func cambiarEstadoCierre(ctx context.Context, id string, s d.SituacionCierre, estado, u string, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE expediente.expediente SET estado=$2,version=version+1,modificado_por=$3,modificado_en=$4 WHERE id=$1 AND version=$5 AND estado=$6`, id, estado, u, t, s.Version, s.Estado)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return d.ErrEstado
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.estado_cambio(expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en) VALUES($1,$2,$3,$4,$5,$6)`, id, s.EquipoID, s.Estado, estado, u, t)
	return err
}
func (*Repositorio) LeerActaCierre(ctx context.Context, id string) (*d.ActaCierre, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	var a d.ActaCierre
	err = tx.QueryRow(ctx, `SELECT p.id::text,p.propuesta_por::text,p.motivo_acta,coalesce(e.motivo_acta_texto,''),coalesce(to_char(e.cierre_fecha,'YYYY-MM-DD'),'')
 FROM expediente.propuesta_cierre p LEFT JOIN expediente.episodio e ON e.propuesta_cierre_id=p.id
 WHERE p.expediente_id=$1 AND (p.estado='pendiente' OR (p.estado='confirmada' AND e.id=(SELECT id FROM expediente.episodio e WHERE expediente_id=$1 AND NOT EXISTS(SELECT 1 FROM expediente.episodio hijo WHERE hijo.reapertura_de_id=e.id))))
 ORDER BY p.propuesta_en DESC,p.id DESC LIMIT 1`, id).Scan(&a.PropuestaID, &a.PropuestaPor, &a.MotivoActa, &a.MotivoActaTexto, &a.Fecha) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &a, err
}
