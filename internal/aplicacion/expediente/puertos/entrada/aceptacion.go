package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type Aceptacion = dominio.Aceptacion
type ResultadoAceptacion = dominio.ResultadoAceptacion
type Conflicto = dominio.Conflicto

var (
	ErrObligatorios = dominio.ErrObligatorios
	ErrProfesional  = dominio.ErrProfesional
	ErrSexo         = dominio.ErrSexo
	ErrEstado       = dominio.ErrEstado
	ErrIdempotencia = dominio.ErrIdempotencia
	ErrClave        = dominio.ErrClave
	ErrNumeracion   = dominio.ErrNumeracion
)

type Aceptaciones interface {
	Aceptar(context.Context, Actor, string, string, Aceptacion) (ResultadoAceptacion, error)
	Consultar(context.Context, Actor, string) (Derivacion, error)
	RechazarAceptacion(context.Context, Actor, string) error
}
