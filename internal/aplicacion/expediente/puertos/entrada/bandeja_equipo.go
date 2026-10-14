package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type FiltroEquipo = d.FiltroEquipo
type PaginaEquipo = d.PaginaEquipo
type FichaEquipo = d.FichaEquipo

type BandejaEquipo interface {
	EtiquetasEquipo(context.Context, Actor, string) (map[string]string, error)
	ListarEquipo(context.Context, Actor, FiltroEquipo) (PaginaEquipo, error)
	AbrirFichaEquipo(context.Context, Actor, string) (FichaEquipo, error)
}
