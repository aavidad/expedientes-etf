package documento_test

import (
	"context"
	"encoding/json"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	textos "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCifrasWordConservaDivisorCeroYFecha(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			ruta := "../../../../idiomas/" + idioma + "/plantillas/memoria/"
			j, err := os.ReadFile(ruta + "v1.0.0.json")
			if err != nil {
				t.Fatal(err)
			}
			var p d.PresentacionMemoria
			if json.Unmarshal(j, &p) != nil {
				t.Fatal("presentación")
			}
			metadatos, err := (textos.TextosCifras{Raiz: "../../../../idiomas"}).LeerTextosCifras(context.Background(), idioma)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range metadatos {
				p.Campos[k] = v
			}
			r := e.CifrasConsulta{Anio: 2026, Fecha: time.Date(2026, 10, 14, 12, 30, 0, 0, time.UTC), Definitivas: true}
			codigos := map[string]bool{}
			var objetivo d.CampoMemoria
			for _, c := range p.Celdas {
				if !codigos[c.Apartado] {
					r.Apartados = append(r.Apartados, e.ApartadoConsulta{Codigo: e.Apartado(c.Apartado)})
					codigos[c.Apartado] = true
				}
				if objetivo.Campo == "" && c.Porcentaje {
					objetivo = c
				}
			}
			if objetivo.Campo == "" {
				t.Fatal("no hay porcentaje")
			}
			f := objetivo.Filtros
			g := e.GrupoConsulta{Subprograma: e.Subprograma(f["subprograma"]), Origen: f["origen"], Sexo: e.Sexo(f["sexo"]), Tramo: f["tramo"], Motivo: e.Motivo(f["motivo"]), Papel: f["papel"], Estado: f["estado"], Causa: f["causa"], Organismo: f["organismo"]}
			cero := 0
			for i := range r.Apartados {
				if string(r.Apartados[i].Codigo) == objetivo.Apartado {
					r.Apartados[i].Celdas = []e.CeldaConsulta{{Grupo: g, Unidad: e.Unidad(objetivo.Unidad), Denominador: &cero, Porcentaje: &cero, AvisoPorcentaje: "estadistica.divisor_cero"}}
					r.Apartados[i].Avisos = map[string]int{"estadistica.divisor_cero": 1}
				}
			}
			campos, err := (word.Motor{}).CamposCifras(context.Background(), r, p)
			if err != nil {
				t.Fatal(err)
			}
			esperado := "0" + p.Campos["separador_decimal"] + "00"
			if campos[objetivo.Campo] != esperado {
				t.Fatal("divisor cero", campos[objetivo.Campo], esperado)
			}
			if !strings.Contains(campos["texto_1"], "2026-10-14T12:30:00Z") || !strings.Contains(campos["texto_1"], metadatos["estadistica.consulta_definitiva"]) {
				t.Fatal("fecha o estado")
			}
			for _, k := range []string{"datos_a", "personal_a", "observaciones_a", "narrativa_d", "narrativa_e", "narrativa_f", "narrativa_g"} {
				if campos[k] != "" {
					t.Fatal("campo manual", k)
				}
			}
			b, err := os.ReadFile(ruta + "v1.0.0.docx")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = (word.Motor{}).Generar(context.Background(), b, d.Relleno{Campos: campos}); err != nil {
				t.Fatal("Word", err)
			}
		})
	}
}
