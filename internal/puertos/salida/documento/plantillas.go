package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type PublicacionPlantillas interface {
	Actual(context.Context, string, string) (d.ReferenciaPlantilla, error)
	Publicar(context.Context, d.Plantilla, int64) (d.ReferenciaPlantilla, error)
	Registrada(context.Context, d.Plantilla) (bool, error)
}
type AlmacenPlantillas interface {
	Guardar(context.Context, d.Plantilla, []byte) error
	Eliminar(context.Context, d.Plantilla) error
}
type ValidadorPlantillas interface {
	Validar(context.Context, []byte, []byte) error
}

type GestionPlantillas interface {
	Listar(context.Context) ([]d.VersionPlantilla, error)
	Version(context.Context, string) (d.Plantilla, error)
	GuardarVersion(context.Context, d.Plantilla, int64) (d.ReferenciaPlantilla, error)
	Activar(context.Context, d.Plantilla, int64) (d.ReferenciaPlantilla, error)
}
