package actividad

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"time"
)

// Equipos consulta el puerto del catálogo dentro de la transacción de actividad.
type Equipos struct {
	Catalogo catalogo.Repositorio
	Ahora    func() time.Time
}

func (e Equipos) ConsultarEquipos(ctx context.Context) ([]dominio.Equipo, error) {
	version, err := e.Catalogo.Vigente(ctx, "asignacion_municipios", e.Ahora())
	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	datos, err := e.Catalogo.Datos(ctx, version)
	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	filas := make([]dominio.Equipo, 0, len(datos.Equipos))
	for _, fila := range datos.Equipos {
		filas = append(filas, dominio.Equipo{ID: fila.ID, ClaveTexto: fila.ClaveTexto})
	}
	return filas, nil
}
