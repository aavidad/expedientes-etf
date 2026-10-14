package catalogo

import (
	"context"
	"errors"
	"strconv"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//etf:tecnico
func (*Repositorio) SMI(ctx context.Context) ([]dominio.SMI, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT anio,importe_mensual_centimos,importe_anual_centimos,version FROM catalogo.smi ORDER BY anio`)
	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	defer filas.Close()
	resultado := make([]dominio.SMI, 0, 15)
	for filas.Next() {
		var fila dominio.SMI
		var mensual, anual *int64
		if err := filas.Scan(&fila.Anio, &mensual, &anual, &fila.Version); err != nil {
			return nil, dominio.ErrPersistencia
		}
		fila.ImporteMensualCentimos, fila.ImporteAnualCentimos = mensual, anual
		fila.ImporteMensual, fila.ImporteAnual = dominio.ImporteSMI(mensual), dominio.ImporteSMI(anual)
		resultado = append(resultado, fila)
	}
	if filas.Err() != nil {
		return nil, dominio.ErrPersistencia
	}
	return resultado, nil
}

//etf:tecnico
func (*Repositorio) GuardarSMI(ctx context.Context, cambio dominio.CambioSMI) (int, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return 0, err
	}
	var version int
	err = tx.QueryRow(ctx, `SELECT catalogo.guardar_smi($1,$2,$3)`, cambio.Anio, cambio.ImporteMensualCentimos, cambio.VersionEsperada).Scan(&version)
	if err == nil {
		return version, nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code == "40001" {
			actual, _ := strconv.Atoi(pg.Detail)
			return 0, &dominio.Conflicto{Actual: actual, Recurso: "smi"}
		}
		if pg.Code == "42501" {
			return 0, dominio.ErrPermiso
		}
		if pg.Code == "23514" || pg.Code == "22003" || pg.Message == "catalogo.edicion_invalida" {
			return 0, dominio.ErrSMI
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, dominio.ErrSMI
	}
	return 0, dominio.ErrPersistencia
}
