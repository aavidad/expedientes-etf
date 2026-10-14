package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type FiltroBandeja = dominio.FiltroBandeja
type ResultadoBandeja = dominio.ResultadoBandeja
type Bandeja interface {
	ConsultarBandeja(context.Context, Actor, FiltroBandeja) (ResultadoBandeja, error)
}
