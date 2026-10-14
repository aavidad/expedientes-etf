package documento

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
)

func (Repositorio) FinalizacionEmitida(ctx context.Context, id string, version int) (bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return false, err
	}
	var existe bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documento.emision WHERE expediente_id=$1 AND tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12') AND (versiones_datos->>'expediente')::integer>=$2)`, id, version).Scan(&existe) //etf:tecnico
	return existe, err
}
