package postgres

import (
	"context"
	"errors"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
)

type PublicacionPaquetes struct{ base *Base }

func NuevaPublicacionPaquetes(b *Base) *PublicacionPaquetes { return &PublicacionPaquetes{b} }

func (p *PublicacionPaquetes) Activar(ctx context.Context, a dominio.Actor, v dominio.VersionInstalada, version int64) (dominio.Referencia, error) {
	r := dominio.Referencia{VersionInstalada: v}
	if !a.PuedeAdministrar() {
		return dominio.Referencia{}, dominio.ErrPermiso
	}
	if !v.Valida() {
		return dominio.Referencia{}, dominio.ErrPaqueteInvalido
	}
	err := p.base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "SET LOCAL statement_timeout = '25s'; SET LOCAL lock_timeout = '10s'"); err != nil {
			return err
		} //etf:tecnico
		return tx.QueryRow(ctx, "SELECT paquete.activar($1,$2,$3,$4,$5,$6)", v.Tipo, v.Codigo, v.Version, v.Huella, v.Ruta, version).Scan(&r.Revision) //etf:tecnico
	})
	if err != nil {
		return dominio.Referencia{}, errorPaquete(err)
	}
	return r, nil
}

func (p *PublicacionPaquetes) Referencias(ctx context.Context, tipo string) ([]dominio.Referencia, error) {
	refs := []dominio.Referencia{}
	leer := func(ctx context.Context) error {
		tx, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, "SELECT * FROM paquete.referencias($1)", tipo) //etf:tecnico
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r dominio.Referencia
			if err := rows.Scan(&r.Tipo, &r.Codigo, &r.Version, &r.Huella, &r.Ruta, &r.Revision); err != nil {
				return err
			}
			refs = append(refs, r)
		}
		return rows.Err()
	}
	var err error
	if _, e := DesdeContexto(ctx); e == nil {
		err = leer(ctx)
	} else {
		err = p.base.Ejecutar(ctx, plataforma.Identidad{}, leer)
	}
	if err != nil {
		return nil, errorPaquete(err)
	}
	return refs, nil
}

func (p *PublicacionPaquetes) Registrada(ctx context.Context, v dominio.VersionInstalada) (bool, error) {
	var registrada bool
	err := p.base.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		tx, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT paquete.ruta_registrada($1,$2)", v.Tipo, v.Ruta).Scan(&registrada) //etf:tecnico
	})
	return registrada, errorPaquete(err)
}

func errorPaquete(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Message == "paquete.acceso_denegado" {
			return dominio.ErrPermiso
		}
		if pg.Code == "40001" || pg.Code == "23505" {
			return dominio.ErrConflicto
		}
		if pg.Code == "22023" || pg.Code == "23514" {
			return dominio.ErrPaqueteInvalido
		}
	}
	return dominio.ErrNoDisponible
}
