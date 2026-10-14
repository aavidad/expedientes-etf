package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type Bandeja interface {
	ConsultarBandeja(context.Context, dominio.FiltroBandeja, string) (dominio.ResultadoBandeja, error)
}
