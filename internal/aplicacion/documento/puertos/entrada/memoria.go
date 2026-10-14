package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type SolicitudMemoria = d.SolicitudMemoria
type ResultadoMemoria = d.ResultadoMemoria
type EmisionesMemoria interface {
	EmitirMemoria(context.Context, d.Actor, d.SolicitudMemoria) (d.ResultadoMemoria, error)
	RechazarMemoria(context.Context, d.Actor) error
}
