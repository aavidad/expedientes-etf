package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type NoAceptacion = dominio.NoAceptacion
type Espera = dominio.Espera
type ResultadoDecision = dominio.ResultadoDecision
type FiltroEspera struct{ Limite, Desplazamiento int }
type FilaEspera struct {
	ID             string    `json:"id"`
	Estado         string    `json:"estado"`
	Version        int       `json:"version"`
	FechaRecepcion time.Time `json:"fecha_recepcion"`
	FechaEspera    string    `json:"fecha_lista_espera"`
}

var (
	ErrMotivo       = dominio.ErrMotivo
	ErrSubprograma  = dominio.ErrSubprograma
	ErrOrigenEspera = dominio.ErrOrigenEspera
	ErrFechaEspera  = dominio.ErrFechaEspera
)

type Decisiones interface {
	NoAceptar(context.Context, Actor, string, string, NoAceptacion) (ResultadoDecision, error)
	DejarEnEspera(context.Context, Actor, string, string, Espera) (ResultadoDecision, error)
	ListarEspera(context.Context, Actor, FiltroEspera) ([]FilaEspera, error)
	RechazarDecision(context.Context, Actor, string) error
}
