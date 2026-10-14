package catalogo

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"github.com/jackc/pgx/v5"
	"time"
)

type Repositorio struct{}

var _ salida.Repositorio = (*Repositorio)(nil)

const columnasVersion = `id::text, ambito, numero, vigente_desde, vigente_hasta, estado, motivo`

func leerVersion(fila pgx.Row) (dominio.Version, error) {
	var v dominio.Version
	err := fila.Scan(&v.ID, &v.Ambito, &v.Numero, &v.VigenteDesde, &v.VigenteHasta, &v.Estado, &v.Motivo)
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.Version{}, dominio.ErrNoDisponible
	}
	if err != nil {
		return dominio.Version{}, dominio.ErrPersistencia
	}
	return v, nil
}
func (*Repositorio) Vigente(ctx context.Context, ambito string, fecha time.Time) (dominio.Version, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Version{}, err
	}
	return leerVersion(tx.QueryRow(ctx, `SELECT `+columnasVersion+` FROM catalogo.version
 WHERE ambito=$1 AND estado='publicada' AND daterange(vigente_desde,vigente_hasta,'[)') @> $2::date`, ambito, fecha))
}
func (*Repositorio) Versiones(ctx context.Context, ambito string) ([]dominio.Version, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT `+columnasVersion+`
 FROM catalogo.version WHERE ambito=$1 ORDER BY numero DESC`, ambito)
	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	defer filas.Close()
	versiones := []dominio.Version{}
	for filas.Next() {
		v, err := leerVersion(filas)
		if err != nil {
			return nil, err
		}
		versiones = append(versiones, v)
	}
	if filas.Err() != nil {
		return nil, dominio.ErrPersistencia
	}
	return versiones, nil
}
func (r *Repositorio) Datos(ctx context.Context, v dominio.Version) (dominio.Datos, error) {
	datos := dominio.Datos{Version: v, Equipos: []dominio.Equipo{}, Centros: []dominio.Centro{}, Municipios: []dominio.Municipio{}, Opciones: []dominio.Opcion{}, Plazos: []dominio.Plazo{}, Tramos: []dominio.Tramo{}}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Datos{}, err
	}
	switch v.Ambito {
	case "numeracion":
		regla, err := r.regla(ctx, v)
		if err != nil {
			return dominio.Datos{}, err
		}
		datos.Numeracion = &regla
	case "listas":
		filas, err := tx.Query(ctx, `SELECT lista,codigo,orden,admite_texto,activa
 FROM catalogo.lista_opcion WHERE version_id=$1 ORDER BY lista,orden`, v.ID)
		if err != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
		defer filas.Close()
		for filas.Next() {
			var o dominio.Opcion
			if err := filas.Scan(&o.Lista, &o.Codigo, &o.Orden, &o.AdmiteTexto, &o.Activa); err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
			datos.Opciones = append(datos.Opciones, o)
		}
		if filas.Err() != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
	case "plazos":
		datos.DuracionesFase, err = r.duracionesFase(ctx, v.ID)
		if err != nil {
			return dominio.Datos{}, err
		}
		filas, err := tx.Query(ctx, `SELECT clave,valor_entero,unidad
 FROM catalogo.parametro WHERE version_id=$1 ORDER BY clave`, v.ID)
		if err != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
		defer filas.Close()
		for filas.Next() {
			var p dominio.Plazo
			if err := filas.Scan(&p.Clave, &p.Valor, &p.Unidad); err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
			datos.Plazos = append(datos.Plazos, p)
		}
		if filas.Err() != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
	case "tramos":
		filas, err := tx.Query(ctx, `SELECT tipo,orden,desde::float8,hasta::float8,unidad
 FROM catalogo.tramo WHERE version_id=$1 ORDER BY tipo,orden`, v.ID)
		if err != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
		defer filas.Close()
		for filas.Next() {
			var t dominio.Tramo
			if err := filas.Scan(&t.Tipo, &t.Orden, &t.Desde, &t.Hasta, &t.Unidad); err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
			datos.Tramos = append(datos.Tramos, t)
		}
		if filas.Err() != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
	case "asignacion_municipios":
		// Consultas constantes por colección, sin una consulta por municipio.
		for _, tabla := range []string{"equipo", "centro_ss"} {
			// tabla procede de esta lista cerrada; los valores externos son parámetros.
			filas, err := tx.Query(ctx, `SELECT id::text,codigo,clave_texto,activo_desde,activo_hasta
 FROM catalogo.`+tabla+`
 ORDER BY codigo`)
			if err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
			for filas.Next() {
				var e dominio.Equipo
				if err := filas.Scan(&e.ID, &e.Codigo, &e.ClaveTexto, &e.ActivoDesde, &e.ActivoHasta); err != nil {
					filas.Close()
					return dominio.Datos{}, dominio.ErrPersistencia
				}
				if tabla == "equipo" {
					datos.Equipos = append(datos.Equipos, e)
				} else {
					datos.Centros = append(datos.Centros, e)
				}
			}
			err = filas.Err()
			filas.Close()
			if err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
		}
		filas, err := tx.Query(ctx, `SELECT m.id::text,m.ine,m.clave_texto,m.tipo,m.matriz_id::text,m.ambito_diputacion,
    a.centro_ss_id::text,a.equipo_id::text,a.zts FROM catalogo.municipio_asignacion a
    JOIN catalogo.municipio m ON m.id=a.municipio_id WHERE a.version_id=$1 ORDER BY m.ine`, v.ID)
		if err != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
		defer filas.Close()
		for filas.Next() {
			var m dominio.Municipio
			if err := filas.Scan(&m.ID, &m.INE, &m.ClaveTexto, &m.Tipo, &m.MatrizID, &m.AmbitoDiputacion, &m.CentroID, &m.EquipoID, &m.ZTS); err != nil {
				return dominio.Datos{}, dominio.ErrPersistencia
			}
			datos.Municipios = append(datos.Municipios, m)
		}
		if filas.Err() != nil {
			return dominio.Datos{}, dominio.ErrPersistencia
		}
	}
	return datos, nil
}
