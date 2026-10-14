package catalogo

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"time"
)

type Repositorio interface {
	Vigente(context.Context, string, time.Time) (dominio.Version, error)
	Versiones(context.Context, string) ([]dominio.Version, error)
	Datos(context.Context, dominio.Version) (dominio.Datos, error)
}
