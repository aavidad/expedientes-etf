package postgres

import (
	"context"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

type HistorialExpediente struct{}

func (HistorialExpediente) HistorialExpediente(ctx context.Context, id string) ([]salida.ApunteExpediente, error) {
	tx, err := DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT id::text,accion,ocurrido_en,coalesce(usuario_id::text,''),resultado FROM auditoria.historial_expediente($1)`, id) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	r := []salida.ApunteExpediente{}
	for filas.Next() {
		var e salida.ApunteExpediente
		if err := filas.Scan(&e.ID, &e.Accion, &e.Fecha, &e.Autor, &e.Resultado); err != nil {
			return nil, err
		}
		r = append(r, e)
	}
	return r, filas.Err()
}
