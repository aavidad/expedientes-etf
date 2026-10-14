package estadistica_test

import (
	"reflect"
	"testing"
	"time"

	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

func dia(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func conocido[T any](v T) e.Dato[T] { return e.Dato[T]{Valor: v, Conocido: true} }
func historia[T any](id string, v T) e.Historia[T] {
	return e.Historia[T]{{ID: id, Vigencia: e.Vigencia{Desde: dia("2020-01-01")}, RegistradoEn: dia("2026-01-01"), Dato: conocido(v)}}
}
func limite(v int) *int { return &v }
func entrada() e.Entrada {
	tramos := []e.Tramo{{Codigo: "d0", DesdeMeses: 0, HastaMeses: limite(6)}, {Codigo: "d6", DesdeMeses: 6, HastaMeses: limite(13)}, {Codigo: "d13", DesdeMeses: 13, HastaMeses: limite(19)}, {Codigo: "d19", DesdeMeses: 19}}
	return e.Entrada{Anio: 2026, FechaCalculo: dia("2026-12-31"), ConocidoEn: dia("2027-02-01"), Reglas: historia("regla-2026", e.Parametros{TramosRiesgo: tramos, TramosReunificacion: tramos})}
}
func menor(id, nacimiento string, sexo e.Sexo, s e.Subprograma) e.Menor {
	return e.Menor{ID: id, Nacimiento: dia(nacimiento), Sexo: sexo, Subprograma: s}
}
func expediente(id, aceptacion, cierre string, motivo e.Motivo, s e.Subprograma, menores ...e.Menor) e.Expediente {
	return e.Expediente{ID: id, EquipoID: "etf-01", Episodios: []e.Episodio{{ID: id + "-ep1", Apertura: dia(aceptacion), Aceptacion: dia(aceptacion), Cierre: dia(cierre), Motivo: motivo}}, Subprograma: historia(id+"-sub", s), Menores: historia(id+"-men", menores)}
}
func calcular(t *testing.T, in e.Entrada) e.Memoria {
	t.Helper()
	m, err := e.Calcular(in)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func comprobarCelda(t *testing.T, r *e.Resultado, g e.Grupo, u e.Unidad, n, sin int) {
	t.Helper()
	c := r.Celdas[e.ClaveCelda{Grupo: g, Unidad: u}]
	if c.Numerador != n || c.SinDato != sin {
		t.Fatalf("apartado %s grupo %+v unidad %s: %+v; esperado %d y sin dato %d", r.Contrato.Apartado, g, u, c, n, sin)
	}
}

func TestApartadosConRecuentosManuales(t *testing.T) {
	in := entrada()
	in.Expedientes = []e.Expediente{
		expediente("A", "2025-05-15", "2026-06-14", e.ObjetivosCumplidos, e.Riesgo,
			menor("a1", "2021-12-31", e.Nina, e.Riesgo), menor("a2", "2008-01-01", e.Nino, e.Riesgo)),
		expediente("B", "2026-01-01", "2026-07-01", e.ObjetivosParciales, e.Reunificacion,
			menor("b1", "2015-06-20", e.Nina, e.Reunificacion), menor("b2", "2010-07-07", e.Nino, e.Riesgo)),
		expediente("C", "2024-10-01", "2026-03-01", e.Traslado, e.Riesgo,
			menor("c1", "2018-03-01", e.Nino, e.Riesgo)),
		expediente("D", "2025-12-01", "", "", e.Reunificacion,
			menor("d1", "2007-12-31", e.Nina, e.Reunificacion), menor("d2", "", e.Nino, e.Reunificacion)),
		expediente("E", "2024-01-01", "2025-10-01", e.ObjetivosCumplidos, e.Reunificacion,
			menor("e1", "2020-02-01", e.Nina, e.Reunificacion)),
		{ID: "F", EquipoID: "etf-01"}, // no aceptado
		expediente("G", "2027-01-01", "", "", e.Riesgo, menor("g1", "2020-03-01", e.Nino, e.Riesgo)),
	}
	in.Expedientes[4].SeguimientosTrasCese = []e.Vigencia{{Desde: dia("2025-10-01"), Hasta: dia("2026-10-01")}}
	m := calcular(t, in)
	// A,B,C,D,E: 5 familias. Menores: 2+2+1+1+1=7. D1 cumplió
	// 18 en 2025 y se excluye; A2 cumple 18 en 2026 y sigue contando.
	// E sólo entra por seguimiento; F y G no entran.
	r := m.Apartados[e.CasosAtendidos]
	if r.Total != (e.Recuento{Familias: 5, Menores: 7}) {
		t.Fatalf("atendidos: %+v", r.Total)
	}
	for _, fila := range []struct {
		g                 e.Grupo
		familias, menores int
	}{
		{e.Grupo{Subprograma: e.Riesgo, Origen: "anterior"}, 2, 3},
		{e.Grupo{Subprograma: e.Reunificacion, Origen: "anterior"}, 2, 2},
		{e.Grupo{Subprograma: e.Riesgo, Origen: "incorporado"}, 0, 1},
		{e.Grupo{Subprograma: e.Reunificacion, Origen: "incorporado"}, 1, 1},
	} {
		comprobarCelda(t, r, fila.g, e.Familias, fila.familias, 0)
		comprobarCelda(t, r, fila.g, e.Menores, fila.menores, 0)
	}
	// B cuenta una sola familia de Reunificación; b2 usa Riesgo.
	// Riesgo: a2,b2,c1 niños y a1 niña. Reunificación: d2 niño; b1,e1 niñas.
	for _, fila := range []struct {
		s    e.Subprograma
		sexo e.Sexo
		n    int
	}{
		{e.Riesgo, e.Nino, 3}, {e.Riesgo, e.Nina, 1}, {e.Reunificacion, e.Nino, 1}, {e.Reunificacion, e.Nina, 2},
	} {
		comprobarCelda(t, m.Apartados[e.MenoresPorSexo], e.Grupo{Subprograma: fila.s, Sexo: fila.sexo}, e.Menores, fila.n, 0)
	}
	// CI=A+B+C: 3 familias, 5 menores. CO=A+B: 2 familias, 4 menores;
	// NC=C: una familia y un menor. A y C son anteriores, B incorporada.
	if m.Apartados[e.CasosCerrados].Total != (e.Recuento{Familias: 3, Menores: 5}) {
		t.Fatal(m.Apartados[e.CasosCerrados].Total)
	}
	comprobarCelda(t, m.Apartados[e.CasosCerrados], e.Grupo{Subprograma: e.Riesgo, Origen: "anterior"}, e.Familias, 2, 0)
	comprobarCelda(t, m.Apartados[e.CasosCerrados], e.Grupo{Subprograma: e.Reunificacion, Origen: "incorporado"}, e.Familias, 1, 0)
	if m.Apartados[e.TratamientoConcluido].Total != (e.Recuento{Familias: 2, Menores: 4}) {
		t.Fatal(m.Apartados[e.TratamientoConcluido].Total)
	}
	comprobarCelda(t, m.Apartados[e.TratamientoConcluido], e.Grupo{Subprograma: e.Riesgo, Motivo: e.ObjetivosCumplidos}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.TratamientoConcluido], e.Grupo{Subprograma: e.Reunificacion, Motivo: e.ObjetivosParciales}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.TratamientoConcluido], e.Grupo{Subprograma: e.Riesgo, Motivo: e.ObjetivosParciales}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.TratamientoInterrumpido], e.Grupo{Subprograma: e.Riesgo, Motivo: e.Traslado}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.TratamientoInterrumpido], e.Grupo{Subprograma: e.Riesgo, Motivo: e.Traslado}, e.Menores, 1, 0)
	if m.Apartados[e.CasosCerrados].Total.Familias != m.Apartados[e.TratamientoConcluido].Total.Familias+m.Apartados[e.TratamientoInterrumpido].Total.Familias {
		t.Fatal("7 debe coincidir con 8+9 con motivos informados")
	}
	if m.Apartados[e.CasosCerrados].Total.Menores != m.Apartados[e.TratamientoConcluido].Total.Menores+m.Apartados[e.TratamientoInterrumpido].Total.Menores {
		t.Fatal("7 debe coincidir con 8+9 también en menores")
	}
	// A: 15/05/2025 a 14/06/2026 son 12 meses completos, no 13.
	// B: 01/01 a 01/07 son 6. C no entra en duración (no concluyó).
	comprobarCelda(t, m.Apartados[e.DuracionRiesgo], e.Grupo{Subprograma: e.Riesgo, Tramo: "d6"}, e.Familias, 1, 0)
	comprobarCelda(t, m.Apartados[e.DuracionReunificacion], e.Grupo{Subprograma: e.Reunificacion, Tramo: "d6"}, e.Familias, 1, 0)
	// Edad: a1=5, c1=8, a2=18 (nota43), b2=16; b1=11 y e1=6.
	// d2 mantiene su sexo y subprograma, pero declara una edad sin dato.
	for _, fila := range []struct {
		g      e.Grupo
		n, sin int
	}{
		{e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Tramo: "edad_0_5"}, 1, 0},
		{e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Tramo: "edad_6_11"}, 1, 0},
		{e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Tramo: "edad_12_17"}, 2, 0},
		{e.Grupo{Subprograma: e.Reunificacion, Sexo: e.Nina, Tramo: "edad_6_11"}, 2, 0},
		{e.Grupo{Subprograma: e.Reunificacion, Sexo: e.Nino}, 0, 1},
	} {
		comprobarCelda(t, m.Apartados[e.MenoresPorEdad], fila.g, e.Menores, fila.n, fila.sin)
	}
	// El total de ME de 1,2,25 es el mismo; la edad desconocida no altera el sexo.
	if m.Apartados[e.MenoresPorSexo].Total.Menores != 7 || m.Apartados[e.MenoresPorEdad].Total.Menores != 7 {
		t.Fatal("poblaciones ME diferentes")
	}
	for _, r := range m.Apartados {
		if r.Contrato.VersionRegla != "regla-2026" || r.Contrato.Denominador == "" {
			t.Fatal(r.Contrato)
		}
		for _, c := range r.Celdas {
			if (r.Contrato.Denominador == "pendiente_programa" && c.Denominador.Conocido) || c.VersionRegla != "regla-2026" {
				t.Fatal(c)
			}
		}
	}
}

func TestCierreUsaSuVigenciaYReaperturaNoLoBorra(t *testing.T) {
	in := entrada()
	exp := expediente("H", "2025-01-10", "2026-02-10", e.ObjetivosCumplidos, e.Riesgo,
		menor("h1", "2019-01-01", e.Nino, e.Riesgo))
	exp.Episodios = append(exp.Episodios, e.Episodio{ID: "H-ep2", Apertura: dia("2026-05-01"), Aceptacion: dia("2025-01-10"), ReaperturaDe: "H-ep1"})
	exp.Subprograma[0].Vigencia.Hasta = dia("2026-04-01")
	exp.Subprograma = append(exp.Subprograma, e.Revision[e.Subprograma]{ID: "H-sub2", Vigencia: e.Vigencia{Desde: dia("2026-04-01")}, RegistradoEn: dia("2026-04-01"), Dato: conocido(e.Reunificacion)})
	exp.Menores[0].Vigencia.Hasta = dia("2026-04-01")
	exp.Menores = append(exp.Menores, e.Revision[[]e.Menor]{ID: "H-men2", Vigencia: e.Vigencia{Desde: dia("2026-04-01")}, RegistradoEn: dia("2026-04-01"), Dato: conocido([]e.Menor{menor("h2", "2022-01-01", e.Nina, e.Reunificacion), menor("h3", "2020-01-01", e.Nino, e.Reunificacion)})})
	in.Expedientes = []e.Expediente{exp}
	m := calcular(t, in)
	// Una familia AT aunque haya dos episodios. Al corte: 2 menores de
	// Reunificación. En el cierre: 1 menor de Riesgo, duración 13 meses.
	if m.Apartados[e.CasosAtendidos].Total != (e.Recuento{Familias: 1, Menores: 2}) {
		t.Fatal(m.Apartados[e.CasosAtendidos].Total)
	}
	comprobarCelda(t, m.Apartados[e.CasosCerrados], e.Grupo{Subprograma: e.Riesgo, Origen: "anterior"}, e.Menores, 1, 0)
	comprobarCelda(t, m.Apartados[e.DuracionRiesgo], e.Grupo{Subprograma: e.Riesgo, Tramo: "d13"}, e.Familias, 1, 0)
	// 30/06: la reapertura ya cuenta, pero un cierre futuro no cuenta.
	in.FechaCalculo = dia("2026-06-30")
	in.Expedientes[0].Episodios[1].Cierre = dia("2026-08-01")
	in.Expedientes[0].Episodios[1].Motivo = e.Traslado
	m = calcular(t, in)
	if m.Apartados[e.CasosCerrados].Total.Familias != 1 || m.Apartados[e.TratamientoConcluido].Total.Familias != 1 {
		t.Fatal("se contó el cierre futuro")
	}
	// Al acabar el año hay dos cierres diferentes: CI cuenta 1 familia,
	// CO/NC y duración no eligen un episodio sin decisión del Programa.
	in.FechaCalculo = dia("2026-12-31")
	m = calcular(t, in)
	if m.Apartados[e.CasosCerrados].Total.Familias != 1 || m.Apartados[e.TratamientoConcluido].Total.Familias != 0 || m.Apartados[e.TratamientoInterrumpido].Total.Familias != 0 {
		t.Fatal("doble cierre clasificado arbitrariamente")
	}
	if m.Apartados[e.DuracionRiesgo].Avisos["estadistica.varios_cierres_pendiente"] != 1 {
		t.Fatal("falta aviso de criterio pendiente")
	}
}

func TestDesconocidosNoSeAsignanACategorias(t *testing.T) {
	in := entrada()
	h := expediente("I", "2026-01-01", "2026-04-01", "", e.Riesgo)
	h.Subprograma, h.Menores = nil, nil
	h.Episodios[0].Aceptacion = time.Time{}
	in.Expedientes = []e.Expediente{h}
	m := calcular(t, in)
	comprobarCelda(t, m.Apartados[e.CasosAtendidos], e.Grupo{}, e.Familias, 0, 1)
	if m.Apartados[e.CasosAtendidos].Total.FamiliasSinRelacionMenores != 1 {
		t.Fatal("cardinalidad desconocida convertida a cero")
	}
	if m.Apartados[e.TratamientoConcluido].Total.Familias != 0 || m.Apartados[e.TratamientoInterrumpido].Total.Familias != 0 || m.Apartados[e.CasosCerrados].Avisos["estadistica.motivo_sin_dato"] != 1 {
		t.Fatal("motivo desconocido repartido")
	}
	// Ya consta que concluyó, pero no la aceptación; dura sin dato.
	in.Expedientes[0].Subprograma = historia("I-sub", e.Reunificacion)
	in.Expedientes[0].Episodios[0].Motivo = e.ObjetivosParciales
	in.Expedientes[0].Menores = historia("I-men", []e.Menor{menor("i1", "2015-03-10", "", "")})
	m = calcular(t, in)
	comprobarCelda(t, m.Apartados[e.DuracionReunificacion], e.Grupo{Subprograma: e.Reunificacion}, e.Familias, 0, 1)
	comprobarCelda(t, m.Apartados[e.MenoresPorSexo], e.Grupo{}, e.Menores, 0, 1)
	comprobarCelda(t, m.Apartados[e.MenoresPorEdad], e.Grupo{Tramo: "edad_6_11"}, e.Menores, 0, 1)
	// Una relación informada sin menores sí es cero, no cardinalidad desconocida.
	in.Expedientes[0].Menores = historia("I-men-vacia", []e.Menor{})
	if calcular(t, in).Apartados[e.CasosAtendidos].Total.FamiliasSinRelacionMenores != 0 {
		t.Fatal("lista vacía confundida con sin dato")
	}
}

func TestReglasVersionadasYTramosConfigurables(t *testing.T) {
	in := entrada()
	in.Expedientes = []e.Expediente{expediente("J", "2025-05-15", "2026-06-14", e.ObjetivosCumplidos, e.Riesgo)}
	in.Reglas[0].Vigencia.Hasta = dia("2026-12-01")
	in.Reglas = append(in.Reglas, e.Revision[e.Parametros]{ID: "regla-diciembre", Vigencia: e.Vigencia{Desde: dia("2026-12-01"), Hasta: dia("2027-01-01")}, RegistradoEn: dia("2026-12-01"), Dato: conocido(e.Parametros{TramosRiesgo: []e.Tramo{{Codigo: "corto", HastaMeses: limite(12)}, {Codigo: "largo", DesdeMeses: 12}}, TramosReunificacion: []e.Tramo{{Codigo: "todos"}}})})
	in.Reglas = append(in.Reglas, e.Revision[e.Parametros]{ID: "regla-2027", Vigencia: e.Vigencia{Desde: dia("2027-01-01")}, RegistradoEn: dia("2027-01-01"), Dato: conocido(e.Parametros{TramosRiesgo: []e.Tramo{{Codigo: "otro"}}, TramosReunificacion: []e.Tramo{{Codigo: "otro"}}})})
	m := calcular(t, in)
	// Son 12 meses: con esta configuración entran en largo. La versión
	// seleccionada es 31/12/2026, aunque el cierre fuera en junio.
	comprobarCelda(t, m.Apartados[e.DuracionRiesgo], e.Grupo{Subprograma: e.Riesgo, Tramo: "largo"}, e.Familias, 1, 0)
	if m.Apartados[e.DuracionRiesgo].Contrato.VersionRegla != "regla-diciembre" {
		t.Fatal("regla de otro año")
	}
}

func TestCambioRetroactivoDeComposicionConservaRevision(t *testing.T) {
	h := historia("comp1", "biparental")
	h = append(h, e.Revision[string]{ID: "comp2", Sustituye: "comp1", Vigencia: h[0].Vigencia, RegistradoEn: dia("2027-01-20"), Dato: conocido("monoparental_madre")})
	original := append(e.Historia[string](nil), h...)
	// Corrección registrada en enero de 2027 con efecto de negocio desde
	// 2020. Una foto conocida el 10/01 sigue usando comp1; al 01/02 usa
	// comp2 para el mismo corte 31/12/2026. No se destruye ni cambia comp1.
	for _, caso := range []struct{ conocido, valor, version string }{
		{"2027-01-10", "biparental", "comp1"}, {"2027-02-01", "monoparental_madre", "comp2"},
	} {
		v, id, err := h.ValorEn(dia("2026-12-31"), dia(caso.conocido))
		if err != nil || !v.Conocido || v.Valor != caso.valor || id != caso.version {
			t.Fatalf("valor %+v versión %s error %v", v, id, err)
		}
	}
	if !reflect.DeepEqual(h, original) {
		t.Fatal("se modificó la historia")
	}
	// El modelo recibe la composición; F0.16 no adelanta el cálculo del 17.
	in := entrada()
	exp := expediente("K", "2025-01-01", "", "", e.Riesgo)
	exp.Composicion = h
	in.Expedientes = []e.Expediente{exp}
	if calcular(t, in).Apartados[e.CasosAtendidos].Total.Familias != 1 {
		t.Fatal("retroactividad altera población")
	}
}
