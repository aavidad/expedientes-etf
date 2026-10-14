package estadistica

import (
	"context"
	c "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

type ReglasMemoria interface {
	ReglasMemoria(context.Context, int, time.Time) (d.Historia[d.Parametros], error)
}
type SMIMemoria interface {
	SMI(context.Context) ([]c.SMI, error)
}
