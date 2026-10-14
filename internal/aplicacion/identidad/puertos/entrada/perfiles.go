package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type CambioPerfiles = dominio.CambioPerfiles
type UsuarioAdministrado = dominio.UsuarioAdministrado

type AdministrarUsuarios interface {
	Listar(context.Context, Acceso) ([]UsuarioAdministrado, error)
	Cambiar(context.Context, Acceso, string, CambioPerfiles) (UsuarioAdministrado, error)
}

type claveAcceso struct{}

func ConAcceso(ctx context.Context, a Acceso) context.Context {
	return context.WithValue(ctx, claveAcceso{}, a)
}
func AccesoActual(ctx context.Context) Acceso { a, _ := ctx.Value(claveAcceso{}).(Acceso); return a }

func (a Acceso) Permite(perfil string) bool {
	return a.Estado == dominio.Activo && a.Usuario.Permite(dominio.Perfil(perfil))
}
