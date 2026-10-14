package documento

import (
	"context"
	"time"
)

type EmisionEquipo struct {
	ID               string    `json:"id"`
	Tipo             string    `json:"tipo"`
	Fecha            time.Time `json:"fecha"`
	Autor            string    `json:"autor"`
	VersionPlantilla string    `json:"version_plantilla"`
}

type ConsultaEquipo interface {
	EmisionesEquipo(context.Context, string) ([]EmisionEquipo, error)
}
