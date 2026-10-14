package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type DatosParaDocumento = d.DatosParaDocumento

// DatosDocumentos lee la derivación, recepción y decisión dentro de la transacción
// del llamador. No inicia otra transacción ni modifica hitos o datos.
type DatosDocumentos interface {
	ParaDocumentos(context.Context, Actor, string) (DatosParaDocumento, error)
}
