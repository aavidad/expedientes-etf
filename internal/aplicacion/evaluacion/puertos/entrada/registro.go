package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
)

type Actor = d.Actor
type Cambio = d.Cambio
type Registro = d.Registro
type Gestor interface {
	Leer(context.Context, Actor, string) (Registro, error)
	Guardar(context.Context, Actor, string, Cambio) (Cambio, error)
}

type Conflicto = d.Conflicto

var (
	ErrAcceso        = d.ErrAcceso
	ErrNoEncontrada  = d.ErrNoEncontrada
	ErrPeticion      = d.ErrPeticion
	ErrSujeto        = d.ErrSujeto
	ErrPuntuacion    = d.ErrPuntuacion
	ErrVersionEscala = d.ErrVersionEscala
	ErrIdempotencia  = d.ErrIdempotencia
	ErrNoDisponible  = d.ErrNoDisponible
)

// DatosDocumentos lee dentro de la transacción de emisión; el llamador la audita.
type DatosDocumentos interface {
	ParaDocumentos(context.Context, Actor, string) (d.DatosDocumento, error)
}
