package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5/pgconn"
	"strconv"
	"time"
)

func errorConservacion(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code == "42501" {
			return d.ErrAcceso
		}
		if pg.Message == "conservacion.conflicto" {
			v, _ := strconv.Atoi(pg.Detail)
			return d.Conflicto{VersionActual: v}
		}
		if pg.Message == "conservacion.no_permitida" {
			return d.ErrConservacion
		}
	}
	return d.ErrNoDisponible
}

//etf:tecnico
func (*Repositorio) MarcarConservacion(ctx context.Context, id string, anios int, regla string, ahora time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT expediente.marcar_conservacion(nullif($1,'')::uuid,$2,$3,$4)`, id, anios, regla, ahora)
	return errorConservacion(err)
}

//etf:tecnico
func (*Repositorio) ListarConservacion(ctx context.Context, ahora time.Time) ([]d.Conservacion, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT expediente.listar_conservacion($1)`, ahora).Scan(&b); err != nil {
		return nil, errorConservacion(err)
	}
	r := []d.Conservacion{}
	if json.Unmarshal(b, &r) != nil {
		return nil, d.ErrNoDisponible
	}
	return r, nil
}

//etf:tecnico
func (*Repositorio) ConfirmarConservacion(ctx context.Context, id string, version int, anonimizar bool, ahora time.Time) (d.Conservacion, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Conservacion{}, err
	}
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT expediente.confirmar_conservacion($1,$2,$3,$4)`, id, version, anonimizar, ahora).Scan(&b); err != nil {
		return d.Conservacion{}, errorConservacion(err)
	}
	var r d.Conservacion
	if json.Unmarshal(b, &r) != nil {
		return r, d.ErrNoDisponible
	}
	return r, nil
}
