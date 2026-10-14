package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type Actor = dominio.Actor
type Derivacion = dominio.Derivacion
type Confirmacion = dominio.Confirmacion
type ReciboDerivacion = dominio.ReciboDerivacion

const MaxBytes = dominio.MaxBytes

var (
	ErrAcceso       = dominio.ErrAcceso
	ErrTamano       = dominio.ErrTamano
	ErrFormato      = dominio.ErrFormato
	ErrInfectado    = dominio.ErrInfectado
	ErrNoDisponible = dominio.ErrNoDisponible
	ErrNoEncontrada = dominio.ErrNoEncontrada
	ErrPeticion     = dominio.ErrPeticion
)

type Subida struct {
	Nombre    string
	Contenido []byte
}

type Derivaciones interface {
	Etiquetas(context.Context, Actor, string) (map[string]string, error)
	ConfirmarDerivacion(context.Context, Actor, string, Confirmacion) (ReciboDerivacion, error)
	Descartar(context.Context, Actor, string) error
	Subir(context.Context, Actor, Subida) (Derivacion, error)
	Consultar(context.Context, Actor, string) (Derivacion, error)
	Rechazar(context.Context, Actor, string) error
}
