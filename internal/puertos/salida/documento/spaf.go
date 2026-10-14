package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

// RepositorioSPAF conserva sólo recibos, versiones y huellas, nunca la tabla.
type RepositorioSPAF interface {
	BloquearSPAF(context.Context, d.Actor, d.SolicitudSPAF) error
	PeticionSPAF(context.Context, d.Actor, string) (d.Emision, string, error)
	Activa(context.Context, string, string) (d.Plantilla, error)
	UltimaSPAF(context.Context, d.SolicitudSPAF) (d.Emision, error)
	GuardarSPAF(context.Context, d.Actor, d.SolicitudSPAF, string, d.Emision) error
}

// ReferenciasPresentacion usa la proyección pública de paquetes publicados.
type ReferenciasPresentacion interface {
	ResolverCatalogo(context.Context, string) (p.ReferenciaCatalogo, error)
}
