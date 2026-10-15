package estadistica_test

import (
	"encoding/json"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"reflect"
	"testing"
)

func TestDesgloseSumaCeldasTotalesYBasesSinDuplicarFamilias(t *testing.T) {
	in := entrada()
	in.Reglas[0].Dato.Valor.FactoresFamilia = []string{"ingresos_insuficientes", "conflicto_familiar"}
	in.Reglas[0].Dato.Valor.FactoresMenores = []string{"absentismo"}
	in.Reglas[0].Dato.Valor.TipologiasDesproteccion = []string{"negligencia"}
	in.Reglas[0].Dato.Valor.RecursosCoordinacion = []string{"spm"}
	in.Reglas[0].Dato.Valor.ResponsablesDesproteccion = []string{"padre"}
	a := expediente("A", "2025-01-01", "2026-05-01", e.ObjetivosCumplidos, e.Riesgo, menor("m1", "2008-04-01", e.Nina, e.Riesgo), menor("m2", "2012-01-01", e.Nina, e.Riesgo))
	a.Numero = "02-01-25"
	a.Adultos = historia("adultos", []e.Adulto{{Sexo: e.Nino, Papel: "padre", Nacimiento: dia("1980-01-01"), NivelEducativo: "primaria", SituacionLaboral: "empleado"}})
	a.Factores = historia("factores", []e.Factor{{Codigo: "ingresos_insuficientes", Presente: conocido(true)}, {Codigo: "conflicto_familiar", Presente: conocido(true)}})
	a.Propuestas = historia("propuestas", e.Propuestas{DeclaracionRiesgo: conocido(true), MedidaProtectora: conocido(true), PIFDerivadoDSR: conocido(true), RespuestaSPM: conocido(e.RespuestaEnEspera)})
	b := expediente("B", "2026-01-01", "2026-06-01", e.Rechazo, e.Reunificacion, menor("m3", "2018-01-01", e.Nino, e.Reunificacion))
	b.SeguimientosTrasCese = []e.Vigencia{{Desde: dia("2026-06-01")}}
	c := expediente("C", "2026-01-01", "", "", e.Riesgo)
	c.Menores = nil
	c.Consentimiento = historia("cons", false)
	in.Expedientes = []e.Expediente{a, b, c, {ID: "D", EquipoID: "etf-01", NoAceptaciones: []e.NoAceptacion{{Fecha: dia("2026-02-01"), ServicioDerivante: "sscc", Causa: "criterios", Subprograma: e.Riesgo}}}}
	in.Actividad = &e.RecuentoActividad{Anio: 2026, Comisiones: 2, Informes: 3, PorOrganismo: map[string]int64{"spm": 3}}
	m := calcular(t, in)
	aportes, err := e.AportesMemoria(in)
	if err != nil {
		t.Fatal(err)
	}
	informe := e.InformeMemoria{FiltroMemoria: e.FiltroMemoria{Anio: 2026}, VersionDesglose: 1, Desglose: aportes}
	for codigo, ap := range m.Apartados {
		apartado := e.ApartadoMemoria{Codigo: codigo, Total: ap.Total}
		for k, c := range ap.Celdas {
			apartado.Celdas = append(apartado.Celdas, e.CeldaMemoria{Grupo: k.Grupo, Unidad: k.Unidad, Celda: c})
		}
		informe.Apartados = append(informe.Apartados, apartado)
	}
	comprobaciones := 0
	for _, ap := range informe.Apartados {
		selecciones := []e.SeleccionDesglose{}
		for _, medida := range []string{"familias", "menores", "adultos", "comisiones", "informes", "familias_sin_relacion_menores"} {
			selecciones = append(selecciones, e.SeleccionDesglose{Celda: -1, Medida: medida})
		}
		for i, c := range ap.Celdas {
			for _, medida := range []string{"numerador", "sin_dato"} {
				selecciones = append(selecciones, e.SeleccionDesglose{Celda: i, Medida: medida})
			}
			if c.Celda.Denominador.Conocido {
				selecciones = append(selecciones, e.SeleccionDesglose{Celda: i, Medida: "denominador"})
			}
		}
		for _, s := range selecciones {
			r, err := e.SeleccionarDesglose(informe, ap.Codigo, s)
			if err != nil {
				t.Fatalf("%s %+v: %v", ap.Codigo, s, err)
			}
			if ap.Codigo == e.ComisionesTecnicas || ap.Codigo == e.InformesTecnicos {
				if r.Aviso != "memoria.desglose.actividad" {
					t.Fatal(r)
				}
				continue
			}
			if r.Aviso != "" {
				t.Fatalf("desglose no reproduce apartado %s selección %+v: %+v", ap.Codigo, s, r)
			}
			suma := 0
			ids := map[string]bool{}
			for _, f := range r.Filas {
				if ids[f.ExpedienteID] {
					t.Fatal("familia duplicada", f)
				}
				ids[f.ExpedienteID] = true
				suma += f.Cantidad
			}
			if suma != r.Cifra {
				t.Fatal("suma", r)
			}
			comprobaciones++
		}
	}
	// Dos niñas del mismo expediente aportan dos unidades, pero una familia.
	for _, a := range aportes {
		if a.ExpedienteID == "A" && a.Apartado == e.CasosAtendidos {
			if a.Total.Familias != 1 || a.Total.Menores != 2 {
				t.Fatal(a)
			}
		}
	}
	bjson, err := json.Marshal(informe)
	if err != nil {
		t.Fatal(err)
	}
	var foto e.InformeMemoria
	if err := json.Unmarshal(bjson, &foto); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(foto.Desglose, informe.Desglose) {
		t.Fatal("foto pierde aportes")
	}
	t.Logf("DESGLOSE-SUMA-OK: %d cifras, 28 apartados y subdivisiones; actividad sin vínculo explícita", comprobaciones)
}
func TestFotoAntiguaNoInventaAportes(t *testing.T) {
	r, err := e.SeleccionarDesglose(e.InformeMemoria{Apartados: []e.ApartadoMemoria{{Codigo: e.CasosAtendidos, Total: e.Recuento{Familias: 2}}}}, e.CasosAtendidos, e.SeleccionDesglose{Celda: -1, Medida: "familias"})
	if err != nil || r.Cifra != 2 || r.Aviso != "memoria.desglose.foto_antigua" || len(r.Filas) != 0 {
		t.Fatal(r, err)
	}
}

func TestDesgloseIdentificaGrupoAunqueCambieSuIndice(t *testing.T) {
	clave := e.ClaveCelda{Grupo: e.Grupo{Subprograma: e.Riesgo}, Unidad: e.Familias}
	informe := e.InformeMemoria{VersionDesglose: 1, Apartados: []e.ApartadoMemoria{{Codigo: e.CasosAtendidos, Celdas: []e.CeldaMemoria{{Grupo: e.Grupo{Subprograma: e.Reunificacion}, Unidad: e.Familias, Celda: e.Celda{Numerador: 1}}, {Grupo: clave.Grupo, Unidad: clave.Unidad, Celda: e.Celda{Numerador: 1}}}}}, Desglose: []e.AporteMemoria{{ExpedienteID: "riesgo", Apartado: e.CasosAtendidos, Celdas: []e.CeldaMemoria{{Grupo: clave.Grupo, Unidad: clave.Unidad, Celda: e.Celda{Numerador: 1}}}}}}
	r, err := e.SeleccionarDesglose(informe, e.CasosAtendidos, e.SeleccionDesglose{Celda: 0, Clave: &clave, Medida: "numerador"})
	if err != nil || r.Grupo != clave.Grupo || len(r.Filas) != 1 || r.Filas[0].ExpedienteID != "riesgo" {
		t.Fatal(r, err)
	}
}
