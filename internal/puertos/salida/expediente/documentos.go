package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// LecturaDocumentos devuelve una versión coherente con la transacción recibida.
type LecturaDocumentos interface {
	ParaDocumentos(context.Context, string) (d.DatosParaDocumento, error)
}
