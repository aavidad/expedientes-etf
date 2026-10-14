package docx

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicacionCamposYEnlacesSeguros(t *testing.T) {
	base, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.1.1.docx")
	if err != nil {
		t.Fatal(err)
	}
	campo := func(instruccion string) string {
		return `<w:p><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>` + instruccion + `</w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>`
	}
	for _, c := range []struct {
		nombre, xml string
		aceptada    bool
	}{
		{"PAGE", campo(" PAGE "), true},
		{"NUMPAGES", campo("NUMPAGES"), true},
		{"SECTIONPAGES", campo("SECTIONPAGES"), true},
		{"PAGE_partido", campo(` PA</w:instrText></w:r><w:r><w:instrText>GE `), true},
		{"DDE", campo(`DDE cmd "/c calc"`), false},
		{"DDE_partido", campo(`D</w:instrText></w:r><w:r><w:instrText>DE cmd`), false},
		{"DDEAUTO", campo(`DDEAUTO cmd`), false},
		{"INCLUDETEXT", campo(`INCLUDETEXT https://archivo.invalid/texto`), false},
		{"HYPERLINK_file", campo(`HYPERLINK file:///C:/privado`), false},
		{"MACROBUTTON", campo(`MACROBUTTON ejecutar`), false},
		{"PAGE_sufijo", campo(`PAGE DDE cmd`), false},
		{"PAGE_opciones", campo(`PAGE \* MERGEFORMAT`), false},
		{"instruccion_huerfana", `<w:p><w:r><w:instrText>PAGE</w:instrText></w:r></w:p>`, false},
		{"campo_incompleto", `<w:p><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText>PAGE</w:instrText></w:r></w:p>`, false},
		{"campo_anidado", campo(`PAGE</w:instrText><w:fldChar w:fldCharType="begin"/><w:instrText>PAGE`), false},
		{"instruccion_en_resultado", strings.Replace(campo("PAGE"), `<w:t>1</w:t>`, `<w:instrText>DDE cmd</w:instrText>`, 1), false},
		{"OLE", `<w:p><w:r><w:object/></w:r></w:p>`, false},
		{"OLE_namespace", `<w:p><w:r><o:OLEObject xmlns:o="urn:schemas-microsoft-com:office:office"/></w:r></w:p>`, false},
		{"ActiveX", `<w:p><w:r><w:control/></w:r></w:p>`, false},
		{"campo_simple", `<w:p><w:fldSimple w:instr="DDE cmd"/></w:p>`, false},
		{"instruccion_borrada", `<w:p><w:r><w:delInstrText>DDE cmd</w:delInstrText></w:r></w:p>`, false},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			nueva := cambiarParte(t, base, "word/document.xml", func(b []byte) []byte {
				return bytes.Replace(b, []byte("</w:body>"), []byte(c.xml+"</w:body>"), 1)
			})
			if err := ValidarCambio(nueva, base); (err == nil) != c.aceptada {
				t.Fatalf("aceptada=%v, error=%v", c.aceptada, err)
			}
			if !c.aceptada {
				if err := ValidarCambio(base, nueva); err != nil {
					t.Fatalf("defecto histórico bloquea candidata segura: %v", err)
				}
			}
		})
	}
	for _, c := range []struct {
		destino  string
		aceptada bool
	}{
		{"https://proteccion.invalid/informacion", true},
		{"file:///C:/privado", false}, {"smb://servidor/carpeta", false},
		{"http://proteccion.invalid", false}, {`\\servidor\carpeta`, false},
		{"//servidor/carpeta", false}, {"https://", false},
		{`https://servidor\carpeta`, false}, {"https://persona@servidor.invalid", false},
	} {
		t.Run(c.destino, func(t *testing.T) {
			nueva := cambiarParte(t, base, "_rels/.rels", func(b []byte) []byte {
				rel := `<Relationship Id="enlace" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" TargetMode="External" Target="` + c.destino + `"/>`
				return bytes.Replace(b, []byte("</Relationships>"), []byte(rel+"</Relationships>"), 1)
			})
			if err := ValidarCambio(nueva, base); (err == nil) != c.aceptada {
				t.Fatalf("aceptada=%v, error=%v", c.aceptada, err)
			}
		})
	}
	for _, nombre := range []string{"word/vbaProject.bin", "word/embeddings/objeto.bin", "word/activeX/control.bin"} {
		t.Run(nombre, func(t *testing.T) {
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			// Copiamos las partes originales y añadimos contenido activo inerte.
			z, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range z.File {
				if err := w.Copy(f); err != nil {
					t.Fatal(err)
				}
			}
			f, err := w.Create(nombre)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Write([]byte("contenido inerte")); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			if err := ValidarCambio(buf.Bytes(), base); err == nil {
				t.Fatal("parte activa aceptada")
			}
		})
	}
}

func TestPublicacion28CandidatasOficiales(t *testing.T) {
	total := 0
	err := filepath.WalkDir("../../../idiomas", func(ruta string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() || !(strings.HasSuffix(ruta, "v1.0.1.docx") || strings.HasSuffix(ruta, "v1.1.1.docx")) {
			return nil
		}
		total++
		t.Run(strings.TrimPrefix(ruta, "../../../idiomas/"), func(t *testing.T) {
			nueva, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(strings.TrimSuffix(ruta, "1.docx") + "0.docx")
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidarCambio(nueva, actual); err != nil {
				t.Fatal(err)
			}
		})
		return nil
	})
	if err != nil || total != 28 {
		t.Fatalf("candidatas=%d, error=%v", total, err)
	}
	t.Log("28 candidatas oficiales aceptadas contra sus predecesoras")
}
