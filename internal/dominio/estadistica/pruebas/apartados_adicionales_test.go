package estadistica_test

import (
	"testing"

	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

func limiteSMI(v int64) *int64 { return &v }

func TestApartados14a16y26a28ConRecuentosManuales(t *testing.T) {
	in := entrada()
	in.Actividad = &e.RecuentoActividad{Anio: 2026, Comisiones: 3, Informes: 5, PorOrganismo: map[string]int64{"spm": 2, "juzgados": 2, "": 1}}
	in.Reglas[0].Dato.Valor.RecursosCoordinacion = []string{"spm", "sscc", "otros"}
	in.Reglas[0].Dato.Valor.FactoresMenores = []string{"factor_a", "factor_b"}
	in.Reglas[0].Dato.Valor.TipologiasDesproteccion = []string{"tipologia_a"}
	in.Reglas[0].Dato.Valor.ResponsablesDesproteccion = []string{"padre", "madre", "otros"}
	x := expediente("RESTO", "2026-01-01", "", "", e.Riesgo, menor("m1", "2018-01-01", e.Nina, e.Riesgo), menor("m2", "2017-01-01", e.Nino, e.Riesgo))
	x.RecursosCoordinacion = historia("recursos", []string{"spm"})
	x.Responsables = historia("responsables", []string{"padre", "otros"})
	x.Factores = historia("factores-resto", []e.Factor{
		{Codigo: "factor_a", Presente: conocido(true), MenoresExcluidos: []string{"m2"}},
		{Codigo: "factor_b", Presente: e.Dato[bool]{}},
		{Codigo: "tipologia_a", Presente: conocido(true)},
	})
	x.Menores = historia("menores-resto", []e.Menor{
		{ID: "m1", Nacimiento: dia("2018-01-01"), Sexo: e.Nina, Subprograma: e.Riesgo, PropuestaAtencionTemprana: conocido(true)},
		{ID: "m2", Nacimiento: dia("2017-01-01"), Sexo: e.Nino, Subprograma: e.Riesgo, PropuestaAtencionTemprana: conocido(false)},
	})
	in.Expedientes = []e.Expediente{x}
	m := calcular(t, in)
	comprobarCelda(t, m.Apartados[e.CoordinacionRecursos], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Estado: "spm"}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.CoordinacionRecursos], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Estado: "sscc"}, e.Menores, 0, 0)
	comprobarCelda(t, m.Apartados[e.AtencionTemprana], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Estado: "si"}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.FactoresMenores], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Estado: "factor_a"}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.FactoresMenores], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Estado: "factor_a"}, e.Menores, 0, 0)
	comprobarCelda(t, m.Apartados[e.FactoresMenores], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Estado: "factor_b"}, e.Menores, 0, 1)
	comprobarCelda(t, m.Apartados[e.TipologiasDesproteccion], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Estado: "tipologia_a"}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.ResponsablesDesproteccion], e.Grupo{Estado: "padre"}, e.Familias, 1, 0)
	if m.Apartados[e.ComisionesTecnicas].Total.Comisiones != 3 || m.Apartados[e.InformesTecnicos].Total.Informes != 5 {
		t.Fatalf("actividad: comisiones %+v informes %+v", m.Apartados[e.ComisionesTecnicas].Total, m.Apartados[e.InformesTecnicos].Total)
	}
	if c := m.Apartados[e.InformesTecnicos].Celdas[e.ClaveCelda{Grupo: e.Grupo{Organismo: "juzgados"}, Unidad: e.Informes}]; c.Numerador != 2 {
		t.Fatalf("informes por organismo: %+v", c)
	}
	if c := m.Apartados[e.InformesTecnicos].Celdas[e.ClaveCelda{Grupo: e.Grupo{}, Unidad: e.Informes}]; c.SinDato != 1 {
		t.Fatalf("organismo desconocido: %+v", c)
	}
	comprobarCelda(t, m.Apartados[e.AtencionTemprana], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Estado: "no"}, e.Menores, 0, 0)
}

func TestApartadosActividadSinPuertoDeclaranSinDato(t *testing.T) {
	m := calcular(t, entrada())
	for _, a := range []e.Apartado{e.ComisionesTecnicas, e.InformesTecnicos} {
		if m.Apartados[a].Avisos["estadistica.recuento_actividad_sin_dato"] != 1 {
			t.Fatalf("apartado %s oculta ausencia del puerto: %+v", a, m.Apartados[a].Avisos)
		}
	}
}

func TestApartadosAdicionalesConRecuentosManuales(t *testing.T) {
	in := entrada()
	in.Reglas[0].Dato.Valor.SMIAnualCentimos = 1_000_000
	in.Reglas[0].Dato.Valor.TramosSMI = []e.TramoSMI{
		{Codigo: "menos_smi", DesdeCentimos: 0, HastaCentimos: limiteSMI(100)},
		{Codigo: "uno_a_dos_smi", DesdeCentimos: 100, HastaCentimos: limiteSMI(200)},
		{Codigo: "mas_dos_smi", DesdeCentimos: 200},
	}
	in.Reglas[0].Dato.Valor.FactoresFamilia = []string{"ingresos_insuficientes", "conflicto_familiar"}
	propuestas := e.Propuestas{
		DeclaracionRiesgo: conocido(true), PIFDerivadoDSR: conocido(true),
		MedidaProtectora: conocido(true), RespuestaSPM: conocido("a_la_espera"),
	}
	a := expediente("M-A", "2026-01-01", "2026-04-01", e.ObjetivosCumplidos, e.Riesgo,
		menor("m-a1", "2018-01-01", e.Nina, e.Riesgo), menor("m-a2", "2014-01-01", e.Nino, e.Riesgo))
	a.Antecedentes = historia("m-a-ant", true)
	a.ServicioDerivante = historia("m-a-ser", "spm")
	a.Consentimiento = historia("m-a-cons", true)
	a.DerivadoSSCCTrasCierre = historia("m-a-sscc", true)
	a.FamiliaNumerosa = historia("m-a-num", true)
	a.Composicion = historia("m-a-comp", "biparental")
	a.SituacionEconomica = historia("m-a-econ", "ingresos_insuficientes")
	a.Ingresos = historia("m-a-ing", e.Ingresos{NetosAnualesCentimos: conocido(int64(1_500_000))})
	a.Adultos = historia("m-a-adultos", []e.Adulto{
		{Papel: "padre", Sexo: e.Nino, Nacimiento: dia("1984-01-01"), NivelEducativo: "secundaria", SituacionLaboral: "empleo"},
		{Papel: "madre", Sexo: e.Nina, Nacimiento: dia("1990-01-01"), NivelEducativo: "primaria", SituacionLaboral: "desempleo"},
	})
	a.Factores = historia("m-a-factores", []e.Factor{
		{Codigo: "ingresos_insuficientes", Presente: conocido(true)},
		{Codigo: "conflicto_familiar", Presente: conocido(false)},
	})
	a.Propuestas = historia("m-a-propuestas", propuestas)

	b := expediente("M-B", "2026-01-01", "", "", e.Riesgo,
		menor("m-b1", "2016-06-01", e.Nino, e.Riesgo))
	b.ServicioDerivante = historia("m-b-ser", "sscc")
	b.Consentimiento = historia("m-b-cons", false)
	b.Propuestas = historia("m-b-propuestas", e.Propuestas{
		DeclaracionRiesgo: conocido(false), PIFDerivadoDSR: conocido(false),
		MedidaProtectora: conocido(false),
	})

	c := expediente("M-C", "2026-01-01", "", "", e.Reunificacion,
		menor("m-c1", "2015-06-01", e.Nina, e.Reunificacion))
	c.ServicioDerivante = historia("m-c-ser", "otro_etf")
	c.Consentimiento = historia("m-c-cons", true)
	c.Propuestas = historia("m-c-propuestas", e.Propuestas{
		DeclaracionRiesgo: conocido(false), PIFDerivadoDSR: conocido(false),
		MedidaProtectora: conocido(false),
	})
	c.SeguimientosTrasCese = []e.Vigencia{{Desde: dia("2026-02-01"), Hasta: dia("2027-02-01")}}

	noAceptado := e.Expediente{ID: "M-D", EquipoID: "etf-01", NoAceptaciones: []e.NoAceptacion{{
		Fecha: dia("2026-05-10"), ServicioDerivante: "spm", Causa: "sin_consentimiento", Subprograma: e.Riesgo,
	}}}
	in.Expedientes = []e.Expediente{a, b, c, noAceptado}
	m := calcular(t, in)

	comprobarCelda(t, m.Apartados[e.Antecedentes], e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.ServicioDerivante], e.Grupo{Subprograma: e.Riesgo, Estado: "spm"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.NoAceptados], e.Grupo{Subprograma: e.Riesgo, Estado: "spm", Causa: "sin_consentimiento"}, e.Familias, 1, 0)
	if m.Apartados[e.Pendientes].Total != (e.Recuento{Familias: 1, Menores: 1}) {
		t.Fatalf("pendientes calculados: %+v", m.Apartados[e.Pendientes].Total)
	}
	comprobarCelda(t, m.Apartados[e.DerivadosSSCC], e.Grupo{Estado: "si"}, e.Familias, 1, 0)
	for _, a := range []e.Apartado{e.PropuestasDSR, e.PIFDerivadoDSR, e.PropuestasMedida} {
		comprobarCelda(t, m.Apartados[a], e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, e.Familias, 1, 0)
		comprobarCelda(t, m.Apartados[a], e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, e.Menores, 2, 0)
		c := m.Apartados[a].Celdas[e.ClaveCelda{Grupo: e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, Unidad: e.Familias}]
		if c.Denominador != conocido(2) || c.Porcentaje == nil || c.Porcentaje.Centesimas != 5000 {
			t.Fatalf("porcentaje apartado %s: %+v", a, c)
		}
	}
	comprobarCelda(t, m.Apartados[e.RespuestaSPM], e.Grupo{Estado: "a_la_espera"}, e.Familias, 1, 0)
	respuesta := m.Apartados[e.RespuestaSPM].Celdas[e.ClaveCelda{Grupo: e.Grupo{Estado: "a_la_espera"}, Unidad: e.Familias}]
	if respuesta.Porcentaje == nil || respuesta.Porcentaje.Centesimas != 10000 {
		t.Fatalf("porcentaje respuesta SPM: %+v", respuesta)
	}
	comprobarCelda(t, m.Apartados[e.SeguimientoReunificacion], e.Grupo{Subprograma: e.Reunificacion}, e.Familias, 1, 0)
	seguimiento := m.Apartados[e.SeguimientoReunificacion].Celdas[e.ClaveCelda{Grupo: e.Grupo{Subprograma: e.Reunificacion}, Unidad: e.Familias}]
	if seguimiento.Porcentaje == nil || seguimiento.Porcentaje.Centesimas != 10000 {
		t.Fatalf("porcentaje seguimiento: %+v", seguimiento)
	}
	comprobarCelda(t, m.Apartados[e.ComposicionFamiliar], e.Grupo{Subprograma: e.Riesgo, Estado: "biparental"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.FamiliasNumerosas], e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.EdadParental], e.Grupo{Papel: "padre", Sexo: e.Nino, Tramo: "edad_40_49"}, e.Adultos, 1, 0)
	if m.Apartados[e.EdadParental].Total.Adultos != 2 || m.Apartados[e.NivelEducativoParental].Total.Adultos != 2 || m.Apartados[e.SituacionLaboralParental].Total.Adultos != 2 {
		t.Fatal("población adulta parental inconsistente")
	}
	comprobarCelda(t, m.Apartados[e.NivelEducativoParental], e.Grupo{Papel: "madre", Sexo: e.Nina, Estado: "primaria"}, e.Adultos, 1, 0)
	comprobarCelda(t, m.Apartados[e.SituacionLaboralParental], e.Grupo{Papel: "madre", Sexo: e.Nina, Estado: "desempleo"}, e.Adultos, 1, 0)
	comprobarCelda(t, m.Apartados[e.SituacionEconomica], e.Grupo{Subprograma: e.Riesgo, Estado: "ingresos_insuficientes"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.IngresosSMI], e.Grupo{Subprograma: e.Riesgo, Tramo: "uno_a_dos_smi"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.FactoresFamilia], e.Grupo{Subprograma: e.Riesgo, Estado: "ingresos_insuficientes"}, e.Familias, 1, 1)
	comprobarCelda(t, m.Apartados[e.FactoresFamilia], e.Grupo{Subprograma: e.Riesgo, Estado: "conflicto_familiar"}, e.Familias, 0, 1)
}

func TestApartado6ExigeMasDeTreintaDiasYSeparaConsentimientoDesconocido(t *testing.T) {
	in := entrada()
	enElLimite := expediente("P30", "2026-12-01", "", "", e.Riesgo)
	enElLimite.Consentimiento = historia("p30-cons", false)
	masDeTreinta := expediente("P31", "2026-11-30", "", "", e.Riesgo)
	masDeTreinta.Consentimiento = nil
	cerrado := expediente("PC", "2026-01-01", "2026-10-01", e.Traslado, e.Riesgo)
	cerrado.Consentimiento = historia("pc-cons", false)
	in.Expedientes = []e.Expediente{enElLimite, masDeTreinta, cerrado}
	r := calcular(t, in).Apartados[e.Pendientes]
	if r.Total.Familias != 1 {
		t.Fatalf("pendientes: %+v", r.Total)
	}
	comprobarCelda(t, r, e.Grupo{Subprograma: e.Riesgo}, e.Familias, 0, 1)
}

func TestPorcentajeMarcaBaseDesconocida(t *testing.T) {
	in := entrada()
	conocidoEnRiesgo := expediente("R", "2026-01-01", "", "", e.Riesgo)
	conocidoEnRiesgo.Propuestas = historia("r-propuestas", e.Propuestas{DeclaracionRiesgo: conocido(true)})
	subprogramaPendiente := expediente("S", "2026-01-01", "", "", e.Riesgo)
	subprogramaPendiente.Subprograma = nil
	in.Expedientes = []e.Expediente{conocidoEnRiesgo, subprogramaPendiente}
	r := calcular(t, in).Apartados[e.PropuestasDSR]
	c := r.Celdas[e.ClaveCelda{Grupo: e.Grupo{Subprograma: e.Riesgo, Estado: "si"}, Unidad: e.Familias}]
	if c.Porcentaje == nil || c.Denominador.Conocido || c.Porcentaje.Aviso != "estadistica.denominador_pendiente" {
		t.Fatalf("base con población de riesgo sin dato: %+v", c)
	}
}

// El juego de F4.09 aporta tres familias, seis menores y 23 factores
// valorados sin marcas. El total es tres; cada factor tiene cero marcas.
func TestApartado24CuentaFamiliasUnaVez(t *testing.T) {
	for _, caso := range []string{"sin_factores", "seleccion_multiple", "sin_valoracion"} {
		t.Run(caso, func(t *testing.T) {
			in := entrada()
			codigos := []string{
				"conciencia", "motivacion", "desempleo", "precariedad", "economia",
				"desahucio", "vivienda_compartida", "habitabilidad", "conflictos", "violencia",
				"familia_extensa", "adicciones", "relacion_parental", "parentalidad_adolescente",
				"salud_mental", "planificacion", "conductas_delictivas", "prision", "aislamiento",
				"entorno", "situacion_irregular", "discapacidad", "desigualdad",
			}
			in.Reglas[0].Dato.Valor.FactoresFamilia = codigos
			in.Expedientes = []e.Expediente{
				expediente("A", "2025-01-15", "2026-02-15", e.ObjetivosCumplidos, e.Riesgo,
					menor("a1", "2015-03-12", e.Nino, e.Riesgo), menor("a2", "2017-08-21", e.Nina, e.Riesgo)),
				expediente("B", "2026-07-01", "", "", e.Riesgo,
					menor("b1", "2008-12-20", e.Nino, e.Riesgo), menor("b2", "2021-02-11", e.Nina, e.Riesgo)),
				expediente("C", "2026-01-01", "2026-05-31", e.ObjetivosCumplidos, e.Reunificacion,
					menor("c1", "2016-04-01", e.Nino, e.Reunificacion), menor("c2", "2010-05-05", e.Nina, e.Reunificacion)),
				// Cerrada antes del año y aceptada después del año: ninguna entra.
				expediente("D", "2024-01-01", "2025-12-31", e.ObjetivosCumplidos, e.Riesgo),
				expediente("E", "2027-01-01", "", "", e.Riesgo),
			}
			for i := range in.Expedientes {
				x := &in.Expedientes[i]
				if caso == "sin_valoracion" {
					continue
				}
				factores := []e.Factor{}
				if caso == "seleccion_multiple" {
					factores = []e.Factor{
						{Codigo: codigos[0], Presente: conocido(true)},
						{Codigo: codigos[1], Presente: conocido(true)},
					}
				}
				x.Factores = historia(x.ID+"-factores", factores)
				x.ListasConocidas = historia(x.ID+"-listas", []string{"factor_familia"})
			}
			m := calcular(t, in)
			r := m.Apartados[e.FactoresFamilia]
			if r.Total != (e.Recuento{Familias: 3}) {
				t.Fatalf("apartado 24: %+v; esperado tres familias", r.Total)
			}
			for j, codigo := range codigos {
				for _, grupo := range []struct {
					s e.Subprograma
					n int
				}{{e.Riesgo, 2}, {e.Reunificacion, 1}} {
					n, sin := 0, 0
					if caso == "seleccion_multiple" && j < 2 {
						n = grupo.n
					}
					if caso == "sin_valoracion" {
						sin = grupo.n
					}
					comprobarCelda(t, r, e.Grupo{Subprograma: grupo.s, Estado: codigo}, e.Familias, n, sin)
				}
			}
		})
	}
}
