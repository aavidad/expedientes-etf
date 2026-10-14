package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

type SMI = dominio.SMI
type CambioSMI = dominio.CambioSMI

var ErrSMI = dominio.ErrSMI
var ParsearImporteSMI = dominio.ParsearImporteSMI

type GestorSMI interface {
	Consultar(context.Context, Actor) ([]SMI, error)
	Guardar(context.Context, Actor, CambioSMI) (SMI, error)
}
