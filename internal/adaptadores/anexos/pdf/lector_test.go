package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func limitesPrueba() Limites {
	return Limites{Paginas: 20, Objetos: 10000, Lecturas: 100000, Glifos: 200000}
}

func TestCorpusRecomendado(t *testing.T) {
	var perfil Perfil
	var esperado map[string]map[string][]string
	var desviaciones map[string]map[string][]string
	leerJSON(t, "perfil.json", &perfil)
	leerJSON(t, "esperado.json", &esperado)
	leerJSON(t, "desviaciones.json", &desviaciones)
	for _, motor := range []string{"dslipak"} {
		for _, nombre := range []string{"anexo3", "anexo4"} {
			t.Run(motor+"/"+nombre, func(t *testing.T) {
				b, e := os.ReadFile("testdata/" + nombre + ".pdf")
				if e != nil {
					t.Fatal(e)
				}
				d, e := Leer(b, motor, limitesPrueba())
				if e != nil {
					t.Fatal(e)
				}
				lectura := Extraer(d, perfil)
				valores := map[string][]string{}
				for _, c := range lectura.Campos {
					if !c.Dudoso || len(c.Motivos) == 0 {
						t.Fatal("anexo.confirmacion_ausente")
					}
					valores[c.Codigo] = append(valores[c.Codigo], c.Valor)
				}
				aciertos, total := 0, 0
				for key, want := range esperado[nombre] {
					total += len(want)
					got := valores[key]
					for i, v := range want {
						if i < len(got) && v == got[i] {
							aciertos++
						}
					}
					if motor == "dslipak" && !reflect.DeepEqual(want, got) {
						if !reflect.DeepEqual(desviaciones[nombre][key], got) || len(got) == 0 {
							t.Errorf("%s: obtenido=%q esperado=%q", key, got, want)
						}
					}
				}
				t.Logf("paginas=%d plantilla=%v campos_exactos=%d/%d", d.Paginas, lectura.PlantillaReconocida, aciertos, total)
				if motor == "dslipak" && !lectura.PlantillaReconocida {
					t.Fatal("anexo.plantilla_no_reconocida")
				}
			})
		}
	}
}

func leerJSON(t *testing.T, nombre string, v any) {
	t.Helper()
	b, e := os.ReadFile("testdata/" + nombre)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, v); e != nil {
		t.Fatal(e)
	}
}

func TestValidacion(t *testing.T) {
	anexo, e := os.ReadFile("testdata/anexo3.pdf")
	if e != nil {
		t.Fatal(e)
	}
	imagen, e := os.ReadFile("testdata/sin_texto.pdf")
	if e != nil {
		t.Fatal(e)
	}
	corrupto, e := os.ReadFile("testdata/corrupto.pdf")
	if e != nil {
		t.Fatal(e)
	}
	for _, motor := range []string{"dslipak"} {
		t.Run(motor, func(t *testing.T) {
			d, e := Leer(imagen, motor, limitesPrueba())
			if e != nil || !d.SinTexto {
				t.Fatalf("sin_texto=%v error=%v", d.SinTexto, e)
			}
			var perfil Perfil
			leerJSON(t, "perfil.json", &perfil)
			l := Extraer(d, perfil)
			if len(l.Campos) != 0 || !reflect.DeepEqual(l.Avisos, []string{"anexo.pdf_sin_texto"}) {
				t.Fatal("anexo.sin_texto_no_detectado")
			}
			casos := []struct {
				nombre  string
				datos   []byte
				limites Limites
				want    error
			}{
				{"corrupto", corrupto, limitesPrueba(), ErrCorrupto},
				{"cabecera", bytes.Replace(anexo, []byte("%PDF"), []byte("%DOC"), 1), limitesPrueba(), ErrCorrupto},
				{"estructura", []byte("%PDF-1.4\n" + strings.Repeat("x", 128) + "\n%%EOF"), limitesPrueba(), ErrCorrupto},
				{"count_incoherente", bytes.Replace(pdfVacio(2, 0), []byte("/Count 2"), []byte("/Count 1"), 1), limitesPrueba(), ErrCorrupto},
				{"arbol_ciclico", bytes.Replace(pdfVacio(1, 0), []byte("[3 0 R"), []byte("[2 0 R"), 1), limitesPrueba(), ErrRecursos},
				{"pagina_invalida", bytes.Replace(pdfVacio(1, 0), []byte("/Type /Page /"), []byte("/Type /Nada /"), 1), limitesPrueba(), ErrCorrupto},
				{"bytes", make([]byte, MaxBytes+1), limitesPrueba(), ErrTamano},
				{"paginas", anexo, Limites{4, 10000, 100000, 200000}, ErrPaginas},
				{"muchas_paginas", pdfVacio(21, 0), limitesPrueba(), ErrPaginas},
				{"objetos", anexo, Limites{20, 3, 100000, 200000}, ErrObjetos},
				{"lecturas", anexo, Limites{20, 10000, 1, 200000}, ErrRecursos},
				{"glifos", anexo, Limites{20, 10000, 100000, 1}, ErrRecursos},
				{"limites_invalidos", anexo, Limites{}, ErrLimites},
			}
			for _, c := range casos {
				t.Run(c.nombre, func(t *testing.T) {
					doc, e := Leer(c.datos, motor, c.limites)
					if !errors.Is(e, c.want) {
						t.Fatalf("obtenido=%v esperado=%v", e, c.want)
					}
					if len(doc.Lineas) > 0 {
						t.Fatal("anexo.resultado_parcial")
					}
				})
			}
			if _, e := Leer(anexo, motor, Limites{5, 10000, 100000, 200000}); e != nil {
				t.Fatal(e)
			}
			// PDF válido en el límite decimal exacto; todos sus offsets se recalculan.
			b := pdfVacio(1, 0)
			padding := MaxBytes - len(b)
			for i := 0; i < 3; i++ {
				b = pdfVacio(1, padding)
				padding += MaxBytes - len(b)
			}
			if len(b) != MaxBytes {
				t.Fatalf("bytes=%d", len(b))
			}
			if d, e := Leer(b, motor, limitesPrueba()); e != nil || !d.SinTexto {
				t.Fatalf("limite_exacto: %v", e)
			}
		})
	}
}

func pdfVacio(paginas, padding int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n%" + strings.Repeat("x", padding) + "\n")
	var offsets []int
	obj := func(s string) {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets), s)
	}
	obj("<< /Type /Catalog /Pages 2 0 R >>")
	var kids strings.Builder
	for i := 0; i < paginas; i++ {
		fmt.Fprintf(&kids, "%d 0 R ", i+3)
	}
	obj(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", paginas, kids.String()))
	for i := 0; i < paginas; i++ {
		obj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>")
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, o := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return b.Bytes()
}
