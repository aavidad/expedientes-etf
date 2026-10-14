package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type PublicarPlantilla = d.PublicarPlantilla
type ReferenciaPlantilla = d.ReferenciaPlantilla
type ConflictoPlantilla = d.ConflictoPlantilla

const MaximoPlantilla = d.MaximoPlantilla

var (
	ErrTamanoPlantilla  = d.ErrTamanoPlantilla
	ErrMarcadores       = d.ErrMarcadores
	ErrAntivirus        = d.ErrAntivirus
	ErrInfectada        = d.ErrInfectada
	ErrVersionPlantilla = d.ErrVersionPlantilla
)

type Plantillas interface {
	Actual(context.Context, Actor, string, string) (ReferenciaPlantilla, error)
	Publicar(context.Context, Actor, PublicarPlantilla, []byte) (ReferenciaPlantilla, error)
	Rechazar(context.Context, Actor) error
}

type VersionPlantilla = d.VersionPlantilla
type ActivarPlantilla = d.ActivarPlantilla
type DetalleMarcadores = d.DetalleMarcadores
type GestionPlantillas interface {
	Listar(context.Context, Actor) ([]VersionPlantilla, error)
	Subir(context.Context, Actor, PublicarPlantilla, []byte) (ReferenciaPlantilla, error)
	Activar(context.Context, Actor, ActivarPlantilla) (ReferenciaPlantilla, error)
	Rechazar(context.Context, Actor) error
}
