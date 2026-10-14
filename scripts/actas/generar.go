// Genera únicamente documentos de prueba con los datos inventados del corpus.
package main

import (
	"encoding/json"
	"flag"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/docx"
	"log"
	"os"
	"path/filepath"
)

type caso struct {
	Nombre, Plantilla string
	Datos             docx.Datos
}

func main() {
	destino := flag.String("destino", "internal/adaptadores/anexos/actas/testdata", "")
	flag.Parse()
	b, err := os.ReadFile(filepath.Join(*destino, "datos.json"))
	if err != nil {
		log.Fatal(err)
	}
	var casos []caso
	if err = json.Unmarshal(b, &casos); err != nil {
		log.Fatal(err)
	}
	for _, c := range casos {
		plantilla, err := os.ReadFile(c.Plantilla)
		if err != nil {
			log.Fatal(err)
		}
		b, err := docx.Generar(plantilla, c.Datos)
		if err != nil {
			log.Fatal(err)
		}
		// El generador está fuera del servidor y publica sólo el corpus inventado.
		// nosemgrep: etf-f101-sin-temporales
		if err = os.WriteFile(filepath.Join(*destino, c.Nombre+".docx"), b, 0600); err != nil {
			log.Fatal(err)
		}
	}
}
