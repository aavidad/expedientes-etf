package evaluacion

import (
	"context"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
)

// Consulta permite a documento obtener las aplicaciones de un expediente sin
// depender de SQL ni HTTP. Conserva momento, sujeto y versión de cada escala.
// La implementación debe autorizar al actor y confirmar la auditoría antes de
// entregar datos; no permite consultar todas las evaluaciones sin expediente.
type Consulta interface {
	Consultar(context.Context, dominio.Actor, string) ([]dominio.Aplicacion, error)
}
