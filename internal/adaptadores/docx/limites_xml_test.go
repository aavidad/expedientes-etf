package docx

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// La entrada mantiene el contrato de una plantilla oficial. Las declaraciones
// añadidas son inertes; la regresión comprueba el coste de copiar sus mapas.
func TestPublicacionLimitaAmplificacionNamespaces(t *testing.T) {
	base, _ := acta(t)
	var espacios strings.Builder
	for i := 0; i < 512; i++ {
		fmt.Fprintf(&espacios, ` xmlns:n%d="urn:etf:%d"`, i, i)
	}
	candidata := cambiarXML(t, base, func(xml []byte) []byte {
		xml = bytes.Replace(xml, []byte("<w:document "), []byte("<w:document "+espacios.String()+" "), 1)
		return bytes.Replace(xml, []byte("</w:body>"), []byte(strings.Repeat("<w:p/>", 2000)+"</w:body>"), 1)
	})
	var antes, despues runtime.MemStats
	runtime.ReadMemStats(&antes)
	err := ValidarCambio(candidata, base)
	runtime.ReadMemStats(&despues)
	t.Logf("DOCX comprimido: %d bytes; asignaciones: %d bytes; rechazado: %t", len(candidata), despues.TotalAlloc-antes.TotalAlloc, err != nil)
	if err == nil {
		t.Fatal("la publicación aceptó una plantilla con amplificación de namespaces sin cota")
	}
}

func TestXMLLimitaTrabajoAcumulado(t *testing.T) {
	var espacios strings.Builder
	for i := 0; i < 32; i++ {
		fmt.Fprintf(&espacios, ` xmlns:n%d="urn:etf:%d"`, i, i)
	}
	// Cada hermano redeclara un espacio y obliga a copiar los heredados.
	xml := []byte("<raiz" + espacios.String() + ">" + strings.Repeat(`<nodo xmlns:z="urn:etf:z"/>`, 16000) + "</raiz>")
	if _, err := leerXML(xml, nil); err == nil {
		t.Fatal("el presupuesto debe contar las copias heredadas entre hermanos")
	}
	// Los hermanos que no declaran nada comparten el mapa del padre.
	compartido := []byte("<raiz" + espacios.String() + ">" + strings.Repeat("<nodo/>", 16000) + "</raiz>")
	if _, err := leerXML(compartido, nil); err != nil {
		t.Fatal("hermanos sin declaraciones rechazados", err)
	}
	if _, err := leerXML([]byte("<raiz"+espacios.String()+"><nodo/></raiz>"), nil); err != nil {
		t.Fatal("documento pequeño con namespaces válidos rechazado", err)
	}
}
