package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Menores interface {
	MenoresPertenecen(context.Context, string, []string) (bool, error)
	BloquearMenores(context.Context, string) (dominio.FichaMenores, error)
	ObtenerMenores(context.Context, string) (dominio.FichaMenores, error)
	GuardarMenores(context.Context, string, dominio.CambioMenores, string, time.Time) error
	RepetirMenores(context.Context, string, string, string, string, time.Time) (dominio.ResultadoMenores, bool, error)
	RecordarMenores(context.Context, string, string, string, string, dominio.ResultadoMenores, time.Time) error
	HistoriaMenores(context.Context, string, string) (dominio.HistoriaMenores, error)
}
type CatalogosMenores interface {
	Obtener(context.Context) (dominio.CatalogosMenores, error)
}

// ConsultaMenores se usa en la transacción del consumidor F2.05 o F4.05,
// con sesión fijada. Audita la lectura en ella; el consumidor confirma antes
// de entregar datos. Devuelve sólo atributos estadísticos, nunca nombres.
type ConsultaMenores interface {
	MenoresVigentes(context.Context, string, string) ([]dominio.MenorConsulta, error)
}
