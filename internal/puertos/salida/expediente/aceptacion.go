package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Aceptaciones interface {
	// Repeticion serializa por usuario, operación y clave dentro de la transacción.
	Repeticion(context.Context, string, string, string, string, time.Time) (dominio.ResultadoAceptacion, bool, error)
	Bloquear(context.Context, string) (dominio.Derivacion, error)
	Siguiente(context.Context, string, int, int) (int, error)
	Aceptar(context.Context, dominio.Aceptacion, dominio.ResultadoAceptacion, string, time.Time) error
	Recordar(context.Context, string, string, string, string, dominio.ResultadoAceptacion, time.Time) error
}
