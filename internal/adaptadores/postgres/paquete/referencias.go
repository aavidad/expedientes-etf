package paquete

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strconv"
)

type Referencias struct{}

func (*Referencias) Consultar(ctx context.Context, codigo string) (dominio.ReferenciaCatalogo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.ReferenciaCatalogo{}, err
	}
	r := dominio.ReferenciaCatalogo{Codigo: codigo}
	err = tx.QueryRow(ctx, `
SELECT a.version,a.version_contenido,i.ruta_relativa,i.huella_sha256 FROM paquete.referencia_activa a JOIN paquete.idioma_instalado i ON i.id=a.idioma_id WHERE a.tipo='catalogo' AND a.codigo=$1
`, codigo).Scan(&r.Version, &r.Contenido, &r.Ruta, &r.Huella)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	return r, nil
}
func (*Referencias) Activar(ctx context.Context, r dominio.ReferenciaCatalogo, esperada int) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var numero int
	err = tx.QueryRow(ctx, `
SELECT paquete.activar_catalogo($1,$2,$3,$4,$5)
`, r.Codigo, r.Contenido, r.Ruta, r.Huella, esperada).Scan(&numero)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "40001" {
			actual, _ := strconv.Atoi(pg.Detail)
			return &dominio.Conflicto{Actual: actual, Codigo: r.Codigo}
		}
		return dominio.ErrPublicacion
	}
	return nil
}

func (*Referencias) Contenido(ctx context.Context, codigo, contenido string) (dominio.ReferenciaCatalogo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.ReferenciaCatalogo{}, err
	}
	r := dominio.ReferenciaCatalogo{Codigo: codigo, Contenido: contenido}
	err = tx.QueryRow(ctx, `SELECT ruta_relativa,huella_sha256
 FROM paquete.idioma_instalado WHERE codigo=$1 AND version=$2`, codigo, contenido).Scan(&r.Ruta, &r.Huella)
	if err != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	return r, nil
}

func (*Referencias) Bloquear(ctx context.Context) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(697014,hashtext('es'));
 SELECT pg_advisory_xact_lock(697014,hashtext('en'));`)
	// El orden coincide con Preparar/Activar: es, en.
	if err != nil {
		return dominio.ErrPublicacion
	}
	return nil
}
func (*Referencias) Conservadas(ctx context.Context) (map[string]map[string]bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT codigo,version
 FROM paquete.idioma_instalado WHERE codigo IN ('es','en') AND version ~ '^[0-9a-f]{64}$'`)
	if err != nil {
		return nil, dominio.ErrPublicacion
	}
	defer rows.Close()
	m := map[string]map[string]bool{"es": {}, "en": {}}
	for rows.Next() {
		var cod, version string
		if err := rows.Scan(&cod, &version); err != nil {
			return nil, dominio.ErrPublicacion
		}
		m[cod][version] = true
	}
	if rows.Err() != nil {
		return nil, dominio.ErrPublicacion
	}
	return m, nil
}
