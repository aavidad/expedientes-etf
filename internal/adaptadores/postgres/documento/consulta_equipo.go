package documento

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
)

func (Repositorio) EmisionesEquipo(ctx context.Context, id string) ([]salida.EmisionEquipo, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT e.id::text,e.tipo_documento,e.generada_en,e.generada_por::text,p.version FROM documento.emision e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.expediente_id=$1 ORDER BY e.generada_en DESC,e.id LIMIT 100`, id) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	r := []salida.EmisionEquipo{}
	for filas.Next() {
		var e salida.EmisionEquipo
		if err := filas.Scan(&e.ID, &e.Tipo, &e.Fecha, &e.Autor, &e.VersionPlantilla); err != nil {
			return nil, err
		}
		r = append(r, e)
	}
	return r, filas.Err()
}
