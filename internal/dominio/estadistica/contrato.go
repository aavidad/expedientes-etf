package estadistica

import (
	"math"
	"time"
)

type Apartado string

const (
	CasosAtendidos            Apartado = "1"
	MenoresPorSexo            Apartado = "2"
	Antecedentes              Apartado = "3"
	ServicioDerivante         Apartado = "4"
	NoAceptados               Apartado = "5"
	Pendientes                Apartado = "6"
	CasosCerrados             Apartado = "7"
	TratamientoConcluido      Apartado = "8"
	TratamientoInterrumpido   Apartado = "9"
	DerivadosSSCC             Apartado = "10"
	PropuestasDSR             Apartado = "11.1"
	PIFDerivadoDSR            Apartado = "11.2"
	PropuestasMedida          Apartado = "12.1"
	RespuestaSPM              Apartado = "12.2"
	DuracionRiesgo            Apartado = "13.1"
	DuracionReunificacion     Apartado = "13.2"
	SeguimientoReunificacion  Apartado = "13.3"
	ComposicionFamiliar       Apartado = "17"
	FamiliasNumerosas         Apartado = "18"
	EdadParental              Apartado = "19"
	NivelEducativoParental    Apartado = "20"
	SituacionLaboralParental  Apartado = "21"
	SituacionEconomica        Apartado = "22"
	IngresosSMI               Apartado = "23"
	FactoresFamilia           Apartado = "24"
	MenoresPorEdad            Apartado = "25"
	CoordinacionRecursos      Apartado = "14"
	AtencionTemprana          Apartado = "14.1"
	ComisionesTecnicas        Apartado = "15"
	InformesTecnicos          Apartado = "16"
	FactoresMenores           Apartado = "26"
	TipologiasDesproteccion   Apartado = "27"
	ResponsablesDesproteccion Apartado = "28"
)

type Unidad string

const (
	Familias   Unidad = "familias"
	Menores    Unidad = "menores"
	Adultos    Unidad = "adultos"
	Comisiones Unidad = "comisiones"
	Informes   Unidad = "informes"
)

type Contrato struct {
	Apartado         Apartado
	Poblacion        string // AT, ME, CI, CO o NC (ESQUEMA §1.10)
	Denominador      string // pendiente_programa: sólo recuentos en esta entrega
	Desconocidos     string
	FechaDatos       string // corte o cierre_episodio
	Corte, FechaEdad time.Time
	VersionRegla     string
	DivisionCero     string
}

// Grupo conserva las dimensiones conocidas; el valor vacío declara sin dato.
// Tramo contiene un código de catálogo (duración) o del contrato de edad.
type Grupo struct {
	Subprograma Subprograma
	Origen      string // anterior o incorporado
	Sexo        Sexo
	Tramo       string
	Motivo      Motivo
	Papel       string // rol parental
	Estado      string // situación afirmativa/negativa
	Causa       string // causa de no aceptación
	Organismo   string
}

type ClaveCelda struct {
	Grupo  Grupo
	Unidad Unidad
}

type Celda struct {
	Numerador               int
	Denominador             Dato[int] // desconocido mientras la base no esté validada
	Porcentaje              *Porcentaje
	SinDato                 int
	Poblacion, VersionRegla string
}

type Recuento struct {
	Familias, Menores          int
	Adultos                    int
	Comisiones, Informes       int
	FamiliasSinRelacionMenores int // cardinalidad desconocida, nunca cero menores
}

type Resultado struct {
	Contrato Contrato
	Total    Recuento
	Celdas   map[ClaveCelda]Celda
	Avisos   map[string]int // claves y cantidades; sin identificadores de personas
}

type Memoria struct {
	Anio              int
	Corte, ConocidoEn time.Time
	Apartados         map[Apartado]*Resultado
}

// Porcentaje devuelve centésimas de punto porcentual: 3333 equivale a 33,33 %.
// La localización pertenece a la presentación. No inventa una base desconocida.
type Porcentaje struct {
	Centesimas int
	Aviso      string
}

func CalcularPorcentaje(numerador int, denominador Dato[int]) (Porcentaje, error) {
	if numerador < 0 || (denominador.Conocido && denominador.Valor < 0) {
		return Porcentaje{}, ErrEntrada
	}
	if !denominador.Conocido {
		return Porcentaje{Aviso: "estadistica.denominador_pendiente"}, nil
	}
	if denominador.Valor == 0 {
		return Porcentaje{Aviso: "estadistica.divisor_cero"}, nil
	}
	valor := math.Round(float64(numerador) / float64(denominador.Valor) * 10000)
	if valor >= float64(int(^uint(0)>>1)) {
		return Porcentaje{}, ErrEntrada
	}
	return Porcentaje{Centesimas: int(valor)}, nil
}

func nuevoResultado(a Apartado, poblacion, fechaDatos, version string, corte, edad time.Time) *Resultado {
	return &Resultado{
		Contrato: Contrato{Apartado: a, Poblacion: poblacion, Denominador: "pendiente_programa", Desconocidos: "sin_dato_separado", FechaDatos: fechaDatos, Corte: corte, FechaEdad: edad, VersionRegla: version, DivisionCero: "estadistica.divisor_cero"},
		Celdas:   make(map[ClaveCelda]Celda), Avisos: make(map[string]int),
	}
}

func (r *Resultado) sumar(g Grupo, u Unidad, desconocido bool) {
	k := ClaveCelda{Grupo: g, Unidad: u}
	c := r.Celdas[k]
	c.Poblacion, c.VersionRegla = r.Contrato.Poblacion, r.Contrato.VersionRegla
	if desconocido {
		c.SinDato++
	} else {
		c.Numerador++
	}
	r.Celdas[k] = c
}
