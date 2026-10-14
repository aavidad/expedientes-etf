package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func cambiarParte(t *testing.T, b []byte, nombre string, fn func([]byte) []byte) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		contenido, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == nombre {
			contenido = fn(contenido)
		}
		out, err := w.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write(contenido); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
func TestValidarPlantillaContraMarcadoresActuales(t *testing.T) {
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidarCambio(b, b); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		nombre string
		parte  string
		fn     func([]byte) []byte
	}{
		{"marcador_desconocido", "word/document.xml", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("{{numero}}"), []byte("{{intruso}}")) }},
		{"marcador_ausente", "word/document.xml", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("{{numero}}"), nil) }},
		{"macro_content_type", "[Content_Types].xml", func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"), []byte("application/vnd.ms-word.document.macroEnabled.main+xml"))
		}},
		{"XML_DTD", "word/document.xml", func(b []byte) []byte { return append([]byte(`<!DOCTYPE document>`), b...) }},
		{"campo_word_activo", "word/document.xml", func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("</w:body>"), []byte(`<w:p><w:fldSimple w:instr="INCLUDETEXT https://fuera.invalid/recurso"><w:r><w:t>Texto</w:t></w:r></w:fldSimple></w:p></w:body>`))
		}},
		{"raiz_tipos_arbitraria", "[Content_Types].xml", func(b []byte) []byte {
			return bytes.ReplaceAll(bytes.ReplaceAll(b, []byte("<Types"), []byte("<Otro")), []byte("</Types>"), []byte("</Otro>"))
		}},
		{"raiz_relaciones_arbitraria", "_rels/.rels", func(b []byte) []byte { return []byte(`<Otro/>`) }},
		{"override_anidado", "[Content_Types].xml", func(b []byte) []byte {
			return []byte(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Default></Types>`)
		}},
		{"sin_tipo_relaciones", "[Content_Types].xml", func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("application/vnd.openxmlformats-package.relationships+xml"), []byte("application/xml"))
		}},
		{"sin_office_document", "_rels/.rels", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("/officeDocument"), []byte("/otro")) }},
		{"destino_inexistente", "_rels/.rels", func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("word/document.xml"), []byte("word/ausente.xml"))
		}},
		{"recurso_externo", "_rels/.rels", func(b []byte) []byte {
			return []byte(strings.Replace(string(b), "<Relationship ", `<Relationship TargetMode="External" `, 1))
		}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			alterada := cambiarParte(t, b, c.parte, c.fn)
			if err := ValidarCambio(alterada, b); err == nil {
				t.Fatal("plantilla peligrosa aceptada")
			}
		})
	}
	if err := ValidarCambio([]byte("PK-falso"), b); err == nil {
		t.Fatal("ZIP disfrazado aceptado")
	}
	partida := cambiarParte(t, b, "word/document.xml", func(b []byte) []byte {
		return bytes.ReplaceAll(b, []byte("{{numero}}"), []byte(`{{nu</w:t></w:r><w:r><w:t>mero}}`))
	})
	if err := ValidarCambio(partida, b); err != nil {
		t.Fatal("marcador partido válido rechazado", err)
	}
}

func TestValidarPlantillaBombaDeDescompresion(t *testing.T) {
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	ceros := make([]byte, 1<<20)
	for range 101 {
		if _, err := f.Write(ceros); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ValidarCambio(buf.Bytes(), b); err == nil {
		t.Fatal("bomba de descompresión aceptada")
	}
}

func TestDetalleMarcadoresPlantilla(t *testing.T) {
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	nueva := cambiarParte(t, b, "word/document.xml", func(xml []byte) []byte { return bytes.ReplaceAll(xml, []byte("{{numero}}"), []byte("{{intruso}}")) })
	err = ValidarCambio(nueva, b)
	diferencia, ok := err.(*DiferenciaMarcadores)
	if !ok || len(diferencia.Faltan) != 1 || diferencia.Faltan[0] != ":numero" || len(diferencia.Sobran) != 1 || diferencia.Sobran[0] != ":intruso" {
		t.Fatal(err)
	}
}
