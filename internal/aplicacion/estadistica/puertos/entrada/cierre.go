package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

type CierresMemoria interface {
	EstadoMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria) (d.EstadoMemoria, error)
	CerrarMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria, string, d.ActoMemoria) (d.ResultadoActoMemoria, error)
	ReabrirMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria, string, d.ActoMemoria) (d.ResultadoActoMemoria, error)
	InstantaneaMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria, int) (d.RevisionMemoria, error)
	DiscrepanciasMemoria(context.Context, d.ActorMemoria, d.FiltroMemoria) (d.DiscrepanciasMemoria, error)
	RechazarCierreMemoria(context.Context, d.ActorMemoria) error
}
type FuenteCierreMemoria interface {
	FuenteMemoria
	EquiposMemoria(context.Context) ([]string, error)
	MemoriaFijadaEnTransaccion(context.Context, d.ActorMemoria, d.InformeMemoria, time.Time) (d.InformeMemoria, error)
}
type ActoMemoria = d.ActoMemoria
type EstadoMemoria = d.EstadoMemoria
type RevisionMemoria = d.RevisionMemoria
type ResultadoActoMemoria = d.ResultadoActoMemoria
type DiscrepanciaMemoria = d.DiscrepanciaMemoria
type DiscrepanciasMemoria = d.DiscrepanciasMemoria

var (
	ErrActoMemoria         = d.ErrActoMemoria
	ErrIdempotenciaMemoria = d.ErrIdempotenciaMemoria
	ErrRevisionMemoria     = d.ErrRevisionMemoria
	ErrBorradorMemoria     = d.ErrBorradorMemoria
)
