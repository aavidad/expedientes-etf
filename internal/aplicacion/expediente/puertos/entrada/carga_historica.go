package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type CargaHistorica = d.CargaHistorica
type ResultadoCargaHistorica = d.ResultadoCargaHistorica
type ValidacionCargaHistorica = d.ValidacionCargaHistorica

var ErrCargaHistorica = d.ErrCargaHistorica
var ErrNumeroHistorico = d.ErrNumeroHistorico
var ErrNumeroDuplicado = d.ErrNumeroDuplicado

type CargasHistoricas interface {
	ValidarCargaHistorica(context.Context, Actor, CargaHistorica) (ValidacionCargaHistorica, error)
	CargarHistoricos(context.Context, Actor, string, string, CargaHistorica) (ResultadoCargaHistorica, error)
	RechazarCargaHistorica(context.Context, Actor) error
}
