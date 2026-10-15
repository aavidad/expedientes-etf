package catalogo

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

// ReglasEstadisticas conserva versiones publicadas conocidas al calcular.
type ReglasEstadisticas interface {
	ReglasSPAF(context.Context, time.Time, time.Time) (d.Historia[d.ReglasSPAF], error)
}

type EquiposEstadistica interface {
	CodigoEquipo(context.Context, string) (string, error)
}

type EquiposMemoria interface {
	EquiposMemoria(context.Context) ([]string, error)
}

// MunicipiosEstadistica resuelve identificadores a códigos de catálogo.
type MunicipiosEstadistica interface {
	CodigosMunicipios(context.Context, []string) (map[string]string, error)
}
