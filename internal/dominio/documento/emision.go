package documento

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

const Anexo6 = "salida/anexo6"
const Anexo5 = "salida/spm/anexo5"

const ActaAceptacion = "entrada/aceptacion"
const OficioSSCC = "entrada/aceptacion/oficio_sscc"
const OficioSPM = "entrada/aceptacion/oficio_spm"
const ActaNoAceptacion = "entrada/no_aceptacion"
const OficioListaEspera = "entrada/no_aceptacion/lista_espera"

const ActaCierre = "cierre/acta_cierre"

const Anexo11 = "finalizacion/anexo11"
const Anexo12 = "finalizacion/anexo12"

func EsFinalizacion(tipo string) bool { return tipo == Anexo11 || tipo == Anexo12 }
func (a Actor) AutorizarTipo(tipo string) error {
	if tipo == Anexo6 || tipo == Anexo5 {
		return a.Autorizar()
	}
	if EsFinalizacion(tipo) {
		if IDValido(a.UsuarioID) && IDValido(a.EquipoID) && slices.Contains(a.Perfiles, "profesional_etf") {
			return nil
		}
		return ErrAcceso
	}
	return a.AutorizarEmision(tipo)
}

func TipoActivo(tipo string) bool {
	return tipo == ActaCierre || tipo == Anexo5 || tipo == Anexo6 || EsFinalizacion(tipo) || EsConsentimiento(tipo) || tipo == ActaAceptacion || tipo == OficioSSCC || tipo == OficioSPM || tipo == ActaNoAceptacion || tipo == OficioListaEspera
}

// TipoPlantilla permite leer plantillas estadísticas sin habilitarlas en el
// caso de uso de documentos de expediente.
func TipoPlantilla(tipo string) bool {
	return TipoActivo(tipo) || tipo == TablaSPAF || tipo == MemoriaWord
}
func EsOficio(tipo string) bool {
	return tipo == ActaCierre || tipo == OficioSSCC || tipo == OficioSPM || tipo == OficioListaEspera
}

var (
	ErrAcceso       = errors.New("documento.acceso_denegado")
	ErrPeticion     = errors.New("documento.peticion_invalida")
	ErrNoDisponible = errors.New("documento.no_disponible")
	ErrPlantilla    = errors.New("documento.plantilla_no_disponible")
	ErrHuella       = errors.New("documento.plantilla_alterada")
	ErrFirmante     = errors.New("documento.firmante_no_disponible")
	ErrIdempotencia = errors.New("error.idempotencia_reutilizada")
	ErrNoAceptada   = errors.New("documento.requiere_no_aceptada")
	ErrListaEspera  = errors.New("documento.requiere_lista_espera")
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var idioma = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
var clave = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var codigo = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)

func IDValido(id string) bool {
	return uuid.MatchString(id) && id != "00000000-0000-0000-0000-000000000000"
}

type Actor struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

func (a Actor) Autorizar() error {
	if IDValido(a.UsuarioID) && (slices.Contains(a.Perfiles, "coordinacion") || ((slices.Contains(a.Perfiles, "equipo") || slices.Contains(a.Perfiles, "profesional_etf")) && IDValido(a.EquipoID))) {
		return nil
	}
	return ErrAcceso
}
func (a Actor) AutorizarEquipo(equipo string) error {
	if a.Autorizar() != nil {
		return ErrAcceso
	}
	if slices.Contains(a.Perfiles, "coordinacion") || equipo != "" && equipo == a.EquipoID {
		return nil
	}
	return ErrAcceso
}

type Solicitud struct {
	FechaRepeticion   time.Time       `json:"-"`
	SalidaID          string          `json:"salida_id,omitempty"`
	Version           int             `json:"version,omitempty"`
	Tipo              string          `json:"tipo"`
	ExpedienteID      string          `json:"expediente_id"`
	Idioma            string          `json:"idioma"`
	Clave             string          `json:"-"`
	IdiomaPaquete     IdiomaPlantilla `json:"-"`
	FechaReproduccion time.Time       `json:"-"`
}

func (s Solicitud) Validar() error {
	if (EsFinalizacion(s.Tipo) || s.Tipo == ActaCierre) && s.Version < 1 {
		return ErrPeticion
	}
	if !TipoActivo(s.Tipo) || !IDValido(s.ExpedienteID) || len(s.Idioma) > 63 || !idioma.MatchString(s.Idioma) || !clave.MatchString(s.Clave) {
		return ErrPeticion
	}
	if s.Tipo == Anexo6 && !IDValido(s.SalidaID) || s.Tipo != Anexo6 && s.SalidaID != "" {
		return ErrPeticion
	}
	return nil
}

// Los valores de presentación llegan formateados por el proveedor del tipo.
// No hay fechas de hitos editables ni operaciones de escritura de expediente.
type Persona struct {
	Nombre     string `json:"nombre"`
	Nacimiento string `json:"nacimiento"`
}
type Adulto struct {
	Persona
	Parentesco string `json:"parentesco"`
}
type Menor struct {
	Persona
	Sexo       string `json:"sexo"`
	CodigoSexo string `json:"codigo_sexo"`
}
type Aceptacion struct {
	Finalizacion      map[string]string              `json:"finalizacion,omitempty"`
	Decision          *DecisionDocumento             `json:"decision,omitempty"`
	Direccion         string                         `json:"direccion"`
	FechaActa         string                         `json:"fecha_acta"`
	DestinatarioSSCC  string                         `json:"destinatario_sscc"`
	MarcaSeleccion    string                         `json:"marca_seleccion"`
	RotuloMenor       string                         `json:"rotulo_menor"`
	Selecciones       map[string]string              `json:"selecciones"`
	Numero            string                         `json:"numero"`
	Equipo            string                         `json:"equipo"`
	Centro            string                         `json:"centro"`
	Municipio         string                         `json:"municipio"`
	ServicioDerivante string                         `json:"servicio_derivante"`
	Descripcion       string                         `json:"descripcion"`
	Familia           string                         `json:"familia"`
	Madre             Persona                        `json:"madre"`
	Padre             Persona                        `json:"padre"`
	Adultos           []Adulto                       `json:"adultos"`
	Menores           []Menor                        `json:"menores"`
	Profesional       string                         `json:"profesional"`
	CamposSPM         map[string]string              `json:"campos_spm,omitempty"`
	FilasSPM          map[string][]map[string]string `json:"filas_spm,omitempty"`
}

type SalidaDocumento struct {
	Campos map[string]string
	Filas  map[string][]map[string]string
}

// DecisionDocumento aporta los datos confirmados de una entrada sin aceptación.
type DecisionDocumento struct {
	Fecha           string `json:"fecha"`
	Motivo          string `json:"motivo"`
	Subprograma     string `json:"subprograma"`
	FechaDerivacion string `json:"fecha_derivacion"`
}
type Datos struct {
	Salida *SalidaDocumento

	ExpedienteID   string
	EquipoID       string
	FechaDocumento time.Time
	Versiones      map[string]string
	Aceptacion     *Aceptacion
}

func (d Datos) Validar(s Solicitud) error {
	if d.ExpedienteID != s.ExpedienteID || (d.EquipoID != "" && !IDValido(d.EquipoID)) || d.FechaDocumento.IsZero() || d.FechaDocumento.Year() < 1 || d.FechaDocumento.Year() > 9999 || len(d.Versiones) == 0 || len(d.Versiones) > 32 || d.Aceptacion == nil || len(d.Aceptacion.Adultos) > 5 || len(d.Aceptacion.Menores) < 1 || len(d.Aceptacion.Menores) > 15 {
		return ErrPeticion
	}
	for k, v := range d.Versiones {
		if !codigo.MatchString(k) || len(v) == 0 || len(v) > 128 {
			return ErrPeticion
		}
	}
	if s.Tipo == Anexo6 && d.Salida == nil {
		return ErrPeticion
	}
	if (s.Tipo == ActaNoAceptacion || s.Tipo == OficioListaEspera) && d.Aceptacion.Decision == nil {
		return ErrPeticion
	}
	return nil
}

type Firmante struct {
	ID      string
	Version int
	Nombre  string
}
type Relleno struct {
	Campos map[string]string              `json:"campos"`
	Filas  map[string][]map[string]string `json:"filas"`
}

func (a Aceptacion) Rellenar(f Firmante) Relleno {
	r := Relleno{Campos: map[string]string{"numero": a.Numero, "equipo": a.Equipo, "centro": a.Centro, "municipio": a.Municipio, "servicio_derivante": a.ServicioDerivante, "descripcion": a.Descripcion, "familia": a.Familia, "madre": a.Madre.Nombre, "madre_nacimiento": a.Madre.Nacimiento, "padre": a.Padre.Nombre, "padre_nacimiento": a.Padre.Nacimiento, "profesional": a.Profesional, "firmante": f.Nombre, "fecha_acta": a.FechaActa, "direccion": a.Direccion, "destinatario_sscc": a.DestinatarioSSCC}, Filas: map[string][]map[string]string{"menores": {}}}
	r.Filas["adultos"] = []map[string]string{}
	r.Filas["lista_menores"] = []map[string]string{}
	for k, v := range a.Finalizacion {
		r.Campos[k] = v
	}
	for k, v := range a.Selecciones {
		r.Campos[k] = v
	}
	if a.Decision != nil {
		r.Campos["motivo"] = a.Decision.Motivo
		r.Campos["subprograma"] = a.Decision.Subprograma
		r.Campos["fecha_decision"] = a.Decision.Fecha
		r.Campos["fecha_derivacion"] = a.Decision.FechaDerivacion
		nombres := make([]string, 0, len(a.Menores))
		for _, menor := range a.Menores {
			nombres = append(nombres, menor.Nombre)
		}
		r.Campos["menores_lista"] = strings.Join(nombres, "; ")
	}
	for i := 0; i < 5; i++ {
		adulto := Adulto{}
		if i < len(a.Adultos) {
			adulto = a.Adultos[i]
		}
		k := fmt.Sprintf("adulto%d", i+1)
		r.Campos[k] = adulto.Nombre
		r.Campos[k+"_parentesco"] = adulto.Parentesco
		r.Campos[k+"_nacimiento"] = adulto.Nacimiento
		r.Filas["adultos"] = append(r.Filas["adultos"], map[string]string{"orden": fmt.Sprint(i + 1), "nombre": adulto.Nombre, "parentesco": adulto.Parentesco, "nacimiento": adulto.Nacimiento})
	}
	for i, m := range a.Menores {
		fila := map[string]string{"nombre": m.Nombre, "nacimiento": m.Nacimiento, "sexo": m.Sexo, "marca_h": "", "marca_m": ""}
		if m.CodigoSexo == "H" {
			fila["marca_h"] = a.MarcaSeleccion
		}
		if m.CodigoSexo == "M" {
			fila["marca_m"] = a.MarcaSeleccion
		}
		r.Filas["menores"] = append(r.Filas["menores"], fila)
		rotulo := ""
		if i == 0 {
			rotulo = a.RotuloMenor
		}
		punto := ""
		if i == len(a.Menores)-1 {
			punto = "."
		}
		r.Filas["lista_menores"] = append(r.Filas["lista_menores"], map[string]string{"nombre": m.Nombre, "rotulo_menor": rotulo, "punto_final": punto})
	}
	for k, v := range a.CamposSPM {
		r.Campos[k] = v
	}
	for k, v := range a.FilasSPM {
		r.Filas[k] = v
	}
	return r
}

type Plantilla struct{ ID, Tipo, Idioma, Version, Ruta, Huella, RolFirmante string }
type IdiomaPlantilla struct{ Ruta, Version string }
type Emision struct {
	ID              string            `json:"id"`
	Tipo            string            `json:"tipo"`
	ExpedienteID    string            `json:"expediente_id"`
	EquipoID        string            `json:"equipo_id,omitempty"`
	Plantilla       Plantilla         `json:"plantilla"`
	IdiomaPaquete   IdiomaPlantilla   `json:"idioma_paquete"`
	FechaDocumento  time.Time         `json:"fecha_documento"`
	GeneradaPor     string            `json:"generada_por"`
	GeneradaEn      time.Time         `json:"generada_en"`
	HuellaDatos     string            `json:"huella_datos"`
	HuellaResultado string            `json:"huella_sha256"`
	VersionesDatos  map[string]string `json:"versiones_datos"`
	VersionMotor    string            `json:"version_motor"`
	Aviso           string            `json:"aviso,omitempty"`
}
type Resultado struct {
	Emision   Emision
	Contenido []byte
	Repetida  bool
}
type ConflictoReproduccion struct{ Emision Emision }

func (*ConflictoReproduccion) Error() string { return "documento.reproduccion_no_disponible" }

// JSON ordena las claves de los mapas; sólo se incluyen valores usados y
// referencias técnicas. La forma canónica nunca se persiste ni se registra.
func HuellaDatos(d Datos, r Relleno, f Firmante) (string, error) {
	return huellaJSON(struct {
		Expediente, Equipo string
		Fecha              time.Time
		Versiones          map[string]string
		FirmanteID         string
		FirmanteVersion    int
		Relleno            Relleno
	}{d.ExpedienteID, d.EquipoID, d.FechaDocumento, d.Versiones, f.ID, f.Version, r})
}
func HuellaPeticion(s Solicitud, p Plantilla, idioma IdiomaPlantilla, datos, motor string) (string, error) {
	s.Clave = ""
	return huellaJSON(struct {
		Solicitud    Solicitud
		Plantilla    Plantilla
		Idioma       IdiomaPlantilla
		Datos, Motor string
	}{s, p, idioma, datos, motor})
}
func huellaJSON(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", ErrPeticion
	}
	defer clear(b)
	return Huella(b), nil
}
func Huella(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func FirmanteValido(f Firmante) bool {
	return IDValido(f.ID) && f.Version > 0 && strings.TrimSpace(f.Nombre) != ""
}

type ConflictoVersion struct{ VersionActual int }

func (ConflictoVersion) Error() string { return "error.conflicto_version" }
