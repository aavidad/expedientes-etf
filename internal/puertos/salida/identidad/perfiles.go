package identidad

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

// Perfiles opera sólo dentro de la transacción y comprueba de nuevo el actor.
type Perfiles interface {
	Listar(context.Context) ([]dominio.UsuarioAdministrado, error)
	Cambiar(context.Context, string, dominio.CambioPerfiles) (dominio.UsuarioAdministrado, error)
}

type claveActorAuditoria struct{}

// ConActorAuditoria transporta la identidad comprobada por el caso de uso.
func ConActorAuditoria(ctx context.Context, u dominio.Usuario) context.Context {
	return context.WithValue(ctx, claveActorAuditoria{}, u)
}
func ActorAuditoria(ctx context.Context) dominio.Usuario {
	u, _ := ctx.Value(claveActorAuditoria{}).(dominio.Usuario)
	return u
}
