package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

const ActaCierre = d.ActaCierre

const Anexo1 = d.Anexo1
const Anexo2 = d.Anexo2

type ConflictoVersion = d.ConflictoVersion
type Actor = d.Actor
type Solicitud = d.Solicitud
type Resultado = d.Resultado
type ConflictoReproduccion = d.ConflictoReproduccion

var (
	ErrNoAceptada   = d.ErrNoAceptada
	ErrListaEspera  = d.ErrListaEspera
	ErrAcceso       = d.ErrAcceso
	ErrPeticion     = d.ErrPeticion
	ErrNoDisponible = d.ErrNoDisponible
	ErrPlantilla    = d.ErrPlantilla
	ErrHuella       = d.ErrHuella
	ErrFirmante     = d.ErrFirmante
	ErrIdempotencia = d.ErrIdempotencia
)

type Emisiones interface {
	Emitir(context.Context, Actor, Solicitud) (Resultado, error)
	Rechazar(context.Context, Actor, string) error
}

func IDActorEmisionValido(a Actor) bool { return d.IDValido(a.UsuarioID) }
