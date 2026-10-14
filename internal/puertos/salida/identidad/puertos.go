package identidad

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"time"
)

type Directorio interface {
	Autenticar(context.Context, string, string) (dominio.IdentidadDirectorio, error)
}

type Usuarios interface {
	// RegistrarAcceso busca exclusivamente por GUID, crea pendiente si no existe
	// y actualiza cuenta/nombre sin conceder perfiles ni reactivar bajas.
	// Una cuenta reutilizada no puede heredar el usuario de otro GUID.
	RegistrarAcceso(context.Context, dominio.IdentidadDirectorio, time.Time) (dominio.Usuario, error)
	Obtener(context.Context, string) (dominio.Usuario, error)
}

type Sesiones interface {
	// Crear falla si la huella ya existe; nunca sobrescribe una sesión.
	Crear(context.Context, dominio.Sesion) error
	// Modificar serializa la lectura y actualización con otras modificaciones
	// y Eliminaciones. El callback recibe el contexto de la misma transacción,
	// que debe usar al consultar al usuario. Si la función falla, no guarda; si no existe, devuelve
	// ErrSesionNoValida. La función sólo se ejecuta una vez, con un valor aislado.
	Modificar(context.Context, dominio.Huella, func(context.Context, *dominio.Sesion) error) error
	// Eliminar es idempotente y no permite que una modificación concurrente
	// vuelva a insertar la sesión después del cierre.
	Eliminar(context.Context, dominio.Huella) error
}

type Reloj interface{ Ahora() time.Time }

// Aleatoriedad debe llenar todos los bytes con un generador criptográfico.
// Se inyecta desde la composición, junto con el reloj y los repositorios.
type Aleatoriedad interface{ Llenar([]byte) error }

// Operaciones confirma el alta o actualización de usuario junto con su sesión.
// El callback recibe el contexto de una única transacción del módulo.
type Operaciones interface {
	EjecutarIdentidad(context.Context, func(context.Context) error) error
	// RegistrarFallo confirma un intento rechazado en una transacción nueva.
	RegistrarFallo(context.Context, string, string) error
}
