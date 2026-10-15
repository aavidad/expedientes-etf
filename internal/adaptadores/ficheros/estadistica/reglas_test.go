package estadistica_test

import (
	"context"
	f "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	c "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"testing"
	"time"
)

type smiPorAnio []c.SMI

func (s smiPorAnio) SMI(context.Context) ([]c.SMI, error) { return s, nil }

func TestD8SMIDelAnioDeMemoriaYTramosDeLaJunta(t *testing.T) {
	anual2026, anual2027 := int64(1709400), int64(1820000)
	lector := f.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: smiPorAnio{{Anio: 2026, ImporteAnualCentimos: &anual2026, Version: 1}, {Anio: 2027, ImporteAnualCentimos: &anual2027, Version: 2}}}
	conocido := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		anio  int
		anual int64
	}{{2026, 1709400}, {2027, 1820000}} {
		h, err := lector.ReglasMemoria(t.Context(), caso.anio, conocido)
		if err != nil {
			t.Fatal(err)
		}
		corte := time.Date(caso.anio, 12, 31, 0, 0, 0, 0, time.UTC)
		p, _, err := h.ValorEn(corte, conocido)
		if err != nil || !p.Conocido || p.Valor.SMIAnualCentimos != caso.anual {
			t.Fatalf("SMI %d: %+v %v", caso.anio, p, err)
		}
		if len(p.Valor.TramosRiesgo) != 4 || len(p.Valor.TramosReunificacion) != 4 {
			t.Fatal("faltan tramos de la Junta")
		}
		for _, tramos := range [][]int{{p.Valor.TramosRiesgo[0].DesdeMeses, p.Valor.TramosRiesgo[1].DesdeMeses, p.Valor.TramosRiesgo[2].DesdeMeses, p.Valor.TramosRiesgo[3].DesdeMeses}, {p.Valor.TramosReunificacion[0].DesdeMeses, p.Valor.TramosReunificacion[1].DesdeMeses, p.Valor.TramosReunificacion[2].DesdeMeses, p.Valor.TramosReunificacion[3].DesdeMeses}} {
			for i, want := range []int{0, 6, 13, 19} {
				if tramos[i] != want {
					t.Fatal("tramos 13.1/13.2", tramos)
				}
			}
		}
		aceptacion := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		vigencia := e.Vigencia{Desde: aceptacion}
		x := e.Expediente{ID: "ingresos", EquipoID: "equipo", Episodios: []e.Episodio{{ID: "episodio", Apertura: aceptacion, Aceptacion: aceptacion}},
			Subprograma: e.Historia[e.Subprograma]{{ID: "sub", Vigencia: vigencia, RegistradoEn: aceptacion, Dato: e.Dato[e.Subprograma]{Conocido: true, Valor: e.Riesgo}}},
			Ingresos:    e.Historia[e.Ingresos]{{ID: "importe", Vigencia: vigencia, RegistradoEn: aceptacion, Dato: e.Dato[e.Ingresos]{Conocido: true, Valor: e.Ingresos{NetosAnualesCentimos: e.Dato[int64]{Conocido: true, Valor: 1750000}}}}},
		}
		m, err := e.Calcular(e.Entrada{Anio: caso.anio, FechaCalculo: corte, ConocidoEn: conocido, Reglas: h, Expedientes: []e.Expediente{x}})
		if err != nil {
			t.Fatal(err)
		}
		tramo := "DE_100_A_150"
		if caso.anio == 2027 {
			tramo = "DE_50_A_100"
		}
		celda := m.Apartados[e.IngresosSMI].Celdas[e.ClaveCelda{Grupo: e.Grupo{Subprograma: e.Riesgo, Tramo: tramo}, Unidad: e.Familias}]
		if celda.Numerador != 1 {
			t.Fatalf("importe 17.500 y SMI %d debe dar %s: %+v", caso.anio, tramo, celda)
		}

	}
}
