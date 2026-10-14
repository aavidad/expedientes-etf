package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type PropuestasCierre interface {
	LeerFinalizacion(context.Context, string) (*d.FinalizacionGuardada, error)
	GuardarFinalizacion(context.Context, string, string, d.FinalizacionGuardada, string, time.Time) error
	Proponer(context.Context, d.PropuestaCierre) (d.PropuestaCierre, error)
	Retirar(context.Context, string, string, time.Time) (d.PropuestaCierre, error)
	Pendiente(context.Context, string) (bool, error)
	ListarPropuestas(context.Context, int, int) ([]d.PropuestaCierre, error)
	RepeticionCierre(context.Context, string, string, string, string, time.Time) (d.PropuestaCierre, bool, error)
	RecordarCierre(context.Context, string, string, string, string, d.PropuestaCierre, time.Time) error
}
