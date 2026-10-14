package estadistica_test

import (
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"reflect"
	"testing"
	"time"
)

func fechaSPAF(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func historiaSPAF[T any](id, desde string, valor T) d.Historia[T] {
	return d.Historia[T]{{ID: id, Vigencia: d.Vigencia{Desde: fechaSPAF(desde)}, RegistradoEn: fechaSPAF("2025-01-01"), Dato: d.Dato[T]{Valor: valor, Conocido: true}}}
}

func entradaSPAF() d.EntradaSPAF {
	return d.EntradaSPAF{Filtro: d.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: "equipo-2"}, FechaCalculo: fechaSPAF("2026-03-01"), ConocidoEn: fechaSPAF("2026-03-01"), Reglas: historiaSPAF("reglas-enero", "2025-01-01", d.ReglasSPAF{MotivosCierre: map[d.Motivo]string{d.Traslado: "traslado_domicilio", d.ObjetivosCumplidos: "consecucion_objetivos"}})}
}

func expedienteSPAF(id, aceptacion, cierre string, menores int) d.ExpedienteSPAF {
	lista := []d.MenorSPAF{}
	for i := 0; i < menores; i++ {
		lista = append(lista, d.MenorSPAF{ID: string(rune('a' + i)), Nombre: "Lucía", Apellidos: "Serrano Ruiz", Nacimiento: d.Dato[time.Time]{Valor: fechaSPAF("2015-06-10"), Conocido: true}})
	}
	return d.ExpedienteSPAF{ID: id, EquipoID: "equipo-2", Numero: id, Episodios: []d.Episodio{{ID: "episodio-1", Apertura: fechaSPAF(aceptacion), Aceptacion: fechaSPAF(aceptacion), Cierre: fechaSPAF(cierre)}}, Ficha: historiaSPAF("ficha-1", "2025-01-01", d.DatosSPAF{Menores: d.Dato[[]d.MenorSPAF]{Valor: lista, Conocido: true}})}
}

func TestSPAFRecuentosManualesYLimitesDelMes(t *testing.T) {
	// Enero: dos aceptaciones, dos cierres, tres filas, seis menores.
	// El cierre de diciembre y la aceptación de febrero no entran.
	e := entradaSPAF()
	e.Expedientes = []d.ExpedienteSPAF{
		expedienteSPAF("2-001-2025", "2025-11-10", "2025-12-31", 4),
		expedienteSPAF("2-002-2025", "2025-11-12", "2026-01-01", 1),
		expedienteSPAF("2-001-2026", "2026-01-01", "2026-01-31", 2),
		expedienteSPAF("2-002-2026", "2026-01-31", "", 3),
		expedienteSPAF("2-003-2026", "2026-02-01", "", 5),
	}
	got, err := d.CalcularSPAF(e)
	want := d.ResumenSPAF{Altas: 2, Bajas: 2, Casos: 3, Menores: 6}
	if err != nil || got.Resumen != want {
		t.Fatalf("resumen=%+v error=%v; esperado=%+v", got.Resumen, err, want)
	}
	if got.VersionRegla != "reglas-enero" || !got.Corte.Equal(fechaSPAF("2026-01-31")) {
		t.Fatal(got)
	}
	for _, f := range got.Filas {
		if f.ExpedienteID == "2-001-2026" && (!f.Alta || !f.Baja || len(f.Cierres) != 1) {
			t.Fatal("alta y baja del mismo mes perdidas", f)
		}
	}
}

func TestSPAFMesSinDatosYDivisorCero(t *testing.T) {
	got, err := d.CalcularSPAF(entradaSPAF())
	if err != nil || got.Resumen != (d.ResumenSPAF{}) || got.Filas == nil || len(got.Filas) != 0 {
		t.Fatal(got, err)
	}
	// SPAF no inventa porcentajes. El contrato estadístico común garantiza que
	// una presentación que los solicite obtenga cero con un aviso, nunca NaN.
	p, err := d.CalcularPorcentaje(0, d.Dato[int]{Valor: 0, Conocido: true})
	if err != nil || p.Centesimas != 0 || p.Aviso != "estadistica.divisor_cero" {
		t.Fatal(p, err)
	}
}

func TestSPAFTrasladoConservaBajaOrigenYAltaDestino(t *testing.T) {
	origen := expedienteSPAF("2-021-2025", "2025-04-01", "2026-01-15", 2)
	origen.Episodios[0].Motivo = d.Traslado
	destino := expedienteSPAF("6-001-2026", "2026-01-15", "", 2)
	destino.EquipoID = "equipo-6"
	e := entradaSPAF()
	e.Expedientes = []d.ExpedienteSPAF{origen}
	a, err := d.CalcularSPAF(e)
	if err != nil || a.Resumen != (d.ResumenSPAF{Bajas: 1, Casos: 1, Menores: 2}) || a.Filas[0].Cierres[0].Motivo.Valor != "traslado_domicilio" {
		t.Fatal(a, err)
	}
	e.Filtro.EquipoID = "equipo-6"
	e.Expedientes = []d.ExpedienteSPAF{destino}
	b, err := d.CalcularSPAF(e)
	if err != nil || b.Resumen != (d.ResumenSPAF{Altas: 1, Casos: 1, Menores: 2}) {
		t.Fatal(b, err)
	}
}

func TestSPAFReaperturaNoDuplicaCasoNiBorraCierres(t *testing.T) {
	e := entradaSPAF()
	x := expedienteSPAF("2-031-2025", "2025-07-01", "2026-01-10", 2)
	x.Episodios = append(x.Episodios, d.Episodio{ID: "episodio-2", ReaperturaDe: "episodio-1", Aceptacion: fechaSPAF("2025-07-01"), Apertura: fechaSPAF("2026-01-12"), Cierre: fechaSPAF("2026-01-20")}, d.Episodio{ID: "episodio-3", ReaperturaDe: "episodio-2", Aceptacion: fechaSPAF("2025-07-01"), Apertura: fechaSPAF("2026-01-22")})
	e.Expedientes = []d.ExpedienteSPAF{x}
	got, err := d.CalcularSPAF(e)
	if err != nil || got.Resumen != (d.ResumenSPAF{Bajas: 1, Casos: 1, Menores: 2}) || len(got.Filas[0].Cierres) != 2 {
		t.Fatal(got, err)
	}
	x.Episodios = []d.Episodio{{ID: "episodio-1", Aceptacion: fechaSPAF("2025-07-01"), Apertura: fechaSPAF("2025-07-01"), Cierre: fechaSPAF("2025-12-31")}, {ID: "episodio-2", ReaperturaDe: "episodio-1", Aceptacion: fechaSPAF("2025-07-01"), Apertura: fechaSPAF("2026-02-01")}}
	e.Expedientes = []d.ExpedienteSPAF{x}
	got, err = d.CalcularSPAF(e)
	if err != nil || got.Resumen.Casos != 0 {
		t.Fatal("reapertura posterior al mes incluida", got, err)
	}
}

func TestSPAFHistoriasReglasAnexosYDatosDesconocidos(t *testing.T) {
	e := entradaSPAF()
	x := expedienteSPAF("2-011-2025", "2025-04-01", "", 2)
	x.Ficha[0].Vigencia.Hasta = fechaSPAF("2026-02-01")
	x.Ficha[0].Dato.Valor.Derivaciones = []d.AnexoSPAF{{Anexo: 3, Fecha: fechaSPAF("2025-04-01")}, {Anexo: 5, Fecha: fechaSPAF("2026-01-20")}, {Anexo: 6, Fecha: fechaSPAF("2026-02-02")}}
	x.Ficha[0].Dato.Valor.FechasAnexos = []d.AnexoSPAF{{Anexo: 1, Fecha: fechaSPAF("2026-01-05")}, {Anexo: 7, Fecha: fechaSPAF("2026-02-03")}}
	x.Ficha = append(x.Ficha, d.Revision[d.DatosSPAF]{ID: "ficha-febrero", Vigencia: d.Vigencia{Desde: fechaSPAF("2026-02-01")}, RegistradoEn: fechaSPAF("2026-02-01"), Dato: d.Dato[d.DatosSPAF]{Conocido: true}})
	e.Reglas[0].Vigencia.Hasta = fechaSPAF("2026-02-01")
	e.Reglas = append(e.Reglas, d.Revision[d.ReglasSPAF]{ID: "reglas-febrero", Vigencia: d.Vigencia{Desde: fechaSPAF("2026-02-01")}, RegistradoEn: fechaSPAF("2026-02-01"), Dato: d.Dato[d.ReglasSPAF]{Conocido: true}})
	e.Expedientes = []d.ExpedienteSPAF{x}
	antes := x.Ficha[0].Dato.Valor
	got, err := d.CalcularSPAF(e)
	if err != nil || got.Resumen.Menores != 2 || got.VersionRegla != "reglas-enero" || got.Filas[0].VersionFicha != "ficha-1" || got.Filas[0].Derivacion.Valor.Anexo != 5 || len(got.Filas[0].FechasAnexos) != 1 || got.Avisos["estadistica.spaf_zts_pendiente"] != 1 {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(antes, x.Ficha[0].Dato.Valor) {
		t.Fatal("mutación de historia")
	}
	x.Ficha = nil
	e.Expedientes = []d.ExpedienteSPAF{x}
	got, err = d.CalcularSPAF(e)
	if err != nil || got.Resumen.CasosSinDatoMenores != 1 || got.Filas[0].NumeroMenores.Conocido {
		t.Fatal("sin dato convertido a cero conocido", got, err)
	}
}

func TestSPAFRechazaDatosIncoherentesYMesNoVencido(t *testing.T) {
	for _, caso := range []string{"mes actual", "mes inválido", "equipo ajeno", "duplicado", "reglas ausentes", "historia solapada"} {
		t.Run(caso, func(t *testing.T) {
			e := entradaSPAF()
			x := expedienteSPAF("2-015-2025", "2025-04-01", "", 1)
			e.Expedientes = []d.ExpedienteSPAF{x}
			switch caso {
			case "mes actual":
				e.FechaCalculo = fechaSPAF("2026-01-31")
				e.ConocidoEn = e.FechaCalculo
			case "mes inválido":
				e.Filtro.Mes = 13
			case "equipo ajeno":
				e.Filtro.EquipoID = "equipo-6"
			case "duplicado":
				e.Expedientes = append(e.Expedientes, x)
			case "reglas ausentes":
				e.Reglas = nil
			case "historia solapada":
				e.Expedientes[0].Ficha = append(e.Expedientes[0].Ficha, historiaSPAF("otra", "2025-01-01", d.DatosSPAF{})...)
			}
			_, err := d.CalcularSPAF(e)
			if err == nil {
				t.Fatal("entrada incoherente aceptada")
			}
		})
	}
	_, err := d.CalcularPorcentaje(-1, d.Dato[int]{Conocido: true})
	if !errors.Is(err, d.ErrEntrada) {
		t.Fatal(err)
	}
}
