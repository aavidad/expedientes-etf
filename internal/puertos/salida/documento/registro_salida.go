package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"time"
)

// RegistroSalida desacopla la anotación manual del futuro conector MOAD.
// La aplicación confirma el resultado junto con la auditoría.
type RegistroSalida interface {
	Anotar(context.Context, d.RegistroSalida, d.SolicitudRegistroSalida, d.Actor, time.Time) (d.RegistroSalida, error)
}
type RepositorioRegistroSalida interface {
	PeticionRegistro(context.Context, d.Actor, d.SolicitudRegistroSalida, string) (d.RegistroSalida, bool, error)
	RecordarRegistro(context.Context, d.Actor, d.SolicitudRegistroSalida, string, d.RegistroSalida) error
	LeerRegistro(context.Context, string) (d.Emision, d.RegistroSalida, error)
	GuardarRegistro(context.Context, d.RegistroSalida, int) error
}
