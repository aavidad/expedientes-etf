package entrada

import (
	"context"
	"time"
)

type EmisionDecision struct {
	ID               string          `json:"id"`
	Tipo             string          `json:"tipo"`
	GeneradaEn       time.Time       `json:"generada_en"`
	PlantillaVersion string          `json:"plantilla_version"`
	Registro         *RegistroSalida `json:"registro,omitempty"`
}
type ConsultaEmisionesDecision interface {
	ConsultarEmisionesDecision(context.Context, Actor, string) ([]EmisionDecision, error)
}
