package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type PlazoCarga = d.PlazoCarga
type LoteCarga = d.LoteCarga

var ErrCargaCerrada = d.ErrCargaCerrada
var ErrDuplicadosSinConfirmar = d.ErrDuplicadosSinConfirmar
var ErrCargaAgotaNumeracion = d.ErrCargaAgotaNumeracion
var ErrLoteConActividad = d.ErrLoteConActividad
var ErrControlCarga = d.ErrControlCarga

type ControlCargas interface {
	ConsultarPlazoCarga(context.Context, Actor) (PlazoCarga, error)
	FijarPlazoCarga(context.Context, Actor, PlazoCarga) (PlazoCarga, error)
	ConsultarLotesCarga(context.Context, Actor, int) ([]LoteCarga, error)
	AnularLoteCarga(context.Context, Actor, string) error
}
