package estadistica

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

// CerradasConsulta sólo devuelve la proyección numérica de la revisión vigente.
type CerradasConsulta interface {
	CifrasCerradas(context.Context, d.FiltroMemoria) (d.CifrasConsulta, bool, error)
}

// CalculoConsulta usa un ejecutor técnico separado, sin permisos de etf_app.
type CalculoConsulta interface {
	CalcularConsulta(context.Context, d.ActorMemoria, d.FiltroMemoria, time.Time) (d.CifrasConsulta, error)
}
type WordConsulta interface {
	GenerarCifras(context.Context, d.CifrasConsulta, string) ([]byte, error)
}
type TextosConsulta interface {
	LeerTextosCifras(context.Context, string) (map[string]string, error)
}
