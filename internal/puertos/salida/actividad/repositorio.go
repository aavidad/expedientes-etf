package actividad

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
)

type Repositorio interface {
	Consultar(context.Context, dominio.Tipo, dominio.Filtro) ([]dominio.Registro, error)
	Registrar(context.Context, dominio.Tipo, string, dominio.Cambio) (dominio.Registro, error)
	Corregir(context.Context, dominio.Tipo, string, dominio.Cambio) (dominio.Registro, error)
	Retirar(context.Context, dominio.Tipo, string, int) (dominio.Registro, error)
	Organismos(context.Context) ([]dominio.Organismo, error)
	Recontar(context.Context, dominio.Filtro) (dominio.Recuento, error)
}

// Equipos es el puerto de lectura de etiquetas del catálogo.
type Equipos interface {
	ConsultarEquipos(context.Context) ([]dominio.Equipo, error)
}
