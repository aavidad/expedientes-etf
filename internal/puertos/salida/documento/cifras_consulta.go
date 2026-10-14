package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type GeneradorCifras interface {
	CamposCifras(context.Context, e.CifrasConsulta, d.PresentacionMemoria) (map[string]string, error)
	Generar(context.Context, []byte, d.Relleno) ([]byte, error)
}
