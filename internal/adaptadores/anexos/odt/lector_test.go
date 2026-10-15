package odt

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestMuestras(t *testing.T) {
	b, err := os.ReadFile("testdata/esperado.json")
	if err != nil {
		t.Fatal(err)
	}
	var esperado map[string]map[string][]string
	if err = json.Unmarshal(b, &esperado); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"anexo3", "anexo4"} {
		t.Run(n, func(t *testing.T) {
			b, err := os.ReadFile("testdata/" + n + ".odt")
			if err != nil {
				t.Fatal(err)
			}
			e, err := Leer(b)
			if err != nil {
				t.Fatal(err)
			}
			if !e.PlantillaReconocida || e.Tipo != n {
				t.Fatal(e.Tipo, e.PlantillaReconocida)
			}
			valores := map[string][]string{}
			for _, c := range e.Campos {
				valores[c.Codigo] = append(valores[c.Codigo], c.Valor)
			}
			if !reflect.DeepEqual(valores, esperado[n]) {
				t.Fatalf("valores distintos: %v", valores)
			}
			if len(valores["menores.persona.nombre"]) != 2 || len(valores["sociodemografia.adulto.nombre"]) != 2 || len(valores["sociodemografia.persona_significativa.nombre"]) != 1 {
				t.Fatal("personas vacías")
			}
			t.Logf("tipo=%s campos=%d menores=2 adultos=2", e.Tipo, len(e.Campos))
		})
	}
}

func mutar(t *testing.T, nombre string, transformar func(*zip.Writer)) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".odt")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range z.File {
		if f.Name == "content.xml" {
			continue
		}
		datos, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		dst, err := w.CreateHeader(&f.FileHeader)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(dst, datos)
		datos.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	transformar(w)
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestContenedorHostil(t *testing.T) {
	for _, caso := range []struct{ nombre, parte, contenido, clave string }{
		{"ruta", "../content.xml", "x", "anexo.odt_zip_invalido"},
		{"absoluta", "/content.xml", "x", "anexo.odt_zip_invalido"},
		{"barra", "Pictures\\dato.png", "x", "anexo.odt_zip_invalido"},
		{"macro", "Scripts/macro.xml", "x", "anexo.odt_contenido_inseguro"},
		{"objeto", "Object 1/content.xml", "x", "anexo.odt_contenido_inseguro"},
		{"bomba", "content.xml", strings.Repeat("x", 33<<20), "anexo.odt_limite_descompresion"},
		{"entidad", "content.xml", `<!DOCTYPE x [<!ENTITY a SYSTEM "file:///etc/passwd">]><x/>`, "anexo.odt_contenido_inseguro"},
		{"profundidad", "content.xml", strings.Repeat("<x>", 65) + strings.Repeat("</x>", 65), "anexo.odt_limite_xml"},
		{"atributos", "content.xml", "<x " + strings.Repeat(`a="v" `, 129) + "/>", "anexo.odt_limite_xml"},
		{"repeticion", "content.xml", `<x xmlns:t="urn:oasis:names:tc:opendocument:xmlns:table:1.0" t:number-rows-repeated="100000000"/>`, "anexo.odt_limite_xml"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			b := mutar(t, "anexo3", func(w *zip.Writer) {
				dst, err := w.Create(caso.parte)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = dst.Write([]byte(caso.contenido)); err != nil {
					t.Fatal(err)
				}
			})
			e, err := Leer(b)
			if err == nil || err.Error() != caso.clave || len(e.Campos) != 0 {
				t.Fatal(err, e)
			}
			acta, err := LeerActa(b)
			if err == nil || err.Error() != caso.clave || len(acta.Campos) != 0 {
				t.Fatal("acta sin protección del contenedor", err, acta)
			}
		})
	}
}

func TestFechasYCasillas(t *testing.T) {
	for _, caso := range []struct{ entrada, salida string }{{"d14/03/24", "14/03/2024"}, {"S06/10/2026", "06/10/2026"}, {"29/02/24", "29/02/2024"}} {
		got, m := fecha(caso.entrada)
		if got != caso.salida || m != "" {
			t.Fatal(got, m)
		}
	}
	if _, m := fecha("31/02/24"); m != "anexo.fecha_invalida" {
		t.Fatal(m)
	}
	for _, c := range []struct{ entrada, salida, motivo string }{{"☒ SI ☐NO", "SI", ""}, {"☐ SI ☒NO", "NO", ""}, {"☐ SI ☐NO", "", "anexo.campo_ausente"}, {"☒ SI ☒NO", "", "anexo.valor_ambiguo"}, {"SI NO", "", "anexo.casilla_no_legible"}} {
		v, m := seleccion(c.entrada, []string{"SI", "NO"})
		if v != c.salida || m != c.motivo {
			t.Fatal(v, m)
		}
	}
}

func TestQuinceMenoresConFilasVacias(t *testing.T) {
	original, err := os.ReadFile("testdata/anexo3.odt")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(original), int64(len(original)))
	if err != nil {
		t.Fatal(err)
	}
	f, err := z.Open("content.xml")
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	inicio := strings.Index(string(content), `<table:table table:name="Tabla4"`)
	fin := inicio + strings.Index(string(content)[inicio:], "</table:table>") + len("</table:table>")
	tabla := string(content)[inicio:fin]
	filas := regexp.MustCompile(`(?s)<table:table-row\b.*?</table:table-row>`).FindAllString(tabla, -1)
	if len(filas) != 5 {
		t.Fatal("filas de la muestra", len(filas))
	}
	for _, cantidad := range []int{15, 16} {
		nuevas := filas[0] + strings.Repeat(filas[1], cantidad) + strings.ReplaceAll(filas[3], "dd/mm/aa.", "d/mm/aa.")
		tablaNueva := strings.Replace(tabla, strings.Join(filas, ""), nuevas, 1)
		datos := strings.Replace(string(content), tabla, tablaNueva, 1)
		b := mutar(t, "anexo3", func(w *zip.Writer) {
			dst, e := w.Create("content.xml")
			if e != nil {
				t.Fatal(e)
			}
			if _, e = dst.Write([]byte(datos)); e != nil {
				t.Fatal(e)
			}
		})
		e, err := Leer(b)
		if cantidad == 16 {
			if err == nil || err.Error() != "anexo.docx_demasiados_menores" {
				t.Fatal(err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, c := range e.Campos {
			if c.Codigo == "menores.persona.nombre" {
				count++
			}
		}
		if count != 15 {
			t.Fatal(count)
		}
	}
}

func TestMetadatosLimpios(t *testing.T) {
	for _, n := range []string{"anexo3", "anexo4"} {
		b, err := os.ReadFile("testdata/" + n + ".odt")
		if err != nil {
			t.Fatal(err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range z.File {
			if !strings.HasSuffix(f.Name, ".xml") {
				continue
			}
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			for _, correo := range regexp.MustCompile(`@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`).FindAll(b, -1) {
				if string(correo) != "@servicios.invalid" {
					t.Fatal(n, "correo sin sanear")
				}
			}
			if f.Name == "meta.xml" {
				root, err := leerXML(b)
				if err != nil {
					t.Fatal(err)
				}
				for _, v := range root.buscar("http://purl.org/dc/elements/1.1/", "creator") {
					if v.texto(nil) != "Expedientes ETF" {
						t.Fatal("autor sin sanear")
					}
				}
				for _, v := range root.buscar("urn:oasis:names:tc:opendocument:xmlns:meta:1.0", "initial-creator") {
					if v.texto(nil) != "Expedientes ETF" {
						t.Fatal("autor inicial sin sanear")
					}
				}
			}
		}
	}
}
