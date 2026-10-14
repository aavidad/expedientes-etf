package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type SolicitudRegistroSalida = d.SolicitudRegistroSalida
type RegistroSalida = d.RegistroSalida
type ConflictoRegistro = d.ConflictoRegistro

var ErrRegistroOficio = d.ErrRegistroOficio

type RegistrosSalida interface {
	AnotarRegistroSalida(context.Context, Actor, SolicitudRegistroSalida) (RegistroSalida, error)
	Rechazar(context.Context, Actor, string) error
}
