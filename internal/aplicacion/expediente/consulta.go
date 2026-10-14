package expediente

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

// ConsultaAdultos es el contrato de lectura para evaluación. El consumidor
// abre su transacción con identidad de sesión y confirma la auditoría antes
// de entregar el resultado. El repositorio registra la lectura en esa transacción.
type ConsultaAdultos struct{ Repositorio salida.ConsultaAdultos }

func (c ConsultaAdultos) AdultosVigentes(ctx context.Context, id, corte string) ([]dominio.AdultoConsulta, error) {
	if c.Repositorio == nil {
		return nil, dominio.ErrNoDisponible
	}
	return c.Repositorio.AdultosVigentes(ctx, id, corte)
}
