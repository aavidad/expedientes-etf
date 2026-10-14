package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// ConsultaAvisos reutiliza las proyecciones históricas de familia, menores e
// intervención por lotes. Se invoca en la transacción con identidad fijada.
type ConsultaAvisos interface {
	DatosAvisos(context.Context, string, string, string) ([]dominio.DatosAvisos, error)
}
