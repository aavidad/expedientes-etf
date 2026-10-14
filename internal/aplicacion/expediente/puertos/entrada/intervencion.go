package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type CambioIntervencion = dominio.CambioIntervencion
type FichaIntervencion = dominio.FichaIntervencion
type ResultadoIntervencion = dominio.ResultadoIntervencion
type HistoriaIntervencion = dominio.HistoriaIntervencion

var ErrIntervencion = dominio.ErrIntervencion

type Intervencion interface {
	EditarIntervencion(context.Context, Actor, string, string, CambioIntervencion) (ResultadoIntervencion, error)
	ConsultarIntervencion(context.Context, Actor, string) (FichaIntervencion, error)
	ConsultarHistoriaIntervencion(context.Context, Actor, string, string) (HistoriaIntervencion, error)
	RechazarIntervencion(context.Context, Actor) error
}
