package identidad

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type Restauracion interface {
	Leer(context.Context) (dominio.RevisionRestauracion, error)
	Resolver(context.Context, string, string, int64, string) error
	Confirmar(context.Context, string, int64) error
}
