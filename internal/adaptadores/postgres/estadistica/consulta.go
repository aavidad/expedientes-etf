package estadistica

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

//etf:tecnico
func (*Repositorio) CifrasCerradas(ctx context.Context, f d.FiltroMemoria) (d.CifrasConsulta, bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.CifrasConsulta{}, false, err
	}
	var b []byte
	err = tx.QueryRow(ctx, `SELECT estadistica.api_cifras_consulta($1,nullif($2,'')::uuid)`, f.Anio, f.EquipoID).Scan(&b)
	if err != nil {
		return d.CifrasConsulta{}, false, err
	}
	if len(b) == 0 {
		return d.CifrasConsulta{}, false, nil
	}
	var r d.CifrasConsulta
	err = json.Unmarshal(b, &r)
	return r, err == nil, err
}
