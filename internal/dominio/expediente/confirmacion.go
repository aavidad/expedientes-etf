package expediente

import (
	"strings"
	"unicode/utf8"
)

// UbicacionRevisada conserva la elección de Coordinación antes de aceptar.
// No asigna el equipo definitivo ni consume numeración.
type UbicacionRevisada struct {
	MunicipioID string `json:"municipio_id"`
	CentroID    string `json:"centro_id"`
	EquipoID    string `json:"equipo_id"`
}
type OpcionUbicacion struct {
	ID          string  `json:"id"`
	ClaveTexto  string  `json:"clave_texto"`
	CentroID    *string `json:"centro_id,omitempty"`
	EquipoID    *string `json:"equipo_id,omitempty"`
	FueraAmbito bool    `json:"fuera_ambito,omitempty"`
}
type OpcionesUbicacion struct {
	Municipios []OpcionUbicacion `json:"municipios"`
	Centros    []OpcionUbicacion `json:"centros"`
	Equipos    []OpcionUbicacion `json:"equipos"`
}
type Confirmacion struct {
	LecturaID string            `json:"lectura_id"`
	Campos    []Campo           `json:"campos"`
	Tipo      string            `json:"tipo"`
	Ubicacion UbicacionRevisada `json:"ubicacion"`
}
type ReciboDerivacion struct {
	ID      string `json:"id"`
	Estado  string `json:"estado"`
	Version int    `json:"version"`
}

func (p Confirmacion) Validar(original Extraccion) error {
	if !IDValido(p.LecturaID) || (p.Tipo != "anexo3" && p.Tipo != "anexo4") || (original.Tipo != "" && original.Tipo != p.Tipo) || len(p.Campos) == 0 || len(p.Campos) > 500 {
		return ErrPeticion
	}
	if !IDValido(p.Ubicacion.MunicipioID) || !IDValido(p.Ubicacion.CentroID) || !IDValido(p.Ubicacion.EquipoID) {
		return ErrObligatorios
	}
	permitidos := make(map[string]bool)
	for _, c := range original.Campos {
		permitidos[c.Codigo] = true
	}
	for _, codigo := range CamposManuales {
		permitidos[codigo] = true
	}
	total := 0
	for _, c := range p.Campos {
		total += len(c.Valor)
		if !permitidos[c.Codigo] || len(c.Valor) > 50000 || !utf8.ValidString(c.Valor) || strings.ContainsRune(c.Valor, '\x00') || c.Pagina < 0 || c.Pagina > 20 {
			return ErrPeticion
		}
	}
	if total > 500000 {
		return ErrPeticion
	}
	return ValidarExtraccion(Extraccion{Campos: p.Campos})
}

// CamposManuales permite completar anexos escaneados sin admitir claves ajenas.
var CamposManuales = []string{
	"servicio.nombre", "servicio.coordinador", "servicio.telefono", "servicio.email",
	"familia.direccion", "familia.telefono", "familia.nombre",
	"menores.persona.nombre", "menores.persona.nacimiento", "menores.persona.escolarizacion",
	"sociodemografia.adulto.nombre", "sociodemografia.adulto.papel", "sociodemografia.adulto.edad", "sociodemografia.adulto.estudios", "sociodemografia.adulto.trabajo",
	"valoracion.diagnostico", "intervenciones.previas", "observaciones.texto",
}
