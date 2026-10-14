package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type AvisosExpediente = dominio.AvisosExpediente
type RecuentoAvisos = dominio.RecuentoAvisos
type Avisos interface {
	ConsultarAvisos(context.Context, Actor, string) (AvisosExpediente, error)
	ContarAvisos(context.Context, Actor, string) (RecuentoAvisos, error)
	RechazarAvisos(context.Context, Actor) error
}
