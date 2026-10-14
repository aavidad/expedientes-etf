package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type PeticionCierre = d.PeticionCierre
type PeticionReapertura = d.PeticionReapertura
type ResultadoCierre = d.ResultadoCierre

var ErrCierre = d.ErrCierre
var ErrDobleIntervencion = d.ErrDobleIntervencion
var ErrCierreSPM = d.ErrCierreSPM

type Cierres interface {
	Cerrar(context.Context, Actor, string, string, PeticionCierre) (ResultadoCierre, error)
	Reabrir(context.Context, Actor, string, string, PeticionReapertura) (ResultadoCierre, error)
	RechazarPropuesta(context.Context, Actor, string) error
}
