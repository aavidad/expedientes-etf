package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type PeticionSPM = d.PeticionSPM
type SalidaSPM = d.SalidaSPM

var ErrSPM = d.ErrSPM

type SalidasSPM interface {
	OperarSPM(context.Context, Actor, string, string, string, PeticionSPM) (SalidaSPM, error)
	ConsultarSalidaSPM(context.Context, Actor, string) (*SalidaSPM, error)
	RechazarSPM(context.Context, Actor, string) error
}

// ConsultaCierreSPM participa en la transacción del cierre. El llamador confirma
// la auditoría antes de entregar datos; no abre una segunda transacción.
type ConsultaCierreSPM interface {
	// ConsultarSalidaSPM devuelve el forzado sólo cuando corresponde a la versión vigente.
	ConsultarSalidaSPM(context.Context, Actor, string) (*SalidaSPM, error)
	PermiteCierreOrdinarioSPM(context.Context, Actor, string) (bool, error)
}
