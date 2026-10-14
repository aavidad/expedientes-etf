package documento

import (
	"errors"
	"path"
	"regexp"
	"slices"
	"time"
)

const MaximoPlantilla = 10 << 20

var (
	ErrTamanoPlantilla  = errors.New("documento.plantilla_tamano")
	ErrMarcadores       = errors.New("documento.plantilla_marcadores")
	ErrAntivirus        = errors.New("documento.plantilla_antivirus")
	ErrInfectada        = errors.New("documento.plantilla_infectada")
	ErrVersionPlantilla = errors.New("documento.plantilla_version_usada")
	versionPlantilla    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

func (a Actor) AutorizarPlantillas() error {
	if !IDValido(a.UsuarioID) || !slices.Contains(a.Perfiles, "administracion") {
		return ErrAcceso
	}
	return nil
}

type PublicarPlantilla struct {
	Tipo          string `json:"tipo"`
	Idioma        string `json:"idioma"`
	Version       string `json:"version"`
	VersionPrevia int64  `json:"version_previa"`
}

func (p PublicarPlantilla) Validar() error {
	if !TipoPlantilla(p.Tipo) || len(p.Idioma) > 63 || !idioma.MatchString(p.Idioma) || len(p.Version) > 32 || !versionPlantilla.MatchString(p.Version) || p.VersionPrevia < 1 || p.VersionPrevia >= 9007199254740991 {
		return ErrPeticion
	}
	return nil
}

type ReferenciaPlantilla struct {
	Plantilla Plantilla `json:"plantilla"`
	Revision  int64     `json:"version_actual"`
}

type ConflictoPlantilla struct{ VersionActual int64 }

func (*ConflictoPlantilla) Error() string { return "documento.plantilla_conflicto" }

// La ruta exclusiva pertenece a esta publicación, también cuando cambia el
// paquete activo de idioma. Las versiones anteriores siguen siendo legibles.
func RutaPublicada(p Plantilla) string {
	return path.Join("plantillas", p.Tipo, ".versiones", p.ID, "v"+p.Version+".docx")
}
func PlantillaPublicada(p Plantilla) bool {
	return IDValido(p.ID) && TipoPlantilla(p.Tipo) && len(p.Idioma) <= 63 && idioma.MatchString(p.Idioma) && len(p.Version) <= 32 && versionPlantilla.MatchString(p.Version) && p.Ruta == RutaPublicada(p)
}

// Las versiones se conservan aunque cambie la referencia activa.
type VersionPlantilla struct {
	Plantilla    Plantilla             `json:"plantilla"`
	Activa       bool                  `json:"activa"`
	Revision     int64                 `json:"version_actual"`
	SubidaPor    string                `json:"subida_por"`
	SubidaEn     time.Time             `json:"subida_en"`
	Activaciones []ActivacionPlantilla `json:"activaciones"`
	Antivirus    string                `json:"antivirus"`
	Marcadores   string                `json:"marcadores"`
}
type ActivacionPlantilla struct {
	Por      string    `json:"por"`
	En       time.Time `json:"en"`
	Revision int64     `json:"revision"`
}
type ActivarPlantilla struct {
	ID            string `json:"id"`
	Idioma        string `json:"idioma"`
	VersionPrevia int64  `json:"version_previa"`
}

func (p ActivarPlantilla) Validar() error {
	if !IDValido(p.ID) || len(p.Idioma) > 63 || !idioma.MatchString(p.Idioma) || p.VersionPrevia < 1 || p.VersionPrevia >= 9007199254740991 {
		return ErrPeticion
	}
	return nil
}

type DetalleMarcadores struct {
	Faltan []string `json:"faltan"`
	Sobran []string `json:"sobran"`
}

func (*DetalleMarcadores) Error() string { return ErrMarcadores.Error() }
func (*DetalleMarcadores) Unwrap() error { return ErrMarcadores }
