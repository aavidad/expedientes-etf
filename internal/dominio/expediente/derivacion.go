package expediente

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/expediente/ubicacion"
)

const MaxBytes = 10_000_000
const DerivacionRecibida = "derivacion_recibida"

var (
	ErrAcceso       = errors.New("expediente.acceso_denegado")
	ErrTamano       = errors.New("expediente.anexo_tamano")
	ErrFormato      = errors.New("expediente.anexo_formato")
	ErrInfectado    = errors.New("expediente.anexo_infectado")
	ErrNoDisponible = errors.New("expediente.no_disponible")
	ErrNoEncontrada = errors.New("expediente.derivacion_no_encontrada")
	ErrPeticion     = errors.New("expediente.peticion_invalida")
)

type Actor struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

func (a Actor) Autorizar() error {
	if !IDValido(a.UsuarioID) || !slices.Contains(a.Perfiles, "coordinacion") {
		return ErrAcceso
	}
	return nil
}

func IDValido(id string) bool {
	return id != "00000000-0000-0000-0000-000000000000" && regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id)
}

type Campo struct {
	Codigo  string   `json:"codigo"`
	Valor   string   `json:"valor"`
	Pagina  int      `json:"pagina"`
	Dudoso  bool     `json:"dudoso"`
	Motivos []string `json:"motivos"`
}

type Extraccion struct {
	Campos              []Campo  `json:"campos"`
	Avisos              []string `json:"avisos"`
	Tipo                string   `json:"tipo"`
	Version             string   `json:"version"`
	PlantillaReconocida bool     `json:"plantilla_reconocida"`
}

type LecturaAnexo struct {
	ID                 string    `json:"id"`
	Formato            string    `json:"formato"`
	TamanoBytes        int       `json:"tamano_bytes"`
	HuellaSHA256       string    `json:"huella_sha256"`
	Antivirus          string    `json:"antivirus"`
	AntivirusDetalle   string    `json:"antivirus_detalle,omitempty"`
	PlantillaDetectada string    `json:"plantilla_detectada"`
	CamposDudosos      []string  `json:"campos_dudosos"`
	LeidaPor           string    `json:"leida_por"`
	LeidaEn            time.Time `json:"leida_en"`
}

type DatosDerivacion struct {
	UbicacionConfirmada *UbicacionRevisada  `json:"ubicacion_confirmada,omitempty"`
	OpcionesUbicacion   OpcionesUbicacion   `json:"opciones_ubicacion"`
	EsquemaVersion      string              `json:"esquema_version"`
	Extraccion          Extraccion          `json:"contenido"`
	PropuestaMunicipio  ubicacion.Propuesta `json:"propuesta_municipio"`
}

type Derivacion struct {
	FechaDerivacion     string           `json:"fecha_derivacion,omitempty"`
	CargadoParaRevision bool             `json:"cargado_para_revision"`
	Traslado            *EnlaceTraslado  `json:"traslado,omitempty"`
	Traslados           []EnlaceTraslado `json:"traslados,omitempty"`
	ServicioDerivante   string           `json:"servicio_derivante,omitempty"`
	Seguimiento         *Seguimiento     `json:"seguimiento,omitempty"`
	NoAceptacion        *NoAceptacion    `json:"no_aceptacion,omitempty"`
	FechaEspera         *string          `json:"fecha_lista_espera,omitempty"`
	Aceptacion          *DatosAceptados  `json:"aceptacion,omitempty"`
	ID                  string           `json:"id"`
	Estado              string           `json:"estado"`
	Numero              *string          `json:"numero"`
	EquipoID            *string          `json:"equipo_id"`
	MunicipioID         *string          `json:"municipio_id"`
	Version             int              `json:"version"`
	Lectura             LecturaAnexo     `json:"lectura_anexo"`
	Datos               DatosDerivacion  `json:"derivacion_datos"`
}

// DatosPermitidos excluye identificadores documentales y escalas pertenecientes
// a evaluación. La propuesta nunca asigna municipio, centro, equipo ni número.
func DatosPermitidos(e Extraccion) Extraccion {
	e.Avisos = slices.DeleteFunc(e.Avisos, func(aviso string) bool { return aviso == "expediente.valorame_pendiente" })
	campos := make([]Campo, 0, len(e.Campos))
	dni := regexp.MustCompile(`(?i)\b(?:[0-9]{8}|[XYZ][0-9]{7})[A-Z]\b`)
	for _, c := range e.Campos {
		if strings.HasPrefix(c.Codigo, "valorame.") {
			continue
		}
		if strings.Contains(strings.ToLower(c.Codigo), "dni") {
			continue
		}
		if v := dni.ReplaceAllString(c.Valor, ""); v != c.Valor {
			c.Valor, c.Dudoso = v, true
			c.Motivos = append(c.Motivos, "expediente.identificador_omitido")
		}
		campos = append(campos, c)
	}
	e.Campos = campos
	return e
}

func ValidarExtraccion(e Extraccion) error {
	menores := 0
	for _, c := range e.Campos {
		if c.Codigo == "menores.persona.nombre" {
			menores++
		}
	}
	if menores > 15 {
		return errors.New("expediente.anexo_demasiados_menores")
	}
	return nil
}

func TextosUbicacion(e Extraccion) (direccion, valoracion string) {
	for _, c := range e.Campos {
		if c.Codigo == "familia.direccion" {
			direccion += " " + c.Valor
		}
		if strings.HasPrefix(c.Codigo, "valoracion.") {
			valoracion += " " + c.Valor
		}
	}
	return strings.TrimSpace(direccion), strings.TrimSpace(valoracion)
}
