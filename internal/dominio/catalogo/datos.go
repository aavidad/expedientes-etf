package catalogo

import "time"

type Equipo struct {
	ID          string     `json:"id"`
	Codigo      string     `json:"codigo"`
	ClaveTexto  string     `json:"clave_texto"`
	ActivoDesde time.Time  `json:"activo_desde"`
	ActivoHasta *time.Time `json:"activo_hasta"`
}

type Centro = Equipo

type Municipio struct {
	ID               string  `json:"id"`
	INE              string  `json:"ine"`
	ClaveTexto       string  `json:"clave_texto"`
	Tipo             string  `json:"tipo"`
	MatrizID         *string `json:"matriz_id"`
	AmbitoDiputacion bool    `json:"ambito_diputacion"`
	CentroID         *string `json:"centro_ss_id"`
	EquipoID         *string `json:"equipo_id"`
	ZTS              *string `json:"zts"`
}

type Opcion struct {
	Lista       string `json:"lista"`
	Codigo      string `json:"codigo"`
	Orden       int    `json:"orden"`
	AdmiteTexto bool   `json:"admite_texto"`
	Activa      bool   `json:"activa"`
}

type Datos struct {
	DuracionesFase []DuracionFase   `json:"duraciones_fase"`
	Numeracion     *ReglaNumeracion `json:"numeracion,omitempty"`
	Plazos         []Plazo          `json:"plazos"`
	Tramos         []Tramo          `json:"tramos"`
	Version        Version          `json:"version"`
	Equipos        []Equipo         `json:"equipos"`
	Centros        []Centro         `json:"centros"`
	Municipios     []Municipio      `json:"municipios"`
	Opciones       []Opcion         `json:"opciones"`
}
