package expediente

import "errors"

var (
	ErrHojaFormato   = errors.New("expediente.hoja_formato")
	ErrHojaLimites   = errors.New("expediente.hoja_limites")
	ErrHojaAntivirus = errors.New("expediente.hoja_antivirus")
	ErrHojaCambiada  = errors.New("expediente.hoja_cambiada")
)

// FuenteHoja contiene sólo comprobaciones técnicas, nunca el fichero ni su nombre.
type FuenteHoja struct {
	Formato string `json:"formato"`
	Huella  string `json:"huella"`
	Bytes   int    `json:"bytes"`
}

type IncidenciaHoja struct {
	Fila    int    `json:"fila"`
	Columna string `json:"columna"`
	Clave   string `json:"clave"`
	Mensaje string `json:"mensaje,omitempty"`
}

type FilaHoja struct {
	Numero      int
	Caso        CasoHistorico
	Incidencias []IncidenciaHoja
}

// ResumenFilaHoja permite revisar el lote sin exponer los nombres completos.
type ResumenFilaHoja struct {
	Fila      int    `json:"fila"`
	Numero    string `json:"numero"`
	Estado    string `json:"estado"`
	Iniciales string `json:"iniciales"`
}

type InformeHoja struct {
	Filas       []ResumenFilaHoja `json:"filas"`
	Valida      bool              `json:"valida"`
	LoteID      string            `json:"lote_id,omitempty"`
	Huella      string            `json:"huella,omitempty"`
	Casos       int               `json:"casos"`
	Incidencias []IncidenciaHoja  `json:"incidencias"`
}

// ErrorFilaHistorica identifica la posición del caso sin añadir sus valores.
type ErrorFilaHistorica struct {
	Indice int
	Causa  error
}

func (e ErrorFilaHistorica) Error() string { return e.Causa.Error() }
func (e ErrorFilaHistorica) Unwrap() error { return e.Causa }
