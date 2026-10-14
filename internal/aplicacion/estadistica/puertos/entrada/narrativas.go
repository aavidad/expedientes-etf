package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type Narrativas interface {
	LeerNarrativas(context.Context, d.ActorMemoria, d.FiltroMemoria) (d.Narrativas, error)
	GuardarNarrativas(context.Context, d.ActorMemoria, d.Narrativas) (d.Narrativas, error)
}

type ActorMemoria = d.ActorMemoria
type FiltroMemoria = d.FiltroMemoria
type NarrativaMemoria = d.Narrativas
type ConflictoMemoria = d.ConflictoMemoria

var (
	ErrPermisoMemoria    = d.ErrPermisoMemoria
	ErrNarrativasMemoria = d.ErrNarrativas
	ErrDisponibleMemoria = d.ErrDisponibleMemoria
	ErrCerradaMemoria    = d.ErrCerradaMemoria
)
