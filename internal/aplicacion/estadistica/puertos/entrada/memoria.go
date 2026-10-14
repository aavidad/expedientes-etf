package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

// FuenteMemoria participa en la transacción de emisión de documento: no abre
// otra transacción ni devuelve datos a HTTP sin la auditoría del llamador.
type FuenteMemoria interface {
	MemoriaEnTransaccion(context.Context, d.ActorMemoria, d.FiltroMemoria, time.Time) (d.InformeMemoria, error)
}

// ConsultaMemoria entrega exclusivamente cifras, tras confirmar la auditoría.
type ConsultaMemoria interface {
	ConsultarMemoria(context.Context, ActorMemoria, FiltroMemoria) (CifrasMemoria, error)
}

type ConsultaEquiposMemoria interface {
	ConsultarEquiposMemoria(context.Context, ActorMemoria) ([]EquipoCifrasMemoria, error)
}

type EquipoCifrasMemoria struct {
	ID         string `json:"id"`
	Codigo     string `json:"codigo"`
	ClaveTexto string `json:"clave_texto"`
}

type CifrasMemoria = d.CifrasMemoria
type CifrasApartadoMemoria = d.CifrasApartadoMemoria
type ContratoCifrasMemoria = d.ContratoCifrasMemoria
type TotalCifrasMemoria = d.TotalCifrasMemoria
type GrupoCifrasMemoria = d.GrupoCifrasMemoria
type DenominadorCifrasMemoria = d.DenominadorCifrasMemoria
type PorcentajeCifrasMemoria = d.PorcentajeCifrasMemoria
type CeldaCifrasMemoria = d.CeldaCifrasMemoria

var ErrCifrasNoPublicadasMemoria = d.ErrCifrasNoPublicadasMemoria
