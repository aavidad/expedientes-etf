package catalogo

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

type SMI interface {
	SMI(context.Context) ([]dominio.SMI, error)
	GuardarSMI(context.Context, dominio.CambioSMI) (int, error)
}
