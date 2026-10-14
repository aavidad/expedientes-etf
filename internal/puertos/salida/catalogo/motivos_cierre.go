package catalogo

import (
	"context"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

type MotivosCierre interface {
	CierreVigente(context.Context, time.Time) (dominio.CatalogoCierre, error)
}
