package catalogo

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"time"
)

type Numeracion interface {
	ReglaVigente(context.Context, time.Time) (dominio.ReglaNumeracion, error)
}
