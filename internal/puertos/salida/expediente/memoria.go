package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

// LecturaMemoria entrega dimensiones históricas sin nombres ni textos libres.
// Comparte la transacción y la identidad RLS del caso de uso. Equipo vacío
// solicita la provincia; sólo Coordinación puede abrir ese contexto.
type LecturaMemoria interface {
	ConsultarMemoria(context.Context, string, time.Time, time.Time) ([]d.Expediente, error)
}

type EquipoMemoria interface {
	EquipoMemoria(context.Context, string) (string, time.Time, error)
}
