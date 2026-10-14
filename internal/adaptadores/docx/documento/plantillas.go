package documento

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/docx"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type ValidadorPlantillas struct{}

func (ValidadorPlantillas) Validar(ctx context.Context, nueva, actual []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := docx.ValidarCambio(nueva, actual); err != nil {
		var diferencia *docx.DiferenciaMarcadores
		if errors.As(err, &diferencia) {
			return &d.DetalleMarcadores{Faltan: diferencia.Faltan, Sobran: diferencia.Sobran}
		}
		return d.ErrMarcadores
	}
	return ctx.Err()
}
