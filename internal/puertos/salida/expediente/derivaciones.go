package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Repositorio interface {
	Guardar(context.Context, dominio.Derivacion) error
	Obtener(context.Context, string) (dominio.Derivacion, error)
}

type ValidadorFichero interface {
	Validar(context.Context, string, []byte) (string, error)
}

type LectorAnexo interface {
	Leer(context.Context, string, []byte) (dominio.Extraccion, error)
}

type NombresMunicipios interface {
	Nombres(context.Context) (map[string]string, error)
}

// RegistroTransaccional usa la transacción del caso de uso, sin abrir ni
// confirmar otra. Registro.Confirmar se reserva para rechazos sin escritura.
type RegistroTransaccional interface {
	RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error)
}

type Reloj interface{ Ahora() time.Time }
type Identificadores interface{ Nuevo() (string, error) }
