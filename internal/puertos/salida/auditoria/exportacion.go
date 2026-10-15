package auditoria

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"io"
)

type LecturaExportacion interface {
	Delimitar(context.Context, dominio.FiltroExportacion) (dominio.TramoExportacion, error)
	Pagina(context.Context, dominio.TramoExportacion, uint64) ([]dominio.Evento, error)
}

type ArchivoExportacion interface {
	io.ReadCloser
	Tamano() int64
	Filas() int
}

// Preparar escribe ficheros privados y devuelve un archivo ya comprobado.
// Cerrar elimina todos los temporales, también cuando se cancela la descarga.
type Empaquetador interface {
	Preparar(context.Context, dominio.TramoExportacion, dominio.FiltroExportacion, string, string, func(func(dominio.Evento) error) error) (ArchivoExportacion, error)
}
