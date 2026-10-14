package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Intervenciones interface {
	BloquearIntervencion(context.Context, string) (dominio.FichaIntervencion, error)
	ObtenerIntervencion(context.Context, string) (dominio.FichaIntervencion, error)
	GuardarIntervencion(context.Context, string, dominio.CambioIntervencion, string, time.Time) error
	RepetirIntervencion(context.Context, string, string, string, string, time.Time) (dominio.ResultadoIntervencion, bool, error)
	RecordarIntervencion(context.Context, string, string, string, string, dominio.ResultadoIntervencion, time.Time) error
	HistoriaIntervencion(context.Context, string, string) (dominio.HistoriaIntervencion, error)
}
type CatalogosIntervencion interface {
	ObtenerIntervencion(context.Context, string) (dominio.CatalogosIntervencion, error)
}

// ConsultaIntervencion sirve a F4.05 y F2.02 dentro de su transacción con
// sesión fijada. Audita en ella; el consumidor confirma antes de entregar.
// Una vigencia nula significa que no hay datos conocidos al corte.
type ConsultaIntervencion interface {
	IntervencionVigente(context.Context, string, string) (dominio.HistoriaIntervencion, error)
}
