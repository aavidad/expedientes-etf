package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type DesgloseCifraMemoria interface {
	DesglosarMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria, d.Apartado, d.SeleccionDesglose) (d.DetalleMemoria, error)
	RechazarPeticionMemoria(context.Context, d.ActorMemoria) error
}
type SeleccionDesglose = d.SeleccionDesglose

type ClaveCeldaMemoria = d.ClaveCelda
