package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strconv"
)

func errorEdicion(err error, recurso string) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code == "40001" {
			n, _ := strconv.Atoi(pg.Detail)
			return &dominio.Conflicto{Actual: n, Recurso: recurso}
		}
		if pg.Code == "42501" {
			return dominio.ErrPermiso
		}
		if pg.Code == "23505" || pg.Code == "23514" || pg.Code == "23P01" || pg.Code == "22003" || pg.Code == "22007" || pg.Message == "catalogo.edicion_invalida" {
			return dominio.ErrEdicion
		}
		if pg.Message == "catalogo.vigencia_invalida" {
			return dominio.ErrVigencia
		}
	}
	return dominio.ErrPersistencia
}
func (r *Repositorio) Ultima(ctx context.Context, ambito string) (dominio.Datos, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Datos{}, err
	}
	v, err := leerVersion(tx.QueryRow(ctx, `
SELECT
`+columnasVersion+`
 FROM catalogo.version WHERE ambito=$1 AND estado='publicada' ORDER BY numero DESC LIMIT 1
`, ambito))
	if errors.Is(err, dominio.ErrNoDisponible) {
		return dominio.Datos{Version: dominio.Version{Ambito: ambito}, Opciones: []dominio.Opcion{}, Plazos: []dominio.Plazo{}, Tramos: []dominio.Tramo{}}, nil
	}
	if err != nil {
		return dominio.Datos{}, err
	}
	return r.Datos(ctx, v)
}
func (r *Repositorio) Publicar(ctx context.Context, ambito string, c dominio.Cambio) (dominio.Version, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Version{}, err
	}
	var regla any
	switch ambito {
	case "asignacion_municipios":
		regla = c.Ubicacion
	case "listas":
		regla = c.Opcion
	case "plazos":
		regla = c.Plazo
		if c.DuracionFase != nil {
			regla = c.DuracionFase
		}
	case "tramos":
		regla = c.Tramo
	default:
		return dominio.Version{}, dominio.ErrAmbito
	}
	b, err := json.Marshal(regla)
	if err != nil {
		return dominio.Version{}, dominio.ErrEdicion
	}
	var id string
	if ambito == "asignacion_municipios" {
		err = tx.QueryRow(ctx, `
SELECT catalogo.editar_ubicacion($1,$2,$3)::text
`, c.VersionEsperada, c.VigenteDesde, b).Scan(&id)
	} else {
		err = tx.QueryRow(ctx, `
SELECT catalogo.editar($1,$2,$3,$4)::text
`, ambito, c.VersionEsperada, c.VigenteDesde, b).Scan(&id)
	}
	if err != nil {
		return dominio.Version{}, errorEdicion(err, ambito)
	}
	return leerVersion(tx.QueryRow(ctx, `
SELECT
`+columnasVersion+`
 FROM catalogo.version WHERE id=$1
`, id))
}

const columnasFirmante = `
id::text,tipo,rol,nombre,cargo_codigo,organismo_codigo,direccion,vigente_desde,vigente_hasta,version
`

func leerFirmante(row pgx.Row) (dominio.Firmante, error) {
	var f dominio.Firmante
	err := row.Scan(&f.ID, &f.Tipo, &f.Rol, &f.Nombre, &f.CargoCodigo, &f.OrganismoCodigo, &f.Direccion, &f.VigenteDesde, &f.VigenteHasta, &f.Version)
	if err != nil {
		return dominio.Firmante{}, errorEdicion(err, "firmantes")
	}
	return f, nil
}
func (*Repositorio) Firmantes(ctx context.Context) ([]dominio.Firmante, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT f.id::text,f.tipo,f.rol,f.nombre,f.cargo_codigo,f.organismo_codigo,f.direccion,f.vigente_desde,f.vigente_hasta,f.version,
 coalesce(jsonb_object_agg(t.codigo,t.version_contenido) FILTER (WHERE t.codigo IS NOT NULL),'{}'::jsonb)
 FROM catalogo.firmante f LEFT JOIN catalogo.firmante_textos t ON t.id=f.id AND t.version=f.version
 GROUP BY f.id,f.version ORDER BY f.tipo,f.rol,f.version DESC`)

	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	defer rows.Close()
	fs := []dominio.Firmante{}
	for rows.Next() {
		var f dominio.Firmante
		err := rows.Scan(&f.ID, &f.Tipo, &f.Rol, &f.Nombre, &f.CargoCodigo, &f.OrganismoCodigo, &f.Direccion, &f.VigenteDesde, &f.VigenteHasta, &f.Version, &f.ReferenciasTextos)
		if err != nil {
			return nil, dominio.ErrPersistencia
		}
		fs = append(fs, f)
	}
	if rows.Err() != nil {
		return nil, dominio.ErrPersistencia
	}
	return fs, nil
}
func (*Repositorio) GuardarFirmante(ctx context.Context, c dominio.Cambio) (dominio.Firmante, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Firmante{}, err
	}
	b, err := json.Marshal(c.Firmante)
	if err != nil {
		return dominio.Firmante{}, dominio.ErrEdicion
	}
	var id string
	err = tx.QueryRow(ctx, `
SELECT catalogo.guardar_firmante($1,$2,$3)::text
`, c.VersionEsperada, c.VigenteDesde, b).Scan(&id)
	if err != nil {
		return dominio.Firmante{}, errorEdicion(err, "firmantes")
	}
	return leerFirmante(tx.QueryRow(ctx, `
SELECT
`+columnasFirmante+`
 FROM catalogo.firmante WHERE id=$1 AND version=$2
`, id, c.VersionEsperada+1))
}
