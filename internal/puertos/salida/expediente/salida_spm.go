package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type SalidasSPM interface {
	LeerSPM(context.Context, string, bool) (*d.SalidaSPM, error)
	GuardarSPM(context.Context, string, d.SalidaSPM, d.PeticionSPM, string, time.Time) error
	RepetirSPM(context.Context, string, string, string, string, time.Time) (d.SalidaSPM, bool, error)
	RecordarSPM(context.Context, string, string, string, string, d.SalidaSPM, time.Time) error
}
