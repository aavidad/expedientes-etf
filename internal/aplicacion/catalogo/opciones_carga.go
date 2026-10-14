package catalogo

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
)

// ConsultarOpcionesCarga publica las etiquetas y limita el equipo al ámbito de sesión.
func (c ConsultaDecision) ConsultarOpcionesCarga(ctx context.Context, a entrada.Actor, idioma string) (entrada.OpcionesDecision, error) {
	return c.consultarOpciones(ctx, a, idioma, true)
}
