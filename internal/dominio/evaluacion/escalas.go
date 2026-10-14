package evaluacion

import (
	"errors"
	"math"
	"regexp"
	"slices"
	"time"
)

var (
	ErrPeticion      = errors.New("evaluacion.peticion_invalida")
	ErrAcceso        = errors.New("evaluacion.acceso_denegado")
	ErrSujeto        = errors.New("evaluacion.sujeto_invalido")
	ErrPuntuacion    = errors.New("evaluacion.puntuacion_invalida")
	ErrVersionEscala = errors.New("evaluacion.version_escala_invalida")
	ErrNoEncontrada  = errors.New("evaluacion.no_encontrada")
	ErrNoDisponible  = errors.New("evaluacion.no_disponible")
)

type Actor struct {
	UsuarioID string
	EquipoID  string
	Perfiles  []string
}

func (a Actor) Autorizar(escritura bool) error {
	if !IDValido(a.UsuarioID) {
		return ErrAcceso
	}
	if slices.Contains(a.Perfiles, "coordinacion") {
		return nil
	}
	if !IDValido(a.EquipoID) || !slices.Contains(a.Perfiles, "profesional_etf") {
		return ErrAcceso
	}
	return nil
}

func IDValido(id string) bool {
	return regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id) && id != "00000000-0000-0000-0000-000000000000"
}

// Item contiene las reglas de una versión publicada. No fija baremos en código.
// Paso permite distinguir enteros, medias y puntuaciones transformadas.
type Item struct {
	// CampoDocumento identifica una fila de los Anexos 11/12; nunca calcula baremos.
	CampoDocumento string   `json:"campo_documento,omitempty"`
	Codigo         string   `json:"codigo"`
	Tipo           string   `json:"tipo"`
	Minimo         float64  `json:"minimo"`
	Maximo         float64  `json:"maximo"`
	Paso           float64  `json:"paso"`
	Valores        []string `json:"valores,omitempty"`
}

type Escala struct {
	ID          string `json:"id"`
	Instrumento string `json:"instrumento"`
	Version     int    `json:"version"`
	Items       []Item `json:"items"`
}

func (e Escala) Validar() error {
	if !IDValido(e.ID) || e.Instrumento == "" || e.Version < 1 || len(e.Items) == 0 {
		return ErrVersionEscala
	}
	items := make(map[string]bool, len(e.Items))
	campos := map[string]bool{}
	for _, i := range e.Items {
		if i.Codigo == "" || items[i.Codigo] {
			return ErrVersionEscala
		}
		items[i.Codigo] = true
		if i.CampoDocumento != "" {
			if !regexp.MustCompile(`^escala_[a-z0-9_]+_[1-9][0-9]*$`).MatchString(i.CampoDocumento) || campos[i.CampoDocumento] {
				return ErrVersionEscala
			}
			campos[i.CampoDocumento] = true
		}
		switch i.Tipo {
		case "numerico":
			if !finito(i.Minimo) || !finito(i.Maximo) || !finito(i.Paso) || i.Minimo > i.Maximo || i.Paso <= 0 || !finito((i.Maximo-i.Minimo)/i.Paso) || len(i.Valores) != 0 {
				return ErrVersionEscala
			}
		case "codigo":
			if len(i.Valores) == 0 || i.Minimo != 0 || i.Maximo != 0 || i.Paso != 0 {
				return ErrVersionEscala
			}
			vistos := map[string]bool{}
			for _, v := range i.Valores {
				if v == "" || vistos[v] {
					return ErrVersionEscala
				}
				vistos[v] = true
			}
		default:
			return ErrVersionEscala
		}
	}
	return nil
}

type Sujeto struct {
	Tipo          string `json:"tipo"`
	AdultoID      string `json:"adulto_id,omitempty"`
	MenorID       string `json:"menor_id,omitempty"`
	ProfesionalID string `json:"profesional_id,omitempty"`
}

// Validar comprueba la forma. La pertenencia se comprueba mediante claves
// compuestas en persistencia y la adscripción profesional en la fecha aplicada.
func (s Sujeto) Validar() error {
	switch s.Tipo {
	case "familia":
		if s.AdultoID == "" && s.MenorID == "" && s.ProfesionalID == "" {
			return nil
		}
	case "figura_parental":
		if IDValido(s.AdultoID) && s.MenorID == "" && s.ProfesionalID == "" {
			return nil
		}
	case "menor":
		if IDValido(s.MenorID) && s.AdultoID == "" && s.ProfesionalID == "" {
			return nil
		}
	case "profesional":
		if IDValido(s.ProfesionalID) && s.AdultoID == "" && s.MenorID == "" {
			return nil
		}
	}
	return ErrSujeto
}

type Puntuacion struct {
	Item          string   `json:"item"`
	ValorNumerico *float64 `json:"valor_numerico,omitempty"`
	ValorCodigo   *string  `json:"valor_codigo,omitempty"`
}

type Aplicacion struct {
	ID                     string       `json:"id"`
	ExpedienteID           string       `json:"expediente_id"`
	EquipoID               string       `json:"equipo_id"`
	Instrumento            string       `json:"instrumento"`
	Momento                string       `json:"momento"`
	Sujeto                 Sujeto       `json:"sujeto"`
	Fecha                  *time.Time   `json:"fecha,omitempty"`
	VersionEscalaID        string       `json:"version_escala_id"`
	DescripcionCualitativa string       `json:"descripcion_cualitativa"`
	Version                int          `json:"version"`
	Puntuaciones           []Puntuacion `json:"puntuaciones"`
}

func (a Aplicacion) Validar(e Escala) error {
	if !IDValido(a.ID) || !IDValido(a.ExpedienteID) || !IDValido(a.EquipoID) || a.Version < 1 || !slices.Contains([]string{"derivacion", "inicial", "intermedia", "final"}, a.Momento) {
		return ErrPeticion
	}
	if err := a.Sujeto.Validar(); err != nil {
		return err
	}
	if err := e.Validar(); err != nil {
		return err
	}
	if a.VersionEscalaID != e.ID || a.Instrumento != e.Instrumento {
		return ErrVersionEscala
	}
	items := make(map[string]Item, len(e.Items))
	for _, i := range e.Items {
		items[i.Codigo] = i
	}
	vistos := map[string]bool{}
	for _, p := range a.Puntuaciones {
		i, existe := items[p.Item]
		if !existe || vistos[p.Item] {
			return ErrPuntuacion
		}
		vistos[p.Item] = true
		if i.Tipo == "codigo" {
			if p.ValorNumerico != nil || p.ValorCodigo == nil || !slices.Contains(i.Valores, *p.ValorCodigo) {
				return ErrPuntuacion
			}
			continue
		}
		if p.ValorNumerico == nil || p.ValorCodigo != nil {
			return ErrPuntuacion
		}
		v := *p.ValorNumerico
		if !finito(v) || v < i.Minimo || v > i.Maximo {
			return ErrPuntuacion
		}
		tramos := (v - i.Minimo) / i.Paso
		if !finito(tramos) || math.Abs(tramos-math.Round(tramos)) > 1e-9 {
			return ErrPuntuacion
		}
	}
	return nil
}

func finito(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
