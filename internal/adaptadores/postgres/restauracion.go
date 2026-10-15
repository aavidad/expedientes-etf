package postgres

import (
	"context"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Restauracion struct{}

//etf:tecnico
func (*Restauracion) Leer(ctx context.Context) (r dominio.RevisionRestauracion, err error) {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	if _, err = e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0))`); err != nil {
		return r, err
	}
	var permitido bool
	if err = e.QueryRow(ctx, `SELECT identidad.es_administrador()`).Scan(&permitido); err != nil {
		return r, err
	}
	if !permitido {
		return r, dominio.ErrAccesoDenegado
	}
	err = e.QueryRow(ctx, `SELECT id::text,copia_en,version FROM identidad.revision_restauracion WHERE confirmada_en IS NULL AND anulada_en IS NULL`).Scan(&r.ID, &r.CopiaEn, &r.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, dominio.ErrSinRevision
	}
	if err != nil {
		return r, errorSQLRestauracion(err)
	}
	r.Cambios = []dominio.CambioRestauracion{}
	r.Equipos = map[string]string{}
	rows, err := e.Query(ctx, `SELECT id::text,anterior,propuesto,decision FROM identidad.cambio_restauracion WHERE revision_id=$1 ORDER BY coalesce(propuesto->>'nombre',anterior->>'nombre'),id`, r.ID)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var c dominio.CambioRestauracion
		if err = rows.Scan(&c.ID, &c.Anterior, &c.Propuesto, &c.Decision); err != nil {
			rows.Close()
			return r, err
		}
		r.Cambios = append(r.Cambios, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return r, err
	}
	// La composición HTTP obtiene las etiquetas a través del puerto de Catálogos.
	return r, nil
}

//etf:tecnico
func (*Restauracion) Resolver(ctx context.Context, id, cambio string, version int64, decision string) error {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `SELECT identidad.resolver_cambio_restauracion($1,$2,$3,$4)`, id, cambio, version, decision)
	return errorSQLRestauracion(err)
}

//etf:tecnico
func (*Restauracion) Confirmar(ctx context.Context, id string, version int64) error {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `SELECT identidad.confirmar_revision_restauracion($1,$2)`, id, version)
	return errorSQLRestauracion(err)
}
func errorSQLRestauracion(err error) error {
	if err == nil {
		return nil
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		for _, clave := range []dominio.Error{dominio.ErrCambiosPendientes, dominio.ErrRevisionPropia, dominio.ErrRevisionRestauracion, dominio.ErrEquipoRestauracion} {
			if p.Message == string(clave) {
				return clave
			}
		}
	}
	return errorSQLPerfiles(err)
}
