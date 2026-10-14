package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"strings"
)

func (*Repositorio) NombresSujetos(ctx context.Context, id string) (map[string]string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,coalesce(datos->>'nombre',''),coalesce(datos->>'apellidos','')
 FROM expediente.adulto WHERE expediente_id=$1
 UNION ALL SELECT id::text,coalesce(datos->>'nombre',''),coalesce(datos->>'apellidos','')
 FROM expediente.menor WHERE expediente_id=$1`, id) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nombres := map[string]string{}
	for rows.Next() {
		var sujeto, nombre, apellidos string
		if err := rows.Scan(&sujeto, &nombre, &apellidos); err != nil {
			return nil, err
		}
		nombres[sujeto] = strings.TrimSpace(nombre + " " + apellidos)
	}
	return nombres, rows.Err()
}
