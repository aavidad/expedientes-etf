package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type Actor = dominio.Actor
type Referencia = dominio.Referencia
type Preparacion = dominio.Preparacion

var (
	ErrPaqueteInvalido = dominio.ErrPaqueteInvalido
	ErrConflicto       = dominio.ErrConflicto
	ErrPermiso         = dominio.ErrPermiso
	ErrAntivirus       = dominio.ErrAntivirus
	ErrInfectado       = dominio.ErrInfectado
	ErrPreparacion     = dominio.ErrPreparacion
	ErrOcupado         = dominio.ErrOcupado
)

type Instalacion interface {
	Preparar(context.Context, Actor, string, string, []byte) (Preparacion, error)
	Activar(context.Context, Actor, string, int64) (Referencia, error)
	Actual(context.Context, Actor, string, string) (Referencia, error)
	Descartar(context.Context, Actor, string) error
}
