package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type PeticionTraslado = d.PeticionTraslado
type ResultadoTraslado = d.ResultadoTraslado
type PeticionSeguimiento = d.PeticionSeguimiento
type Seguimiento = d.Seguimiento
type AvisoSeguimiento = d.AvisoSeguimiento

var ErrTraslado = d.ErrTraslado
var ErrSeguimiento = d.ErrSeguimiento

type TrasladoSeguimiento interface {
	Trasladar(context.Context, Actor, string, string, PeticionTraslado) (ResultadoTraslado, error)
	Seguimiento(context.Context, Actor, string, string, string, PeticionSeguimiento) (Seguimiento, error)
	ConsultarSeguimiento(context.Context, Actor, string) (*Seguimiento, error)
	ConsultarAvisosSeguimiento(context.Context, Actor) ([]AvisoSeguimiento, error)
	RechazarPropuesta(context.Context, Actor, string) error
}
