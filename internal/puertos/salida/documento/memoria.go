package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type RepositorioMemoria interface {
	BloquearMemoria(context.Context, string, string) error
	PeticionMemoria(context.Context, string, string) (d.EmisionMemoria, error)
	PlantillaMemoria(context.Context, string) (d.Plantilla, error)
	GuardarMemoria(context.Context, d.EmisionMemoria) error
}
type TextosMemoria interface {
	LeerMemoria(context.Context, d.IdiomaPlantilla, d.Plantilla) (d.PresentacionMemoria, error)
}
type GeneradorMemoria interface {
	CamposMemoria(context.Context, e.InformeMemoria, d.PresentacionMemoria) (map[string]string, error)
	Generar(context.Context, []byte, d.Relleno) ([]byte, error)
	Version() string
}
