package expediente

import (
	"context"
	estadistica "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

// LecturaSPAF aporta una foto coherente desde el módulo dueño de la ficha.
// Usa la transacción y la identidad RLS del contexto; nunca abre otra ni lee
// tablas sin RLS. Incluye historia, episodios confirmados y equivalencias de
// motivos, conocidos en el instante recibido, para un único equipo.
// El adaptador persistente conserva como desconocidos los campos de ficha
// que todavía no tienen una fuente histórica confirmada.
type LecturaSPAF interface {
	ConsultarSPAF(context.Context, estadistica.FiltroSPAF, time.Time, time.Time) ([]estadistica.ExpedienteSPAF, estadistica.Historia[estadistica.ReglasSPAF], error)
}
