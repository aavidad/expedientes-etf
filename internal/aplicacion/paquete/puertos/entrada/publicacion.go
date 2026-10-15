package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"time"
)

type Catalogos = dominio.Catalogos
type Preparada = dominio.Preparada
type ConflictoPublicacion = dominio.Conflicto
type PublicacionCatalogos interface {
	LimpiarHuerfanos(context.Context, time.Time) error
	TextosVersion(context.Context, string, string) (map[string]string, error)
	Catalogos(context.Context) (Catalogos, error)
	Preparar(context.Context, string, map[string]string, map[string]int) (Preparada, error)
	Activar(context.Context, Preparada) error
}

// PublicacionUbicaciones prepara denominación y variantes en una instantánea.
type PublicacionUbicaciones interface {
	PrepararUbicacion(context.Context, string, map[string]string, map[string][]string, map[string]int) (Preparada, error)
}
