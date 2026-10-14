package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type ConfirmacionHito = dominio.ConfirmacionHito
type ResultadoHito = dominio.ResultadoHito
type Hito = dominio.Hito

var (
	ErrHitoTipo  = dominio.ErrHitoTipo
	ErrHitoFecha = dominio.ErrHitoFecha
)

type Hitos interface {
	ConfirmarHito(context.Context, Actor, string, string, ConfirmacionHito) (ResultadoHito, error)
	ConsultarHitos(context.Context, Actor, string) ([]Hito, error)
	RechazarHito(context.Context, Actor, string) error
}
