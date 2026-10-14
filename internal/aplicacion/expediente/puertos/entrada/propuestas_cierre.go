package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type FinalizacionGuardada = d.FinalizacionGuardada
type Finalizacion = d.Finalizacion
type VersionCierre = d.VersionCierre
type PropuestaCierre = d.PropuestaCierre

var ErrFinalizacion = d.ErrFinalizacion
var ErrConsentimientoCierre = d.ErrConsentimientoCierre

type PropuestasCierre interface {
	ConsultarFinalizacion(context.Context, Actor, string) (FinalizacionGuardada, error)
	GuardarFinalizacion(context.Context, Actor, string, string, Finalizacion) (PropuestaCierre, error)
	ProponerCierre(context.Context, Actor, string, string, VersionCierre) (PropuestaCierre, error)
	RetirarPropuestaCierre(context.Context, Actor, string, string, VersionCierre) (PropuestaCierre, error)
	ListarPropuestasCierre(context.Context, Actor, int, int) ([]PropuestaCierre, error)
	RechazarPropuesta(context.Context, Actor, string) error
}
