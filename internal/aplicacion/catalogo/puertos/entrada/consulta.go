package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"time"
)

type Actor = dominio.Actor
type Version = dominio.Version
type Datos = dominio.Datos

var (
	ErrVigencia     = dominio.ErrVigencia
	ErrAmbito       = dominio.ErrAmbito
	ErrNoDisponible = dominio.ErrNoDisponible
)

// Actor procede exclusivamente de composición o de la futura sesión validada.
// El adaptador HTTP nunca lo obtiene de parámetros ni cabeceras del cliente.
type Catalogos interface {
	Vigente(context.Context, Actor, string, time.Time) (Version, error)
	Memoria(context.Context, Actor, string, int) (Version, error)
	Versiones(context.Context, Actor, string) ([]Version, error)
	Datos(context.Context, Actor, string, time.Time) (Datos, error)
}
