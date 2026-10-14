//go:build generar_corpus

// Este programa rellena las copias preparadas por generar.py con el motor Word.
// Se ejecuta únicamente para regenerar datos de prueba en el entorno aislado.
package docx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	motor "github.com/aavidad/expedientes-etf/internal/adaptadores/docx"
)

func TestGenerarCorpus(t *testing.T) {
	zona := os.Getenv("ETF_DOCX_CORPUS")
	if zona == "" {
		t.Fatal("anexo.docx_generador_argumentos")
	}
	for _, nombre := range []string{"anexo3", "anexo4", "version_desconocida"} {
		plantilla, err := os.ReadFile(filepath.Join(zona, nombre+".base.docx"))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(zona, nombre+".controles.json"))
		if err != nil {
			t.Fatal(err)
		}
		var controles map[string]string
		if err = json.Unmarshal(b, &controles); err != nil {
			t.Fatal(err)
		}
		doc, err := motor.Generar(plantilla, motor.Datos{Controles: controles})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(zona, nombre+".docx"), doc, 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.Remove(filepath.Join(zona, nombre+".base.docx")); err != nil {
			t.Fatal(err)
		}
		if err = os.Remove(filepath.Join(zona, nombre+".controles.json")); err != nil {
			t.Fatal(err)
		}
	}
}
