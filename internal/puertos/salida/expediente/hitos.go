package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Hitos interface {
	RepeticionHito(context.Context, string, string, string, string, time.Time) (dominio.ResultadoHito, bool, error)
	BloquearHito(context.Context, string) (dominio.SituacionHitos, error)
	ConfirmarHito(context.Context, dominio.SituacionHitos, dominio.ConfirmacionHito, string, time.Time) error
	RecordarHito(context.Context, string, string, string, string, dominio.ResultadoHito, time.Time) error
	ConsultarHitos(context.Context, string) ([]dominio.Hito, error)
}
