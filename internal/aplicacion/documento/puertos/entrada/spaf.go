package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type SolicitudSPAF = d.SolicitudSPAF
type ConflictoSPAF = d.ConflictoSPAF
type EmisionesSPAF interface {
	EmitirSPAF(context.Context, Actor, SolicitudSPAF) (Resultado, error)
	RechazarSPAF(context.Context, Actor) error
}
