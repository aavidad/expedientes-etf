package catalogo

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

type Edicion interface {
	Ultima(context.Context, string) (dominio.Datos, error)
	Publicar(context.Context, string, dominio.Cambio) (dominio.Version, error)
	Firmantes(context.Context) ([]dominio.Firmante, error)
	GuardarFirmante(context.Context, dominio.Cambio) (dominio.Firmante, error)
}
