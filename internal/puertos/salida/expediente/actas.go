package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type LectorActas interface {
	LeerActa(context.Context, string, []byte) (d.Extraccion, error)
}
