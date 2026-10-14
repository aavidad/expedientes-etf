package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type SalidasServicio interface {
	BloquearSalida(context.Context, string) (d.Derivacion, error)
	DatosSalida(context.Context, string) (d.DatosParaDocumento, error)
	RepeticionSalida(context.Context, string, string, string, string, time.Time) (d.SalidaServicio, bool, error)
	RecordarSalida(context.Context, string, string, string, string, d.SalidaServicio, time.Time) error
	CrearSalida(context.Context, d.SalidaServicio, string, time.Time) error
	LeerSalida(context.Context, string, string) (d.SalidaServicio, error)
	ConfirmarSalida(context.Context, d.SalidaServicio, string, time.Time) error
}
