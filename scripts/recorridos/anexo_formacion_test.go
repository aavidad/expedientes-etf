package recorridos_test

import (
	"os"
	"strings"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx"
)

// El documento del aula debe conservar una plantilla reconocida y sus seis menores.
func TestAnexoDeFormacionReconocido(t *testing.T) {
	contenido, err := os.ReadFile("../../docs/formacion/anexo3-ortega-salcedo.docx")
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := docx.Leer(contenido)
	if err != nil {
		t.Fatal(err)
	}
	if !lectura.PlantillaReconocida || lectura.Tipo != "anexo3" || lectura.Version != "cliente-2026-v1" {
		t.Fatal("Identificación incorrecta del documento de formación")
	}
	var nombres []string
	agron := false
	for _, campo := range lectura.Campos {
		if campo.Codigo == "menores.persona.nombre" {
			nombres = append(nombres, campo.Valor)
		}
		agron = agron || strings.Contains(campo.Valor, "Agrón")
	}
	esperados := []string{"Nerea", "Álvaro", "Celia", "Mateo", "Irene", "Hugo"}
	if len(nombres) != len(esperados) || !agron {
		t.Fatal("Menores o ubicación incorrectos en el documento de formación")
	}
	for i, nombre := range nombres {
		if !strings.EqualFold(strings.TrimSpace(nombre), esperados[i]+" Pérez Ortega") {
			t.Fatalf("Nombre incorrecto en la fila %d del documento de formación", i+1)
		}
	}
}
