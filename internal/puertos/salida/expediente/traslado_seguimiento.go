package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Traslados interface {
	RepetirTraslado(context.Context, string, string, string, string, time.Time) (d.ResultadoTraslado, bool, error)
	RecordarTraslado(context.Context, string, string, string, string, d.ResultadoTraslado, time.Time) error
	CrearTraslado(context.Context, string, string, string, d.PeticionTraslado, d.CopiaTraslado, string, time.Time) error
}
type Seguimientos interface {
	ObtenerSeguimiento(context.Context, string, bool) (*d.Seguimiento, error)
	RepetirSeguimiento(context.Context, string, string, string, string, time.Time) (d.Seguimiento, bool, error)
	RecordarSeguimiento(context.Context, string, string, string, string, d.Seguimiento, time.Time) error
	IniciarSeguimiento(context.Context, string, string, string, d.PeticionSeguimiento, d.ReglaSeguimiento, string, time.Time) (d.Seguimiento, error)
	CambiarSeguimiento(context.Context, d.Seguimiento, string, d.PeticionSeguimiento, d.ReglaSeguimiento, string, time.Time) (d.Seguimiento, error)
	AvisosSeguimiento(context.Context, string, int) ([]d.AvisoSeguimiento, error)
	VencerSeguimientos(context.Context) ([]string, error)
}
