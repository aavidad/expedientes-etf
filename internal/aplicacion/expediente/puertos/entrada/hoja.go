package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type SubidaHoja struct {
	EquipoID  string
	Formato   string
	Contenido []byte
}

type InformeHoja = d.InformeHoja
type IncidenciaHoja = d.IncidenciaHoja

var ErrHojaFormato = d.ErrHojaFormato
var ErrHojaLimites = d.ErrHojaLimites
var ErrHojaAntivirus = d.ErrHojaAntivirus
var ErrHojaCambiada = d.ErrHojaCambiada

type HojasHistoricas interface {
	ValidarHoja(context.Context, Actor, SubidaHoja) (InformeHoja, error)
	ConfirmarHoja(context.Context, Actor, string, string, string, SubidaHoja) (ResultadoCargaHistorica, InformeHoja, error)
	RechazarCargaHistorica(context.Context, Actor) error
}

// TextosHoja permite traducir el informe sin incluir mensajes en Go.
type TextosHoja interface {
	Textos(context.Context, string) (map[string]string, error)
}
