package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type CambioMenores = dominio.CambioMenores
type FichaMenores = dominio.FichaMenores
type ResultadoMenores = dominio.ResultadoMenores
type HistoriaMenores = dominio.HistoriaMenores

var ErrMenores = dominio.ErrMenores

type Menores interface {
	EditarMenores(context.Context, Actor, string, string, CambioMenores) (ResultadoMenores, error)
	ConsultarMenores(context.Context, Actor, string) (FichaMenores, error)
	ConsultarHistoriaMenores(context.Context, Actor, string, string) (HistoriaMenores, error)
	RechazarMenores(context.Context, Actor) error
}
