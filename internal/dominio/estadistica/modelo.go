package estadistica

import "time"

// Error contiene sólo una clave, sin valores del expediente.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrEntrada      Error = "estadistica.entrada_invalida"
	ErrHistoria     Error = "estadistica.historia_invalida"
	ErrSolapamiento Error = "estadistica.vigencias_solapadas"
	ErrReglas       Error = "estadistica.reglas_invalidas"
)

type Subprograma string

const (
	Riesgo        Subprograma = "riesgo"
	Reunificacion Subprograma = "reunificacion"
)

type Sexo string

const (
	Nino Sexo = "nino"
	Nina Sexo = "nina"
)

// Motivo es el código estadístico de cierre, ya traducido por el catálogo.
// El valor vacío indica que no consta; no equivale a OtrasCausas.
type Motivo string

const (
	ObjetivosCumplidos Motivo = "objetivos_cumplidos"
	ObjetivosParciales Motivo = "objetivos_parciales"
	Rechazo            Motivo = "rechazo"
	NoAdherencia       Motivo = "no_adherencia"
	Traslado           Motivo = "traslado"
	MayoriaEdad        Motivo = "mayoria_edad"
	PropuestaDSR       Motivo = "propuesta_dsr"
	MedidaProtectora   Motivo = "medida_protectora"
	IngresoCentro      Motivo = "ingreso_centro"
	OtrasCausas        Motivo = "otras_causas"
)

type Dato[T any] struct {
	Valor    T
	Conocido bool
}

// Vigencia usa fechas civiles: desde incluido, hasta excluido; cero = sin fin.
type Vigencia struct{ Desde, Hasta time.Time }

// Revision conserva la revisión sustituida. Sustituye reemplaza la revisión
// completa: si cambia el intervalo, el productor aporta también los fragmentos
// que deban seguir vigentes. RegistradoEn es un instante, no una fecha de negocio.
type Revision[T any] struct {
	ID           string
	Vigencia     Vigencia
	RegistradoEn time.Time
	Sustituye    string
	Dato         Dato[T]
}

type Historia[T any] []Revision[T]

// Episodio conserva la apertura y cierre confirmados. Aceptacion identifica la
// fecha del Acta aplicable al episodio; cero declara una fecha desconocida.
// Reabrir aporta otro episodio enlazado, sin borrar el cierre del anterior.
type Episodio struct {
	ID                           string
	Apertura, Aceptacion, Cierre time.Time
	Motivo                       Motivo
	ReaperturaDe                 string
}

type Menor struct {
	ID                        string
	Nacimiento                time.Time   // cero = desconocido
	Sexo                      Sexo        // vacío = desconocido
	Subprograma               Subprograma // propio, nunca deducido del de la familia
	PropuestaAtencionTemprana Dato[bool]
	CentroProteccion          Dato[string]
}

type Ingresos struct {
	NetosAnualesCentimos Dato[int64]
	Tramo                Dato[string]
}

// Adulto conserva sólo dimensiones estadísticas, nunca identidad personal.
type Adulto struct {
	Papel            string
	Sexo             Sexo
	Nacimiento       time.Time
	NivelEducativo   string
	SituacionLaboral string
}

type NoAceptacion struct {
	Fecha             time.Time
	ServicioDerivante string
	Causa             string
	Subprograma       Subprograma
}

type Propuestas struct {
	DeclaracionRiesgo Dato[bool]
	PIFDerivadoDSR    Dato[bool]
	MedidaProtectora  Dato[bool]
	RespuestaSPM      Dato[string]
}

const (
	RespuestaInicioProcedimiento = "inicio_desamparo_o_medida"
	RespuestaEnEspera            = "a_la_espera"
	RespuestaNoNecesaria         = "no_necesaria"
)

type TramoSMI struct {
	Codigo        string
	DesdeCentimos int64 // centésimas del SMI anual
	HastaCentimos *int64
}

// Factor permite selección múltiple y exclusiones individuales (D4).
// Conocido=false distingue una valoración pendiente de una selección negativa.
type Factor struct {
	Lista            string // vacío mantiene el contrato de productores anteriores
	Codigo           string
	Presente         Dato[bool]
	MenoresExcluidos []string
}

// Expediente es una proyección estadística: no contiene nombres ni narrativas.
// Menores versiona conjuntamente pertenencia y atributos; una lista conocida
// vacía significa cero menores, una historia ausente significa sin dato.
type Expediente struct {
	ID, EquipoID           string
	Episodios              []Episodio
	Estados                Historia[string]
	SeguimientosTrasCese   []Vigencia // seguimiento confirmado de Reunificación
	Composicion            Historia[string]
	Ingresos               Historia[Ingresos]
	Subprograma            Historia[Subprograma]
	Menores                Historia[[]Menor]
	Factores               Historia[[]Factor]
	ListasConocidas        Historia[[]string]
	Antecedentes           Historia[bool]
	ServicioDerivante      Historia[string]
	NoAceptaciones         []NoAceptacion
	Consentimiento         Historia[bool]
	DerivadoSSCCTrasCierre Historia[bool]
	FamiliaNumerosa        Historia[bool]
	Adultos                Historia[[]Adulto]
	SituacionEconomica     Historia[string]
	Propuestas             Historia[Propuestas]
	RecursosCoordinacion   Historia[[]string]
	Responsables           Historia[[]string]
}

// Tramo comprende meses enteros en [DesdeMeses, HastaMeses); nil = sin límite.
// No se incluyen tramos predeterminados: los decide el catálogo versionado.
type Tramo struct {
	Codigo     string
	DesdeMeses int
	HastaMeses *int
}

type Parametros struct {
	TramosRiesgo, TramosReunificacion                                   []Tramo
	TramosSMI                                                           []TramoSMI
	FactoresFamilia                                                     []string
	FactoresMenores, TipologiasDesproteccion, ResponsablesDesproteccion []string
	RecursosCoordinacion                                                []string
	SMIAnualCentimos                                                    int64
}

type Entrada struct {
	Anio         int
	FechaCalculo time.Time            // limita los hechos de un año en curso
	ConocidoEn   time.Time            // limita las revisiones disponibles para reproducir la foto
	Reglas       Historia[Parametros] // se seleccionan a 31/12/Anio
	Expedientes  []Expediente         // alcance autorizado por la aplicación
	Actividad    *RecuentoActividad   // cifras agregadas del puerto actividad.ConsultaMemoria
}

// RecuentoActividad refleja respuesta del puerto de actividad, sin filas ni observaciones.
type RecuentoActividad struct {
	Anio                 int
	Comisiones, Informes int64
	PorOrganismo         map[string]int64
}

func fecha(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (v Vigencia) contiene(t time.Time) bool {
	t = fecha(t)
	return !t.Before(fecha(v.Desde)) && (v.Hasta.IsZero() || t.Before(fecha(v.Hasta)))
}

func (v Vigencia) valida() bool {
	return !v.Desde.IsZero() && (v.Hasta.IsZero() || fecha(v.Hasta).After(fecha(v.Desde)))
}

func (v Vigencia) cruza(desde, hasta time.Time) bool {
	return !fecha(v.Desde).After(hasta) && (v.Hasta.IsZero() || fecha(v.Hasta).After(desde))
}
