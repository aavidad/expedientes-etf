package documento

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrRegistroOficio = errors.New("documento.registro_solo_oficios")

type ConflictoRegistro struct {
	VersionActual int `json:"version_actual"`
}

func (*ConflictoRegistro) Error() string { return "error.conflicto_version" }

func (a Actor) AutorizarEntrada() error {
	if !IDValido(a.UsuarioID) || !slices.Contains(a.Perfiles, "coordinacion") {
		return ErrAcceso
	}
	return nil
}

type SolicitudRegistroSalida struct {
	EmisionID string `json:"-"`
	Clave     string `json:"-"`
	Version   int    `json:"version"`
	Numero    string `json:"numero_registro"`
	Fecha     string `json:"fecha_registro"`
}

func (s SolicitudRegistroSalida) Validar() error {
	fecha, err := time.Parse("2006-01-02", s.Fecha)
	if !IDValido(s.EmisionID) || s.Version < 1 || !clave.MatchString(s.Clave) || len(s.Numero) > 128 || strings.TrimSpace(s.Numero) != s.Numero || s.Numero == "" || !utf8.ValidString(s.Numero) || err != nil || fecha.Year() < 1 {
		return ErrPeticion
	}
	for _, r := range s.Numero {
		if r < 32 || r == 127 {
			return ErrPeticion
		}
	}
	return nil
}

type RegistroSalida struct {
	EmisionID         string    `json:"emision_id"`
	Adaptador         string    `json:"adaptador"`
	Estado            string    `json:"estado"`
	Numero            string    `json:"numero_registro,omitempty"`
	Fecha             string    `json:"fecha_registro,omitempty"`
	ReferenciaExterna string    `json:"referencia_externa,omitempty"`
	AnotadoPor        string    `json:"anotado_por,omitempty"`
	AnotadoEn         time.Time `json:"anotado_en,omitempty"`
	ErrorClave        string    `json:"error_clave,omitempty"`
	Version           int       `json:"version"`
}

func HuellaRegistro(s SolicitudRegistroSalida) (string, error) {
	return huellaJSON(struct {
		EmisionID     string
		Version       int
		Numero, Fecha string
	}{s.EmisionID, s.Version, s.Numero, s.Fecha})
}
