package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Borradores interface {
	Poner(context.Context, dominio.Derivacion) error
	Obtener(context.Context, string, string) (dominio.Derivacion, error)
	Descartar(context.Context, string, string) error
}
type Confirmaciones interface {
	RepetirConfirmacion(context.Context, string, string, string, time.Time) (dominio.ReciboDerivacion, bool, error)
	// Serializa por lectura y rechaza otra confirmación de la misma lectura.
	BloquearLectura(context.Context, string) error
	RecordarConfirmacion(context.Context, string, string, string, dominio.ReciboDerivacion, time.Time) error
}

type EtiquetasUbicacion interface {
	NombresEn(context.Context, string) (map[string]string, error)
}
