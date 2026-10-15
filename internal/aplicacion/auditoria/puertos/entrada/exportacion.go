package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"io"
)

type ActorExportacion = dominio.ActorExportacion
type FiltroExportacion = dominio.FiltroExportacion
type DescargaAuditoria struct {
	Contenido io.ReadCloser
	Tamano    int64
	Filas     int
	Final     dominio.Punto
}
type Exportador interface {
	Rechazar(context.Context, ActorExportacion) error
	Exportar(context.Context, ActorExportacion, FiltroExportacion, string, string) (DescargaAuditoria, error)
}
