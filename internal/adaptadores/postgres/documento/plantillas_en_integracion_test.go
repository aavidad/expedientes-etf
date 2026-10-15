package documento_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

type cambioTraduccion struct{ Previa, Nueva string }

func traduccionesEN(t *testing.T) map[string]cambioTraduccion {
	t.Helper()
	resultado := map[string]cambioTraduccion{}
	for _, nombre := range []string{"traduccion-entrada.json", "traduccion-salida-spaf.json", "traduccion-residuos.json"} {
		var cambios map[string]cambioTraduccion
		leerDatosTraduccion(t, nombre, &cambios)
		for tipo, cambio := range cambios {
			if _, repetido := resultado[tipo]; repetido {
				t.Fatalf("tipo repetido: %s", tipo)
			}
			resultado[tipo] = cambio
		}
	}
	return resultado
}

func leerDatosTraduccion(t *testing.T, nombre string, destino any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../../../idiomas/en/plantillas", nombre))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, destino); err != nil {
		t.Fatal(err)
	}
}

// Se concatena el texto de cada párrafo para reconocer también palabras y
// marcadores partidos entre varios runs. Incluye cabeceras y glossary.
func textosWordEN(t *testing.T, b []byte) map[string]string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	partes := map[string]string{}
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, "word/") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		dec := xml.NewDecoder(r)
		var texto strings.Builder
		for {
			token, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				r.Close()
				t.Fatal(err)
			}
			switch token := token.(type) {
			case xml.StartElement:
				if token.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" && token.Name.Local == "listItem" {
					for _, a := range token.Attr {
						if a.Name.Local == "displayText" {
							texto.WriteString(a.Value)
							texto.WriteByte('\n')
						}
					}
				}
				if token.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" && token.Name.Local == "t" {
					var valor string
					if err := dec.DecodeElement(&valor, &token); err != nil {
						r.Close()
						t.Fatal(err)
					}
					texto.WriteString(valor)
				}
			case xml.EndElement:
				if token.Name.Local == "p" {
					texto.WriteByte('\n')
				}
			}
		}
		r.Close()
		partes[f.Name] = texto.String()
	}
	return partes
}

func generarYComprobarEN(t *testing.T, plantilla []byte) []byte {
	t.Helper()
	var reglas struct{ Prohibidas, Permitidas []string }
	leerDatosTraduccion(t, "verificacion-traduccion.json", &reglas)
	if len(reglas.Prohibidas) == 0 {
		t.Fatal("detector sin frases")
	}
	marcas := regexp.MustCompile(`\{\{([#/]?[a-z][a-z0-9_.]*)\}\}`)
	relleno := d.Relleno{Campos: map[string]string{}, Filas: map[string][]map[string]string{}}
	for _, texto := range textosWordEN(t, plantilla) {
		for _, marca := range marcas.FindAllStringSubmatch(texto, -1) {
			switch clave := marca[1]; clave[0] {
			case '#':
				relleno.Filas[clave[1:]] = []map[string]string{{}}
			case '/':
			default:
				relleno.Campos[clave] = "314159"
			}
		}
	}
	resultado, err := (word.Motor{}).Generar(t.Context(), plantilla, relleno)
	if err != nil {
		t.Fatal(err)
	}
	for parte, texto := range textosWordEN(t, resultado) {
		if strings.Contains(texto, "{{") || strings.Contains(texto, "}}") {
			t.Fatalf("marcadores pendientes: %s", parte)
		}
		texto = strings.ToLower(texto)
		for _, permitida := range reglas.Permitidas {
			texto = strings.ReplaceAll(texto, strings.ToLower(permitida), "")
		}
		for _, prohibida := range reglas.Prohibidas {
			if strings.Contains(texto, strings.ToLower(prohibida)) {
				t.Fatalf("texto castellano %q en %s", prohibida, parte)
			}
		}
	}
	return resultado
}

func TestPlantillasENTraducidasGeneracionReal(t *testing.T) {
	for tipo, cambio := range traduccionesEN(t) {
		t.Run(tipo, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("../../../../idiomas/en/plantillas", tipo, "v"+cambio.Nueva+".docx"))
			if err != nil {
				t.Fatal(err)
			}
			resultado := generarYComprobarEN(t, b)
			if directorio := os.Getenv("ETF_DOCX_EVIDENCIA_DIR"); directorio != "" {
				destino := filepath.Join(directorio, "en", tipo, "v"+cambio.Nueva+".docx")
				if err := os.MkdirAll(filepath.Dir(destino), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(destino, resultado, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
