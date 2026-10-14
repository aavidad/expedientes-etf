package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"time"
)

// ValidadorInstalacion comprueba el ZIP en memoria. No sustituye la revisión
// antivirus ni publica, registra o activa una referencia.
type ValidadorInstalacion interface {
	Validar(context.Context, string, []byte) (dominio.ContenidoInstalacion, error)
}

// Publicacion confirma registro, referencia y auditoría como una sola operación.
// Registrada permite resolver un fallo de confirmación sin borrar contenido usado.
type Publicacion interface {
	Activar(context.Context, dominio.Actor, dominio.VersionInstalada, int64) (dominio.Referencia, error)
	Referencias(context.Context, string) ([]dominio.Referencia, error)
	Registrada(context.Context, dominio.VersionInstalada) (bool, error)
}

type AlmacenVersiones interface {
	Guardar(context.Context, dominio.ContenidoInstalacion) (dominio.VersionInstalada, error)
	Eliminar(context.Context, dominio.VersionInstalada) error
	Huerfanas(context.Context, time.Time) ([]dominio.VersionInstalada, error)
}
