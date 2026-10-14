package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type LectorHoja interface {
	LeerHoja(context.Context, string, []byte) ([]d.FilaHoja, error)
}
