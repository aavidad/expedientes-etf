package documento

import "context"

// ComprobacionFinalizacion consulta emisiones dentro de la transacción del cierre.
type ComprobacionFinalizacion interface {
	FinalizacionEmitida(context.Context, string, int) (bool, error)
}
