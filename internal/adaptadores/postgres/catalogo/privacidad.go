package catalogo

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/jackc/pgx/v5"
	"time"
)

// UmbralConsulta usa la vigencia actual, no la del año de la Memoria.
func (r *Repositorio) UmbralConsulta(ctx context.Context, ahora time.Time) (int, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return 0, err
	}
	var n int
	err = tx.QueryRow(ctx, `SELECT p.valor_entero FROM catalogo.parametro p
 JOIN catalogo.version v ON v.id=p.version_id
 WHERE v.ambito='plazos' AND v.estado='publicada'
 AND daterange(v.vigente_desde,v.vigente_hasta,'[)') @> ($1::timestamptz AT TIME ZONE 'Europe/Madrid')::date
 AND p.clave='umbral_privacidad_consulta' AND p.unidad='casos'`, ahora).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 3, nil
	}
	if err != nil || n < 3 {
		return 0, d.ErrDisponibleMemoria
	}
	return n, nil
}
