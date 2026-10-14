package estadistica

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

type Cierres interface {
	BloquearMemorias(context.Context) error
	EstadoMemoria(context.Context, d.FiltroMemoria, bool, string, time.Time) (d.EstadoMemoria, error)
	LeerRevision(context.Context, d.FiltroMemoria, int) (d.RevisionMemoria, error)
	RevisionConocida(context.Context, d.FiltroMemoria, time.Time) (d.RevisionMemoria, bool, error)
	RepetirActo(context.Context, string, string, string, string, time.Time) (d.ResultadoActoMemoria, bool, error)
	GuardarActo(context.Context, d.EstadoMemoria, d.ActoMemoria, *d.RevisionMemoria, []d.InformeMemoria, string, string, string, string, time.Time) (d.ResultadoActoMemoria, error)
	BasesEquipo(context.Context, string) ([]d.BaseEquipoMemoria, error)
	AvisarDiscrepancia(context.Context, d.DiscrepanciaMemoria) error
	Discrepancias(context.Context, d.FiltroMemoria) ([]d.DiscrepanciaMemoria, error)
	RevisionesCerradas(context.Context, d.FiltroMemoria) ([]d.RevisionMemoria, error)
}
