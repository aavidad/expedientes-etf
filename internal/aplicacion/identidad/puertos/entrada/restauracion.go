package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type RevisarRestauracion interface {
	Leer(context.Context, Acceso) (dominio.RevisionRestauracion, error)
	Resolver(context.Context, Acceso, string, string, int64, string) error
	Confirmar(context.Context, Acceso, string, int64) error
}
