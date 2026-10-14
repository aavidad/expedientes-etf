package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type SalidaServicio = d.SalidaServicio
type NuevaSalidaServicio = d.NuevaSalidaServicio
type ConfirmarSalidaServicio = d.ConfirmarSalidaServicio

type SalidasServicio interface {
	CrearSalidaServicio(context.Context, Actor, string, string, NuevaSalidaServicio) (SalidaServicio, error)
	ConfirmarSalidaServicio(context.Context, Actor, string, string, ConfirmarSalidaServicio) (SalidaServicio, error)
	RechazarSalida(context.Context, Actor, string) error
	ParaAnexo6(context.Context, Actor, string, string) (DatosParaDocumento, SalidaServicio, error)
}
