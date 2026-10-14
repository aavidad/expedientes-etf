package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type ConsultaCifrasMemoria interface {
	RechazarPeticionMemoria(context.Context, d.ActorMemoria) error
	ConsultarCifras(context.Context, d.ActorMemoria, d.FiltroMemoria) (d.CifrasConsulta, error)
	DescargarCifras(context.Context, d.ActorMemoria, d.FiltroMemoria, string) ([]byte, error)
}
type NavegacionMemoria interface {
	RechazarPeticionMemoria(context.Context, d.ActorMemoria) error
	ConsultarMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria) (d.VistaMemoria, error)
	ComposicionMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria, d.Apartado) ([]d.DesgloseMemoria, error)
}
type CifrasConsulta = d.CifrasConsulta
type VistaMemoria = d.VistaMemoria
type DesgloseMemoria = d.DesgloseMemoria
type Apartado = d.Apartado
