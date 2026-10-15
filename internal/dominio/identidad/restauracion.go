package identidad

import (
	"encoding/json"
	"time"
)

const (
	ErrSinRevision          Error = "restauracion.no_pendiente"
	ErrRevisionRestauracion Error = "restauracion.acceso_bloqueado"
	ErrCambiosPendientes    Error = "restauracion.cambios_pendientes"
	ErrEquipoRestauracion   Error = "restauracion.equipo_no_disponible"
	ErrRevisionPropia       Error = "restauracion.revision_propia"
)

type CambioRestauracion struct {
	ID        string          `json:"id"`
	Anterior  json.RawMessage `json:"anterior"`
	Propuesto json.RawMessage `json:"propuesto"`
	Decision  *string         `json:"decision"`
}
type RevisionRestauracion struct {
	ID      string               `json:"id"`
	CopiaEn time.Time            `json:"copia_en"`
	Version int64                `json:"version"`
	Cambios []CambioRestauracion `json:"cambios"`
	Equipos map[string]string    `json:"equipos"`
}
