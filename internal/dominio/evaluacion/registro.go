package evaluacion

import (
	"errors"
	"time"
)

var ErrIdempotencia = errors.New("evaluacion.idempotencia_incompatible")

type Conflicto struct {
	VersionActual int `json:"version_actual"`
}

func (Conflicto) Error() string { return "evaluacion.conflicto" }

type InstrumentoAplicado struct {
	Sujeto       *Sujeto   `json:"sujeto,omitempty"`
	Momento      string    `json:"momento,omitempty"`
	ID           string    `json:"id"`
	ExpedienteID string    `json:"expediente_id"`
	EquipoID     string    `json:"equipo_id"`
	Codigo       string    `json:"codigo"`
	Fecha        time.Time `json:"fecha"`
	TextoOtros   string    `json:"texto_otros,omitempty"`
}
type Definiciones struct {
	Instrumentos []string `json:"codigos"`
	Escalas      []Escala `json:"escalas"`
}

type Registro struct {
	Definiciones *Definiciones         `json:"definiciones,omitempty"`
	Instrumentos []InstrumentoAplicado `json:"instrumentos"`
	Escalas      []Aplicacion          `json:"escalas"`
}
type Cambio struct {
	Aplicacion      *Aplicacion          `json:"aplicacion,omitempty"`
	Instrumento     *InstrumentoAplicado `json:"instrumento,omitempty"`
	VersionEsperada int                  `json:"version_esperada"`
	Clave           string               `json:"clave_idempotencia"`
}

// DatosDocumento proyecta únicamente las iniciales que identifican los resultados.
type DatosDocumento struct {
	Registro  Registro
	Iniciales map[string]string
}
