package documento

import (
	"regexp"
	"time"
)

const MemoriaWord = "memoria"

type SolicitudMemoria struct {
	Anio     int    `json:"anio"`
	EquipoID string `json:"equipo_id"`
	Idioma   string `json:"idioma"`
	Clave    string `json:"-"`
}

func (s SolicitudMemoria) Validar() error {
	if s.Anio < 1 || s.Anio > 9998 || (s.EquipoID != "" && !IDValido(s.EquipoID)) || len(s.Idioma) > 63 || !idioma.MatchString(s.Idioma) || !regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`).MatchString(s.Clave) {
		return ErrPeticion
	}
	return nil
}

type EmisionMemoria struct {
	ID              string           `json:"id"`
	Solicitud       SolicitudMemoria `json:"solicitud"`
	Plantilla       Plantilla        `json:"plantilla"`
	IdiomaPaquete   IdiomaPlantilla  `json:"idioma_paquete"`
	GeneradaEn      time.Time        `json:"generada_en"`
	GeneradaPor     string           `json:"generada_por"`
	HuellaPeticion  string           `json:"-"`
	HuellaDatos     string           `json:"huella_datos"`
	HuellaResultado string           `json:"huella_resultado"`
	VersionMotor    string           `json:"version_motor"`
}
type ResultadoMemoria struct {
	Emision   EmisionMemoria
	Contenido []byte
	Repetida  bool
}

// CampoMemoria vincula una celda de la plantilla a una dimensión estadística.
// Los selectores y los textos fijos pertenecen al paquete de idioma.
type CampoMemoria struct {
	Campo      string            `json:"campo"`
	Apartado   string            `json:"apartado"`
	Unidad     string            `json:"unidad"`
	Filtros    map[string]string `json:"filtros"`
	Porcentaje bool              `json:"porcentaje"`
	Total      bool              `json:"total"`
}
type PresentacionMemoria struct {
	Campos map[string]string `json:"campos"`
	Celdas []CampoMemoria    `json:"celdas"`
}
