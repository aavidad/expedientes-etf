package expediente

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

func (a Actor) AutorizarLecturaEquipo() error {
	if !IDValido(a.UsuarioID) {
		return ErrAcceso
	}
	if slices.Contains(a.Perfiles, "coordinacion") || (slices.Contains(a.Perfiles, "profesional_etf") && IDValido(a.EquipoID)) {
		return nil
	}
	return ErrAcceso
}

type FiltroEquipo struct {
	Busqueda       string
	Estado         string
	Subprograma    string
	Fase           string
	Limite         int
	Desplazamiento int
}

func (f FiltroEquipo) Validar() error {
	if !utf8.ValidString(f.Busqueda) || strings.ContainsAny(f.Busqueda, "\n\r\x00") || utf8.RuneCountInString(f.Busqueda) > 100 || f.Limite < 1 || f.Limite > 100 || f.Desplazamiento < 0 || f.Desplazamiento > 10000 ||
		!slices.Contains([]string{"", "aceptada", "en_intervencion", "cerrada"}, f.Estado) ||
		!slices.Contains([]string{"", "RIESGO_SOCIAL", "REUNIFICACION"}, f.Subprograma) ||
		!slices.Contains([]string{"", "recepcion", "evaluacion", "intervencion", "cierre_devolucion", "seguimiento"}, f.Fase) {
		return ErrPeticion
	}
	return nil
}

type ResumenEquipo struct {
	ID              string    `json:"id"`
	EquipoID        string    `json:"equipo_id"`
	Numero          string    `json:"numero"`
	Familia         string    `json:"familia"`
	Estado          string    `json:"estado"`
	Subprograma     string    `json:"subprograma"`
	Fase            string    `json:"fase"`
	FechaRecepcion  time.Time `json:"fecha_recepcion"`
	FechaAceptacion string    `json:"fecha_aceptacion"`
	Menores         int       `json:"menores"`
	Version         int       `json:"version"`
	Avisos          []string  `json:"avisos"`
}

type PaginaEquipo struct {
	EquipoClave string          `json:"equipo_clave"`
	Filas       []ResumenEquipo `json:"filas"`
	HayMas      bool            `json:"hay_mas"`
}

type SucesoExpediente struct {
	ID               string    `json:"id"`
	Tipo             string    `json:"tipo"`
	Codigo           string    `json:"codigo"`
	Fecha            time.Time `json:"fecha"`
	Autor            string    `json:"autor"`
	VersionPlantilla string    `json:"version_plantilla,omitempty"`
	Resultado        string    `json:"resultado,omitempty"`
}

type FichaEquipo struct {
	SalidasServicio []SalidaServicio `json:"salidas_servicio"`
	ResumenEquipo
	Hitos         []SucesoExpediente `json:"hitos"`
	Emisiones     []SucesoExpediente `json:"emisiones"`
	Historial     []SucesoExpediente `json:"historial"`
	Capacidades   []string           `json:"capacidades"`
	MotivosCierre []OpcionCierre     `json:"motivos_cierre"`
	PuedeProponer bool               `json:"puede_proponer"`
}

type OpcionCierre struct {
	Codigo            string `json:"codigo"`
	AdmiteTexto       bool   `json:"admite_texto"`
	SinConsentimiento bool   `json:"sin_consentimiento"`
}
