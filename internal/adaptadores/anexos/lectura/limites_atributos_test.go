package lectura

// Regresión F5.02: un document.xml con un elemento de millones de atributos
// se rechaza antes de que el decodificador XML los reserve en memoria.

import (
	"archive/zip"
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
)

func docxAlmacenado(t *testing.T, documento string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for nombre, contenido := range map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"word/document.xml":   documento,
	} {
		w, err := z.CreateHeader(&zip.FileHeader{Name: nombre, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(contenido))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDOCXRechazaAtributosMasivosSinReservarlos(t *testing.T) {
	inicio := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body`
	casos := map[string]string{
		"un_elemento": inicio + strings.Repeat(` x=""`, 1_590_000) + `/></w:document>`,
		"muchos":      inicio + `>` + strings.Repeat(`<w:p a="" b="" c="" d="" e="" f="" g="" h=""/>`, 150_000) + `</w:body></w:document>`,
	}
	for nombre, documento := range casos {
		b := docxAlmacenado(t, documento)
		var antes, despues runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&antes)
		_, errValidar := (Validador{}).Validar(context.Background(), "anexo.docx", b)
		_, errLeer := NuevoLector().Leer(context.Background(), "docx", b)
		runtime.ReadMemStats(&despues)
		asignado := despues.TotalAlloc - antes.TotalAlloc
		if errValidar == nil || errValidar.Error() != "anexo.docx_limite_xml" || errLeer == nil || errLeer.Error() != "anexo.docx_limite_xml" || asignado > 96<<20 {
			t.Fatalf("%s: validar %v, leer %v, asignados %d bytes", nombre, errValidar, errLeer, asignado)
		}
	}
}
