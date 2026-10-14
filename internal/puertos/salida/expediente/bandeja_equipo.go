package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// Bandeja sólo proyecta datos de expediente; RLS conserva el ámbito de sesión.
type BandejaEquipo interface {
	ListarEquipo(context.Context, d.FiltroEquipo) (d.PaginaEquipo, error)
	FichaEquipo(context.Context, string) (d.FichaEquipo, error)
}

// ConsultaAvisosEquipo proyecta los datos de avisos sólo para la página visible.
type ConsultaAvisosEquipo interface {
	DatosAvisosEquipo(context.Context, []string, string, string) ([]d.DatosAvisos, error)
}
