package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
)

//etf:tecnico
func (*LecturaEstadistica) NumerosMemoria(ctx context.Context, ids []string) (map[string]string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,coalesce(numero,'') FROM expediente.expediente WHERE id=ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := map[string]string{}
	for rows.Next() {
		var id, n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		r[id] = n
	}
	return r, rows.Err()
}
