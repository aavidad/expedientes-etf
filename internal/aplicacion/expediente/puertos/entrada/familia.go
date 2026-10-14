package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type CambioFamilia = dominio.CambioFamilia
type FichaFamilia = dominio.FichaFamilia
type ResultadoFamilia = dominio.ResultadoFamilia
type HistoriaFamilia = dominio.HistoriaFamilia

var ErrFamilia = dominio.ErrFamilia

type Familias interface {
	EditarFamilia(context.Context, Actor, string, string, CambioFamilia) (ResultadoFamilia, error)
	ConsultarFamilia(context.Context, Actor, string) (FichaFamilia, error)
	ConsultarHistoriaFamilia(context.Context, Actor, string, string, int) (HistoriaFamilia, error)
	RechazarFamilia(context.Context, Actor) error
}
