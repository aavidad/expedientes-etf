package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type FiltroEspera struct{ Limite, Desplazamiento int }

type FilaEspera struct {
	ID             string    `json:"id"`
	Estado         string    `json:"estado"`
	Version        int       `json:"version"`
	FechaRecepcion time.Time `json:"fecha_recepcion"`
	FechaEspera    string    `json:"fecha_lista_espera"`
}

type Decisiones interface {
	RepeticionDecision(context.Context, string, string, string, string, time.Time) (dominio.ResultadoDecision, bool, error)
	Decidir(context.Context, dominio.NoAceptacion, dominio.ResultadoDecision, string, string, time.Time) error
	RecordarDecision(context.Context, string, string, string, string, dominio.ResultadoDecision, time.Time) error
	ListarEspera(context.Context, FiltroEspera) ([]FilaEspera, error)
}
