package lectura

import (
	"context"
	"io"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/odt"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/pdf"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

// Lector limita a una lectura simultánea. PDF se ejecuta en un proceso efímero
// con entrada y salida por tuberías; nunca usa nombres ni ficheros temporales.
type Lector struct {
	plazas chan struct{}
	pdf    *pdf.Lector
}

var _ expediente.LectorAnexo = (*Lector)(nil)

func NuevoLector() *Lector { return &Lector{plazas: make(chan struct{}, 1), pdf: pdf.NuevoLector()} }

func (l *Lector) LeerActa(ctx context.Context, formato string, b []byte) (dominio.Extraccion, error) {
	select {
	case l.plazas <- struct{}{}:
		defer func() { <-l.plazas }()
	case <-ctx.Done():
		return dominio.Extraccion{}, ctx.Err()
	}
	if formato == "docx" {
		return docx.LeerActa(b)
	}
	return l.pdf.LeerActa(ctx, formato, b)
}

func (l *Lector) Leer(ctx context.Context, formato string, b []byte) (dominio.Extraccion, error) {
	if err := ctx.Err(); err != nil {
		return dominio.Extraccion{}, err
	}
	select {
	case l.plazas <- struct{}{}:
		defer func() { <-l.plazas }()
	case <-ctx.Done():
		return dominio.Extraccion{}, ctx.Err()
	}
	if formato == "docx" || formato == "odt" {
		var e docx.Lectura
		var err error
		if formato == "odt" {
			e, err = odt.Leer(b)
		} else {
			e, err = docx.Leer(b)
		}
		if err != nil {
			return dominio.Extraccion{}, err
		}
		r := dominio.Extraccion{PlantillaReconocida: e.PlantillaReconocida, Avisos: e.Avisos, Tipo: e.Tipo, Version: e.Version}
		for _, c := range e.Campos {
			r.Campos = append(r.Campos, dominio.Campo{Codigo: c.Codigo, Valor: c.Valor, Pagina: c.Pagina, Dudoso: c.Dudoso, Motivos: c.Motivos})
		}
		return r, nil
	}
	return l.pdf.Leer(ctx, formato, b)
}

// EjecutarPDF conserva el enganche del ejecutable y de las pruebas de F1.01.
func EjecutarPDF(entrada io.Reader, salida io.Writer) error { return pdf.EjecutarPDF(entrada, salida) }
