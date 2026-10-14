package docx

// Regresión F5.02: una parte de WordprocessingML con extensión distinta de
// .xml o .rels no puede eludir la revisión de campos, OLE y altChunk.

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func conParteRenombrada(t *testing.T, plantilla []byte, nombre string) []byte {
	t.Helper()
	cabecera := `<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>DDEAUTO cmd "/c calc"</w:instrText></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p><w:p><w:fldSimple w:instr="INCLUDEPICTURE &quot;\\servidor.invalid\x.png&quot;"/></w:p></w:hdr>`
	z, err := zip.NewReader(bytes.NewReader(plantilla), int64(len(plantilla)))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, f := range z.File {
		if f.Name != "[Content_Types].xml" {
			if err := w.Copy(f); err != nil {
				t.Fatal(err)
			}
			continue
		}
		r, _ := f.Open()
		var tipos bytes.Buffer
		_, _ = tipos.ReadFrom(r)
		_ = r.Close()
		nuevo := strings.Replace(tipos.String(), "</Types>", `<Override PartName="/`+nombre+`" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/></Types>`, 1)
		e, _ := w.Create(f.Name)
		_, _ = e.Write([]byte(nuevo))
	}
	e, _ := w.Create(nombre)
	_, _ = e.Write([]byte(cabecera))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestPublicacionRevisaPartesXMLConOtraExtension(t *testing.T) {
	base, _ := acta(t)
	for _, nombre := range []string{"word/header9.bin", "word/header9.dat", "word/header9.xml.bin"} {
		if err := ValidarCambio(conParteRenombrada(t, base, nombre), base); err == nil {
			t.Fatalf("%s: se aceptó una cabecera con DDE fuera de la revisión", nombre)
		}
	}
	// Control: la misma parte con extensión .xml ya se rechazaba.
	if err := ValidarCambio(conParteRenombrada(t, base, "word/header9.xml"), base); err == nil {
		t.Fatal("cabecera .xml con DDE aceptada")
	}
}
