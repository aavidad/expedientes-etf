package estadistica

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

// CifrasPublicadas comparte la transacción de lectura y su auditoría.
// Consultar accede exclusivamente a la proyección agregada para Consulta.
// Publicar guarda sólo cifras, nunca el InformeMemoria con sus narrativas.
type CifrasPublicadas interface {
	Consultar(context.Context, d.FiltroMemoria) (d.CifrasMemoria, error)
	Publicar(context.Context, d.CifrasMemoria) error
}
