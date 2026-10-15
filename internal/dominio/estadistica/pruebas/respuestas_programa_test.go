package estadistica_test

import (
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"testing"
)

func TestD2cEsperaCuentaEnApartado6SinAceptacionNiNumero(t *testing.T) {
	espera := e.Expediente{ID: "espera", Estados: historia("espera-estado", "lista_espera"), Subprograma: historia("espera-sub", e.Riesgo)}
	recibida := e.Expediente{ID: "recibida", Estados: historia("recibida-estado", "derivacion_recibida")}
	noAceptada := e.Expediente{ID: "no-aceptada", Estados: historia("rechazo-estado", "no_aceptada")}
	in := entrada()
	in.Expedientes = []e.Expediente{espera, recibida, noAceptada}
	m := calcular(t, in)
	if m.Apartados[e.Pendientes].Total.Familias != 1 || m.Apartados[e.CasosAtendidos].Total.Familias != 0 {
		t.Fatalf("espera no es caso atendido: pendientes=%+v atendidos=%+v", m.Apartados[e.Pendientes].Total, m.Apartados[e.CasosAtendidos].Total)
	}
	comprobarCelda(t, m.Apartados[e.Pendientes], e.Grupo{Subprograma: e.Riesgo}, e.Familias, 1, 0)
	if m.Apartados[e.Pendientes].Contrato.Poblacion != "PE" {
		t.Fatal("población excluye la espera")
	}
}

func TestD2cEsperaRespetaCorteConocimientoYSaleAlAceptar(t *testing.T) {
	in := entrada()
	x := expediente("salida", "2026-12-20", "", "", e.Riesgo)
	x.Consentimiento = historia("cons-salida", false)
	x.Estados = e.Historia[string]{
		{ID: "recibida", Vigencia: e.Vigencia{Desde: dia("2026-01-01"), Hasta: dia("2026-12-01")}, RegistradoEn: dia("2026-01-01"), Dato: conocido("derivacion_recibida")},
		{ID: "espera", Vigencia: e.Vigencia{Desde: dia("2026-12-01"), Hasta: dia("2026-12-20")}, RegistradoEn: dia("2026-12-01"), Dato: conocido("lista_espera")},
		{ID: "aceptada", Vigencia: e.Vigencia{Desde: dia("2026-12-20")}, RegistradoEn: dia("2026-12-20"), Dato: conocido("aceptada")},
	}
	in.Expedientes = []e.Expediente{x}
	for _, caso := range []struct {
		corte      string
		pendientes int
	}{{"2026-11-30", 0}, {"2026-12-15", 1}, {"2026-12-31", 0}} {
		in.FechaCalculo = dia(caso.corte)
		if got := calcular(t, in).Apartados[e.Pendientes].Total.Familias; got != caso.pendientes {
			t.Fatalf("corte %s: %d, esperado %d", caso.corte, got, caso.pendientes)
		}
	}
	in.FechaCalculo = dia("2026-12-15")
	in.ConocidoEn = dia("2026-11-30")
	if got := calcular(t, in).Apartados[e.Pendientes].Total.Familias; got != 0 {
		t.Fatalf("espera aún no conocida: %d", got)
	}
}

func TestD7AbuelosParejasYTutoresCuentanComoFigurasParentales(t *testing.T) {
	in := entrada()
	x := expediente("parentales", "2026-01-01", "", "", e.Riesgo)
	x.Adultos = historia("adultos", []e.Adulto{
		{Papel: "abuelo", Sexo: e.Nino, Nacimiento: dia("1960-01-01"), NivelEducativo: "primaria", SituacionLaboral: "jubilado"},
		{Papel: "pareja_madre", Sexo: e.Nina, Nacimiento: dia("1980-01-01"), NivelEducativo: "secundaria", SituacionLaboral: "empleo"},
		{Papel: "tutor", Sexo: e.Nino, Nacimiento: dia("1985-01-01"), NivelEducativo: "superior", SituacionLaboral: "empleo"},
	})
	in.Expedientes = []e.Expediente{x}
	m := calcular(t, in)
	for _, a := range []e.Apartado{e.EdadParental, e.NivelEducativoParental, e.SituacionLaboralParental} {
		r := m.Apartados[a]
		if r.Total.Adultos != 3 {
			t.Fatalf("apartado %s: %+v", a, r.Total)
		}
		porSexo := map[e.Sexo]int{}
		for k, c := range r.Celdas {
			porSexo[k.Grupo.Sexo] += c.Numerador
		}
		if porSexo[e.Nino] != 2 || porSexo[e.Nina] != 1 {
			t.Fatalf("apartado %s debe contar dos padres y una madre: %+v", a, porSexo)
		}
	}
}
