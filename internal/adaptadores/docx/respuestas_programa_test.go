package docx

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestD18NuevasPlantillasSinAnexosAntiguosYConMismosMarcadores(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		base := filepath.Join("../../../idiomas", idioma, "plantillas")
		b, err := os.ReadFile(filepath.Join(base, "respuestas-2026-10-14.json"))
		if err != nil {
			t.Fatal(err)
		}
		var cambios map[string]struct {
			Previa, Nueva string
			Sustituciones map[string]string
		}
		if err = json.Unmarshal(b, &cambios); err != nil {
			t.Fatal(err)
		}
		for tipo, c := range cambios {
			t.Run(idioma+"/"+tipo, func(t *testing.T) {
				previa, err := os.ReadFile(filepath.Join(base, tipo, "v"+c.Previa+".docx"))
				if err != nil {
					t.Fatal(err)
				}
				nueva, err := os.ReadFile(filepath.Join(base, tipo, "v"+c.Nueva+".docx"))
				if err != nil {
					t.Fatal(err)
				}
				if !maps.Equal(marcadoresRevision(t, previa), marcadoresRevision(t, nueva)) {
					t.Fatal("marcadores cambiados")
				}
				raiz, err := leerXML(partes(t, nueva)["word/document.xml"], nil)
				if err != nil {
					t.Fatal(err)
				}
				texto := textoNodo(raiz)
				if regexp.MustCompile(`2L|2J|2P|2M|2O`).MatchString(texto) {
					t.Fatal("numeración antigua en versión nueva")
				}
				if tipo == "spaf" {
					if strings.Contains(texto, "ZTS") || !(strings.Contains(texto, "1, 2, 7") && strings.Contains(texto, "8")) {
						t.Fatal("D18/D19 no aplicadas a SPAF")
					}
					b, err := Generar(nueva, Datos{Campos: map[string]string{"anio": "2026", "mes": "09", "equipo": "ETF 02", "altas": "1", "bajas": "0", "casos": "1", "menores": "1"}, Filas: map[string][]map[string]string{"casos_spaf": {{"numero": "02-01-26", "menores": "Nerea Romero Cano", "numero_menores": "1", "guardadores": "Clara Cano Ruiz", "municipio_zts": "Dúrcal", "caracteristicas": "", "derivacion": "", "servicio": "", "fechas": "", "fase": "", "temporalizacion": "", "finalidad_pronostico": "", "instrumentos": "", "cierres": ""}}}})
					if err != nil {
						t.Fatal(err)
					}
					r, err := leerXML(partes(t, b)["word/document.xml"], nil)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(textoNodo(r), "Dúrcal") || strings.Contains(textoNodo(r), "ZTS") {
						t.Fatal("Word SPAF debe contener sólo municipio")
					}
				}
			})
		}
	}
}
