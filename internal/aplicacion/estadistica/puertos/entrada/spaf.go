package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

type ActorSPAF = dominio.ActorSPAF
type FiltroSPAF = dominio.FiltroSPAF
type TablaSPAF = dominio.TablaSPAF

const (
	ErrEntradaSPAF    = dominio.ErrEntrada
	ErrPermisoSPAF    = dominio.ErrPermisoSPAF
	ErrDisponibleSPAF = dominio.ErrDisponibleSPAF
)

type ConsultaSPAF interface {
	ConsultarSPAF(context.Context, ActorSPAF, FiltroSPAF) (TablaSPAF, error)
}

type OpcionesSPAF = dominio.OpcionesSPAF
type ConsultaOpcionesSPAF interface {
	ConsultarOpcionesSPAF(context.Context, ActorSPAF) (OpcionesSPAF, error)
}

// CalculoSPAF se ejecuta dentro de la transacción de emisión del llamador.
// El llamador confirma su auditoría antes de entregar el resultado.
type CalculoSPAF interface {
	CalcularEnTransaccion(context.Context, ActorSPAF, FiltroSPAF, time.Time) (TablaSPAF, error)
}
