package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Familias interface {
	AdultosPertenecen(context.Context, string, []string) (bool, error)
	BloquearFamilia(context.Context, string) (dominio.FichaFamilia, error)
	ObtenerFamilia(context.Context, string) (dominio.FichaFamilia, error)
	GuardarFamilia(context.Context, string, dominio.CambioFamilia, string, time.Time) error
	RepetirFamilia(context.Context, string, string, string, string, time.Time) (dominio.ResultadoFamilia, bool, error)
	RecordarFamilia(context.Context, string, string, string, string, dominio.ResultadoFamilia, time.Time) error
	HistoriaFamilia(context.Context, string, string) (dominio.HistoriaFamilia, error)
}
type CatalogosFamilia interface {
	Obtener(context.Context) (dominio.CatalogosFamilia, error)
}

// ConsultaAdultos se invoca dentro de la transacción del consumidor, con el
// contexto de sesión ya fijado. La lectura se audita antes de devolver datos.
type ConsultaAdultos interface {
	AdultosVigentes(context.Context, string, string) ([]dominio.AdultoConsulta, error)
}
