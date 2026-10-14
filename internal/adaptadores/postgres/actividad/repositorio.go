package actividad

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"time"
)

type Repositorio struct{ CalculoInterno bool }

//etf:tecnico
func tabla(tipo dominio.Tipo) (string, string, error) {
	switch tipo {
	case dominio.Comision:
		return "actividad.comision_tecnica", "observaciones,''::text,''::text", nil
	case dominio.Informe:
		return "actividad.informe_tecnico", "''::text,organismo,organismo_otros", nil
	default:
		return "", "", dominio.ErrDatos
	}
}

//etf:tecnico
func columnas(detalle string) string {
	return "id::text,equipo_id::text,to_char(fecha,'YYYY-MM-DD')," + detalle + ",version"
}

func leer(fila pgx.Row) (dominio.Registro, error) {
	var r dominio.Registro
	err := fila.Scan(&r.ID, &r.EquipoID, &r.Fecha, &r.Observaciones, &r.Organismo, &r.OrganismoOtros, &r.Version)
	if err != nil {
		return dominio.Registro{}, errorSQL(err)
	}
	return r, nil
}

func errorSQL(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.ErrNoEncontrado
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return dominio.ErrPermiso
		case "23514", "23503", "22007", "22008", "22P02":
			return dominio.ErrDatos
		}
	}
	return dominio.ErrPersistencia
}

func fechas(f dominio.Filtro) (time.Time, time.Time) {
	inicio := time.Date(f.Anio, 1, 1, 0, 0, 0, 0, time.UTC)
	return inicio, inicio.AddDate(1, 0, 0)
}

//etf:tecnico
func (*Repositorio) Consultar(ctx context.Context, tipo dominio.Tipo, f dominio.Filtro) ([]dominio.Registro, error) {
	t, detalle, err := tabla(tipo)
	if err != nil {
		return nil, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	inicio, fin := fechas(f)
	filas, err := tx.Query(ctx, "SELECT "+columnas(detalle)+" FROM "+t+" WHERE NOT retirada AND fecha >= $1 AND fecha < $2 AND ($3::uuid IS NULL OR equipo_id=$3::uuid) ORDER BY fecha,id", inicio, fin, equipo(f))
	if err != nil {
		return nil, errorSQL(err)
	}
	defer filas.Close()
	resultado := make([]dominio.Registro, 0)
	for filas.Next() {
		r, err := leer(filas)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, r)
	}
	if err := filas.Err(); err != nil {
		return nil, errorSQL(err)
	}
	return resultado, nil
}

func equipo(f dominio.Filtro) any {
	if f.EquipoID == "" {
		return nil
	}
	return f.EquipoID
}

//etf:tecnico
func (*Repositorio) Registrar(ctx context.Context, tipo dominio.Tipo, equipo string, c dominio.Cambio) (dominio.Registro, error) {
	t, detalle, err := tabla(tipo)
	if err != nil {
		return dominio.Registro{}, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Registro{}, err
	}
	if tipo == dominio.Comision {
		return leer(tx.QueryRow(ctx, "INSERT INTO "+t+"(equipo_id,fecha,observaciones) VALUES($1,$2::date,$3) RETURNING "+columnas(detalle), equipo, c.Fecha, c.Observaciones))
	}
	return leer(tx.QueryRow(ctx, "INSERT INTO "+t+"(equipo_id,fecha,organismo,organismo_otros) VALUES($1,$2::date,$3,$4) RETURNING "+columnas(detalle), equipo, c.Fecha, c.Organismo, c.OrganismoOtros))
}

// bloquear comprueba también el equipo antes de revelar una versión en conflicto.
//
//etf:tecnico
func bloquear(ctx context.Context, tx postgres.Ejecutor, t, id string, esperada int) error {
	var actual int
	err := tx.QueryRow(ctx, "SELECT version FROM "+t+" WHERE id=$1::uuid AND NOT retirada AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])) FOR UPDATE", id).Scan(&actual)
	if err != nil {
		return errorSQL(err)
	}
	if actual != esperada {
		return &dominio.Conflicto{Actual: actual}
	}
	return nil
}

//etf:tecnico
func (*Repositorio) Corregir(ctx context.Context, tipo dominio.Tipo, id string, c dominio.Cambio) (dominio.Registro, error) {
	t, detalle, err := tabla(tipo)
	if err != nil {
		return dominio.Registro{}, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Registro{}, err
	}
	if err := bloquear(ctx, tx, t, id, c.VersionEsperada); err != nil {
		return dominio.Registro{}, err
	}
	if tipo == dominio.Comision {
		return leer(tx.QueryRow(ctx, "UPDATE "+t+" SET fecha=$2::date,observaciones=$3,version=version+1 WHERE id=$1::uuid RETURNING "+columnas(detalle), id, c.Fecha, c.Observaciones))
	}
	return leer(tx.QueryRow(ctx, "UPDATE "+t+" SET fecha=$2::date,organismo=$3,organismo_otros=$4,version=version+1 WHERE id=$1::uuid RETURNING "+columnas(detalle), id, c.Fecha, c.Organismo, c.OrganismoOtros))
}

//etf:tecnico
func (*Repositorio) Retirar(ctx context.Context, tipo dominio.Tipo, id string, version int) (dominio.Registro, error) {
	t, detalle, err := tabla(tipo)
	if err != nil {
		return dominio.Registro{}, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Registro{}, err
	}
	if err := bloquear(ctx, tx, t, id, version); err != nil {
		return dominio.Registro{}, err
	}
	return leer(tx.QueryRow(ctx, "UPDATE "+t+" SET retirada=true,version=version+1 WHERE id=$1::uuid RETURNING "+columnas(detalle), id))
}

//etf:tecnico
func (*Repositorio) Organismos(ctx context.Context) ([]dominio.Organismo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT codigo,clave_texto,admite_texto,version FROM actividad.organismo ORDER BY orden`)
	if err != nil {
		return nil, errorSQL(err)
	}
	defer filas.Close()
	resultado := make([]dominio.Organismo, 0, 8)
	for filas.Next() {
		var o dominio.Organismo
		if err := filas.Scan(&o.Codigo, &o.ClaveTexto, &o.AdmiteTexto, &o.Version); err != nil {
			return nil, errorSQL(err)
		}
		resultado = append(resultado, o)
	}
	if err := filas.Err(); err != nil {
		return nil, errorSQL(err)
	}
	return resultado, nil
}

//etf:tecnico
func (s *Repositorio) Recontar(ctx context.Context, f dominio.Filtro) (dominio.Recuento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Recuento{}, err
	}
	inicio, fin := fechas(f)
	r := dominio.Recuento{Anio: f.Anio, EquipoID: f.EquipoID, PorOrganismo: make(map[string]int64)}
	// Una sola sentencia mantiene comisiones e informes en la misma instantánea.
	consulta := `
	 SELECT 'comision'::text,''::text,count(*) FROM actividad.comision_tecnica
	 WHERE NOT retirada AND fecha >= $1 AND fecha < $2 AND ($3::uuid IS NULL OR equipo_id=$3::uuid)
	 UNION ALL
	 SELECT 'informe'::text,organismo,count(*) FROM actividad.informe_tecnico
	 WHERE NOT retirada AND fecha >= $1 AND fecha < $2 AND ($3::uuid IS NULL OR equipo_id=$3::uuid)
	 GROUP BY organismo`
	if s.CalculoInterno {
		consulta = strings.ReplaceAll(consulta, "actividad.", "estadistica.calculo_")
	}
	filas, err := tx.Query(ctx, consulta, inicio, fin, equipo(f))
	if err != nil {
		return dominio.Recuento{}, errorSQL(err)
	}
	defer filas.Close()
	for filas.Next() {
		var tipo, codigo string
		var total int64
		if err := filas.Scan(&tipo, &codigo, &total); err != nil {
			return dominio.Recuento{}, errorSQL(err)
		}
		if tipo == "comision" {
			r.Comisiones = total
		} else {
			r.PorOrganismo[codigo] = total
			r.Informes += total
		}
	}
	if err := filas.Err(); err != nil {
		return dominio.Recuento{}, errorSQL(err)
	}
	return r, nil
}
