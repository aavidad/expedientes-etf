package lectura

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--lector-anexo-pdf" {
		if err := EjecutarPDF(os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestDOCXReal(t *testing.T) {
	b, err := os.ReadFile("../docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	f, err := (Validador{}).Validar(context.Background(), "Derivacion.docx", b)
	if err != nil || f != "docx" {
		t.Fatal(f, err)
	}
	e, err := NuevoLector().Leer(context.Background(), f, b)
	if err != nil || !e.PlantillaReconocida || len(e.Campos) < 40 {
		t.Fatal(e, err)
	}
	if _, err := (Validador{}).Validar(context.Background(), "Derivacion.pdf", b); !errors.Is(err, dominio.ErrFormato) {
		t.Fatal(err)
	}
}

func TestPDFRealYSinTexto(t *testing.T) {
	for _, n := range []string{"anexo3.pdf", "anexo4.pdf", "sin_texto.pdf", "corrupto.pdf"} {
		t.Run(n, func(t *testing.T) {
			b, err := os.ReadFile("../pdf/testdata/" + n)
			if err != nil {
				t.Fatal(err)
			}
			e, err := NuevoLector().Leer(context.Background(), "pdf", b)
			if n == "corrupto.pdf" {
				if err == nil {
					t.Fatal("PDF corrupto admitido")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if n == "sin_texto.pdf" {
				if len(e.Campos) != 0 || len(e.Avisos) != 1 || e.Avisos[0] != "anexo.pdf_sin_texto" {
					t.Fatal(e)
				}
			} else if !e.PlantillaReconocida || len(e.Campos) < 40 {
				t.Fatal(e)
			}
			if n == "anexo3.pdf" && e.Tipo != "anexo3" || n == "anexo4.pdf" && e.Tipo != "anexo4" {
				t.Fatal("tipo de anexo no leído", e.Tipo)
			}
		})
	}
}

func documentoZIP(t *testing.T, nombre string, datos []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, err := z.Create(nombre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(datos); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestMacrosBombaZIPYRutas(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		b      []byte
		clave  string
	}{
		{"macro.docm", []byte("PK"), "anexo.docx_macros"},
		{"macro.docx", documentoZIP(t, "word/vbaProject.bin", []byte("codigo")), "anexo.docx_macros"},
		{"bomba.docx", documentoZIP(t, "word/document.xml", bytes.Repeat([]byte("x"), 33<<20)), "anexo.docx_limite_descompresion"},
		{"salto.docx", documentoZIP(t, "../word/document.xml", []byte("<document/>")), "anexo.docx_zip_invalido"},
		{"anidado.docx", documentoZIP(t, "word/embeddings/documento.zip", []byte("PK")), "expediente.anexo_formato"},
		{"entidad.docx", documentoZIP(t, "word/document.xml", []byte(`<!DOCTYPE foo SYSTEM "file:///etc/passwd"><foo/>`)), "anexo.docx_xml_inseguro"},
		{"macro_tipo.docx", documentoZIP(t, "[Content_Types].xml", []byte(`<Types><Default Extension="bin" ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/></Types>`)), "anexo.docx_macros"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := (Validador{}).Validar(context.Background(), caso.nombre, caso.b)
			if err == nil || err.Error() != caso.clave {
				t.Fatal(err)
			}
		})
	}
	if _, err := (Validador{}).Validar(context.Background(), "Grande.docx", make([]byte, dominio.MaxBytes+1)); !errors.Is(err, dominio.ErrTamano) {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := NuevoLector().Leer(ctx, "pdf", []byte(strings.Repeat("x", 100))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
