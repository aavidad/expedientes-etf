package documento_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMemoriaPlantillaJuntaTodasLasCeldasYTextos(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			ruta := "../../../../idiomas/" + idioma + "/plantillas/memoria/"
			b, err := os.ReadFile(ruta + "v1.0.0.docx")
			if err != nil {
				t.Fatal(err)
			}
			j, err := os.ReadFile(ruta + "v1.0.0.json")
			if err != nil {
				t.Fatal(err)
			}
			var p d.PresentacionMemoria
			if err = json.Unmarshal(j, &p); err != nil {
				t.Fatal(err)
			}
			informe := e.InformeMemoria{FiltroMemoria: e.FiltroMemoria{Anio: 2026}, Corte: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Narrativas: e.Narrativas{D: "Las visitas se acordaron con las familias. <seguimiento> & coordinación."}}
			codigos := map[string]bool{}
			for _, c := range p.Celdas {
				if !codigos[c.Apartado] {
					codigos[c.Apartado] = true
					informe.Apartados = append(informe.Apartados, e.ApartadoMemoria{Codigo: e.Apartado(c.Apartado), Total: e.Recuento{FamiliasSinRelacionMenores: 2}})
				}
			}
			motor := word.Motor{}
			campos, err := motor.CamposMemoria(context.Background(), informe, p)
			if err != nil {
				t.Fatal(err)
			}
			salida, err := motor.Generar(context.Background(), b, d.Relleno{Campos: campos})
			if err != nil {
				t.Fatal(err)
			}
			repetida, err := motor.Generar(context.Background(), b, d.Relleno{Campos: campos})
			if err != nil || !bytes.Equal(salida, repetida) {
				t.Fatal("resultado no reproducible", err)
			}
			z, err := zip.NewReader(bytes.NewReader(salida), int64(len(salida)))
			if err != nil {
				t.Fatal(err)
			}
			var xml string
			for _, f := range z.File {
				if f.Name == "word/document.xml" {
					r, err := f.Open()
					if err != nil {
						t.Fatal(err)
					}
					contenido, err := io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
					xml = string(contenido)
				}
			}
			if strings.Contains(xml, "{{") || !strings.Contains(xml, "&lt;seguimiento&gt; &amp; coordinación.") {
				t.Fatal("marcadores pendientes o XML sin escapar")
			}
			if !strings.Contains(xml, "2026") || !strings.Contains(xml, p.Campos["cabecera_g"]) {
				t.Fatal("falta año o narrativa G")
			}
			if strings.Contains(xml, ">72<") || strings.Contains(xml, ">26,77<") {
				t.Fatal("cifras heredadas del cliente")
			}
			if strings.Count(xml, "<w:tbl>") < 32 {
				t.Fatal("se perdieron tablas de la Junta")
			}
		})
	}
}

func TestMemoriaTotalesEdadCalculados(t *testing.T) {
	b, err := os.ReadFile("../../../../idiomas/es/plantillas/memoria/v1.0.0.json")
	if err != nil {
		t.Fatal(err)
	}
	var p d.PresentacionMemoria
	if err = json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	informe := e.InformeMemoria{FiltroMemoria: e.FiltroMemoria{Anio: 2026}}
	seen := map[string]bool{}
	for _, spec := range p.Celdas {
		if !seen[spec.Apartado] {
			seen[spec.Apartado] = true
			informe.Apartados = append(informe.Apartados, e.ApartadoMemoria{Codigo: e.Apartado(spec.Apartado)})
		}
	}
	for i := range informe.Apartados {
		if informe.Apartados[i].Codigo == e.MenoresPorEdad {
			informe.Apartados[i].Total.Menores = 9
			informe.Apartados[i].Celdas = []e.CeldaMemoria{
				{Grupo: e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nino, Tramo: "0_5"}, Unidad: e.Menores, Celda: e.Celda{Numerador: 2}},
				{Grupo: e.Grupo{Subprograma: e.Riesgo, Sexo: e.Nina, Tramo: "6_11"}, Unidad: e.Menores, Celda: e.Celda{Numerador: 3}},
				{Grupo: e.Grupo{Subprograma: e.Reunificacion, Sexo: e.Nino, Tramo: "12_17"}, Unidad: e.Menores, Celda: e.Celda{Numerador: 4}},
			}
		}
	}
	campos, err := (word.Motor{}).CamposMemoria(context.Background(), informe, p)
	if err != nil {
		t.Fatal(err)
	}
	for clave, esperado := range map[string]string{"celda_29_6_1": "2", "celda_29_6_3": "3", "celda_29_6_5": "4", "celda_29_6_7": "0", "celda_29_7_1": "6", "celda_29_7_3": "3"} {
		if campos[clave] != esperado {
			t.Fatal(clave, campos[clave], esperado)
		}
	}
}
