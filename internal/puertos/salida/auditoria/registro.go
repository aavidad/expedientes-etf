package auditoria

import (
	"context"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

// Registro asigna el ID y encadena bajo exclusión mutua. Confirmar sólo
// devuelve éxito una vez confirmado el evento; un error no acredita registro.
// El adaptador persistente debe confirmar antes de retornar, sin depender de
// una transacción que vaya a confirmarse después de la entrega.
// Este puerto cubre lecturas/descargas; las escrituras de negocio necesitan
// compartir su propia transacción con auditoría (F0.08b).
type Registro interface {
	Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error)
}

// RegistroTransaccional comparte el contexto de la operación con su apunte.
// Confirmar registra fallos después de revertir la transacción de negocio.
type RegistroTransaccional interface {
	Registro
	RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error)
}
