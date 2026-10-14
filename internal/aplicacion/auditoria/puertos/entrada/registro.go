package entrada

import (
	"context"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

// Registro permite a otros módulos confirmar su auditoría antes de iniciar
// la entrega, sin depender de la implementación ni del almacenamiento.
type Registro interface {
	RegistrarAntesDeEntregar(context.Context, auditoria.Datos, func(context.Context) error) (auditoria.Evento, error)
}
