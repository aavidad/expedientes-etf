package estadistica_test

import (
	"testing"
	"time"

	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

func TestDivisorCeroYPorcentajesConBaseExplicita(t *testing.T) {
	// Los esperados son porcentajes manuales en centésimas de punto:
	// 1/3=33,33 %, 2/3=66,67 %, 1/8=12,50 %, 2/1=200,00 %.
	for _, caso := range []struct {
		n        int
		d        e.Dato[int]
		esperado e.Porcentaje
	}{
		{0, conocido(0), e.Porcentaje{Aviso: "estadistica.divisor_cero"}},
		{1, conocido(0), e.Porcentaje{Aviso: "estadistica.divisor_cero"}},
		{1, e.Dato[int]{}, e.Porcentaje{Aviso: "estadistica.denominador_pendiente"}},
		{1, conocido(3), e.Porcentaje{Centesimas: 3333}},
		{2, conocido(3), e.Porcentaje{Centesimas: 6667}},
		{1, conocido(8), e.Porcentaje{Centesimas: 1250}},
		{2, conocido(1), e.Porcentaje{Centesimas: 20000}},
	} {
		p, err := e.CalcularPorcentaje(caso.n, caso.d)
		if err != nil || p != caso.esperado {
			t.Fatalf("%d/%+v: %+v, %v", caso.n, caso.d, p, err)
		}
	}
	if _, err := e.CalcularPorcentaje(-1, conocido(2)); err != e.ErrEntrada {
		t.Fatal(err)
	}
	if _, err := e.CalcularPorcentaje(1, conocido(-2)); err != e.ErrEntrada {
		t.Fatal(err)
	}
	// Una Memoria sin expedientes mantiene cero recuentos y ninguna base
	// de porcentaje inventada para rellenar el vacío.
	for a, r := range calcular(t, entrada()).Apartados {
		if r.Total != (e.Recuento{}) {
			t.Fatal(r)
		}
		switch a {
		case e.PropuestasDSR, e.PIFDerivadoDSR, e.PropuestasMedida, e.RespuestaSPM, e.SeguimientoReunificacion:
			if len(r.Celdas) == 0 || r.Avisos["estadistica.divisor_cero"] == 0 {
				t.Fatal(r)
			}
			for _, c := range r.Celdas {
				if c.Porcentaje == nil || *c.Porcentaje != (e.Porcentaje{Aviso: "estadistica.divisor_cero"}) {
					t.Fatal(c)
				}
			}
		default:
			if len(r.Celdas) != 0 {
				t.Fatal(r)
			}
		}
	}
}

func TestFronterasDeMesesYTramos(t *testing.T) {
	for _, caso := range []struct {
		inicio, fin string
		meses       int
	}{
		{"2026-01-31", "2026-02-28", 0}, {"2024-02-29", "2025-02-28", 11},
		{"2026-01-01", "2026-01-01", 0}, {"2026-01-02", "2026-07-01", 5},
		{"2026-01-02", "2026-07-02", 6}, {"2025-01-02", "2026-01-02", 12},
		{"2025-01-02", "2026-02-02", 13}, {"2025-01-02", "2026-07-02", 18},
		{"2025-01-02", "2026-08-02", 19},
	} {
		n, err := e.MesesCompletos(dia(caso.inicio), dia(caso.fin))
		if err != nil || n != caso.meses {
			t.Fatalf("%s a %s: %d; %v", caso.inicio, caso.fin, n, err)
		}
	}
	if _, err := e.MesesCompletos(time.Time{}, dia("2026-01-01")); err != e.ErrEntrada {
		t.Fatal(err)
	}
	if _, err := e.MesesCompletos(dia("2026-01-02"), dia("2026-01-01")); err != e.ErrEntrada {
		t.Fatal(err)
	}
	in := entrada()
	// Fechas a mano producen 5,6,12,13,18,19 meses: reparto 1,2,2,1.
	for _, caso := range []struct{ id, aceptacion, cierre string }{
		{"M5", "2026-01-02", "2026-07-01"}, {"M6", "2026-01-02", "2026-07-02"},
		{"M12", "2025-01-02", "2026-01-02"}, {"M13", "2025-01-02", "2026-02-02"},
		{"M18", "2025-01-02", "2026-07-02"}, {"M19", "2025-01-02", "2026-08-02"},
	} {
		in.Expedientes = append(in.Expedientes, expediente(caso.id, caso.aceptacion, caso.cierre, e.ObjetivosCumplidos, e.Riesgo))
	}
	r := calcular(t, in).Apartados[e.DuracionRiesgo]
	for _, caso := range []struct {
		tramo string
		n     int
	}{{"d0", 1}, {"d6", 2}, {"d13", 2}, {"d19", 1}} {
		comprobarCelda(t, r, e.Grupo{Subprograma: e.Riesgo, Tramo: caso.tramo}, e.Familias, caso.n, 0)
	}
}

func TestEdadEnLimitesYNota43(t *testing.T) {
	in := entrada()
	in.Expedientes = []e.Expediente{expediente("L", "2025-01-01", "", "", e.Riesgo,
		menor("l5", "2021-01-01", e.Nino, e.Riesgo), menor("l6", "2020-12-31", e.Nino, e.Riesgo),
		menor("l11", "2015-01-01", e.Nino, e.Riesgo), menor("l12", "2014-12-31", e.Nino, e.Riesgo),
		menor("l17", "2009-01-01", e.Nino, e.Riesgo), menor("l18", "2008-12-31", e.Nino, e.Riesgo),
		menor("l19", "2007-12-31", e.Nino, e.Riesgo))}
	// 5: uno en 0-5; 6 y 11: dos en 6-11; 12,17 y quien cumple
	// 18 el 31/12: tres en 12-17. Quien cumplió 18 en 2025 queda fuera.
	m := calcular(t, in)
	for _, caso := range []struct {
		tramo string
		n     int
	}{{"edad_0_5", 1}, {"edad_6_11", 2}, {"edad_12_17", 3}} {
		comprobarCelda(t, m.Apartados[e.MenoresPorEdad], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Tramo: caso.tramo}, e.Menores, caso.n, 0)
	}
	if m.Apartados[e.CasosAtendidos].Total.Menores != 6 {
		t.Fatal(m.Apartados[e.CasosAtendidos].Total)
	}
	// En junio, la edad del contrato sigue siendo la del 31/12, incluidas
	// las personas que todavía no han cumplido 18 en la fecha de cálculo.
	in.FechaCalculo = dia("2026-06-30")
	comprobarCelda(t, calcular(t, in).Apartados[e.MenoresPorEdad], e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Tramo: "edad_12_17"}, e.Menores, 3, 0)
}

func TestSeguimientoFueraDelAnioYSubprograma(t *testing.T) {
	in := entrada()
	exp := expediente("S", "2024-01-01", "2025-01-01", e.ObjetivosCumplidos, e.Reunificacion)
	exp.SeguimientosTrasCese = []e.Vigencia{{Desde: dia("2025-01-01"), Hasta: dia("2026-01-01")}}
	in.Expedientes = []e.Expediente{exp}
	// Hasta es exclusivo: terminó el 31/12/2025, no cruza 2026.
	if calcular(t, in).Apartados[e.CasosAtendidos].Total.Familias != 0 {
		t.Fatal("seguimiento terminado contado")
	}
	in.Expedientes[0].SeguimientosTrasCese[0].Hasta = dia("2026-01-02")
	if calcular(t, in).Apartados[e.CasosAtendidos].Total.Familias != 1 {
		t.Fatal("se omitió seguimiento activo el 01/01")
	}
	in.Expedientes[0].Subprograma = historia("S-riesgo", e.Riesgo)
	if _, err := e.Calcular(in); err != e.ErrEntrada {
		t.Fatal("seguimiento de Riesgo admitido", err)
	}
	in.Expedientes[0].Subprograma = nil
	m := calcular(t, in)
	if m.Apartados[e.CasosAtendidos].Total.Familias != 0 || m.Apartados[e.CasosAtendidos].Avisos["estadistica.seguimiento_subprograma_sin_dato"] != 1 {
		t.Fatal("seguimiento desconocido clasificado")
	}
}

func TestVigenciasAdyacentesYErroresDeHistoria(t *testing.T) {
	h := historia("v1", "biparental")
	h[0].Vigencia.Hasta = dia("2026-07-01")
	h = append(h, e.Revision[string]{ID: "v2", Vigencia: e.Vigencia{Desde: dia("2026-07-01")}, RegistradoEn: dia("2026-07-02"), Dato: conocido("monoparental_madre")})
	for _, caso := range []struct{ corte, esperado string }{{"2026-06-30", "biparental"}, {"2026-07-01", "monoparental_madre"}} {
		v, _, err := h.ValorEn(dia(caso.corte), dia("2027-01-01"))
		if err != nil || v.Valor != caso.esperado {
			t.Fatal(v, err)
		}
	}
	// La revisión de julio aún no estaba registrada el 01/07: sin dato,
	// sin acudir a la ficha actual ni inventar la vigencia anterior.
	v, id, err := h.ValorEn(dia("2026-07-01"), dia("2026-07-01"))
	if err != nil || v.Conocido || id != "" {
		t.Fatal(v, id, err)
	}
	h[0].Vigencia.Hasta = dia("2026-07-02")
	if _, _, err := h.ValorEn(dia("2026-07-01"), dia("2027-01-01")); err != e.ErrSolapamiento {
		t.Fatal(err)
	}
	h[0].Vigencia.Hasta = dia("2026-07-01")
	h[1].Sustituye = "inexistente"
	if _, _, err := h.ValorEn(dia("2026-07-01"), dia("2027-01-01")); err != e.ErrHistoria {
		t.Fatal(err)
	}
}

func TestEntradaIncoherenteSeRechaza(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		alterar  func(*e.Entrada)
		esperado error
	}{
		{"anio", func(in *e.Entrada) { in.Anio = 0 }, e.ErrEntrada},
		{"corte_anterior", func(in *e.Entrada) { in.FechaCalculo = dia("2025-12-31") }, e.ErrEntrada},
		{"reglas_ausentes", func(in *e.Entrada) { in.Reglas = nil }, e.ErrReglas},
		{"tramo_con_hueco", func(in *e.Entrada) { in.Reglas[0].Dato.Valor.TramosRiesgo[1].DesdeMeses = 7 }, e.ErrReglas},
		{"expediente_repetido", func(in *e.Entrada) { in.Expedientes = append(in.Expedientes, in.Expedientes[0]) }, e.ErrEntrada},
		{"menor_repetido", func(in *e.Entrada) {
			m := in.Expedientes[0].Menores[0].Dato.Valor[0]
			in.Expedientes[0].Menores[0].Dato.Valor = append(in.Expedientes[0].Menores[0].Dato.Valor, m)
		}, e.ErrEntrada},
		{"cierre_anterior", func(in *e.Entrada) { in.Expedientes[0].Episodios[0].Cierre = dia("2024-12-31") }, e.ErrEntrada},
		{"reapertura_ajena", func(in *e.Entrada) { in.Expedientes[0].Episodios[0].ReaperturaDe = "otro" }, e.ErrEntrada},
		{"nacimiento_futuro", func(in *e.Entrada) {
			in.FechaCalculo = dia("2026-06-30")
			in.Expedientes[0].Menores[0].Dato.Valor[0].Nacimiento = dia("2026-09-01")
		}, e.ErrEntrada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			in := entrada()
			in.Expedientes = []e.Expediente{expediente("V", "2025-01-01", "", "", e.Riesgo, menor("v1", "2015-01-01", e.Nino, e.Riesgo))}
			caso.alterar(&in)
			if _, err := e.Calcular(in); err != caso.esperado {
				t.Fatalf("%v; esperado %v", err, caso.esperado)
			}
		})
	}
}

func TestReaperturaElMismoDiaNoDuplicaFamilia(t *testing.T) {
	in := entrada()
	exp := expediente("R", "2025-01-01", "2026-02-01", e.ObjetivosCumplidos, e.Riesgo)
	exp.Episodios = append(exp.Episodios, e.Episodio{ID: "R-ep2", Apertura: dia("2026-02-01"), Aceptacion: dia("2025-01-01"), ReaperturaDe: "R-ep1"})
	in.Expedientes = []e.Expediente{exp}
	m := calcular(t, in)
	if m.Apartados[e.CasosAtendidos].Total.Familias != 1 || m.Apartados[e.CasosCerrados].Total.Familias != 1 {
		t.Fatal("reapertura del mismo día rechazada o duplicada")
	}
	// Un ciclo desconectado el mismo día no se confunde con una cadena válida.
	in.Expedientes[0].Episodios = append(in.Expedientes[0].Episodios,
		e.Episodio{ID: "c1", Apertura: dia("2026-03-01"), Cierre: dia("2026-03-01"), ReaperturaDe: "c2"},
		e.Episodio{ID: "c2", Apertura: dia("2026-03-01"), Cierre: dia("2026-03-01"), ReaperturaDe: "c1"})
	if _, err := e.Calcular(in); err != e.ErrEntrada {
		t.Fatal("ciclo admitido", err)
	}
}
