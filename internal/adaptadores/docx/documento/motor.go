package documento

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/docx"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type Motor struct{}

func (Motor) Version() string { return "docx-1.0.0" }
func (Motor) Generar(ctx context.Context, b []byte, r d.Relleno) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	contenido, err := docx.Generar(b, docx.Datos{Campos: r.Campos, Filas: r.Filas})
	if err != nil {
		return nil, d.ErrPlantilla
	}
	if err := ctx.Err(); err != nil {
		clear(contenido)
		return nil, err
	}
	return contenido, nil
}

// MotorSPAF amplía sólo el número de casos, manteniendo las cotas de memoria
// y de XML del motor común. Las plantillas de personas conservan quince filas.
type MotorSPAF struct{}

func (MotorSPAF) Version() string { return "docx-spaf-1.0.0" }
func (MotorSPAF) Generar(ctx context.Context, b []byte, r d.Relleno) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	contenido, err := docx.Generar(b, docx.Datos{Campos: r.Campos, Filas: r.Filas, LimiteFilas: 1000})
	if err != nil {
		return nil, d.ErrPlantilla
	}
	if err := ctx.Err(); err != nil {
		clear(contenido)
		return nil, err
	}
	return contenido, nil
}
