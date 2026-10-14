package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"time"
)

type FicherosCatalogo interface {
	LeerCatalogo(context.Context, dominio.ReferenciaCatalogo) (map[string]string, error)
	LimpiarCatalogo(context.Context, string, map[string]bool, time.Time) error
	EscribirCatalogo(context.Context, string, map[string]string) (dominio.ReferenciaCatalogo, error)
}
type Referencias interface {
	Bloquear(context.Context) error
	Conservadas(context.Context) (map[string]map[string]bool, error)
	Contenido(context.Context, string, string) (dominio.ReferenciaCatalogo, error)
	Consultar(context.Context, string) (dominio.ReferenciaCatalogo, error)
	Activar(context.Context, dominio.ReferenciaCatalogo, int) error
}
