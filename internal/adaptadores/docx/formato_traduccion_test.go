package docx

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Compara todas las partes del paquete: sólo pueden variar el texto traducido,
// su espacio significativo, los rótulos de opciones y el idioma de corrección.
// También comprueba campos Word, relaciones, imágenes, estilos, tablas y runs.
func TestTraduccionesENConservanFormatoYMarcadores(t *testing.T) {
	base := "../../../idiomas/en/plantillas"
	archivos, err := filepath.Glob(filepath.Join(base, "traduccion-*.json"))
	if err != nil || len(archivos) == 0 {
		t.Fatalf("inventario de traducciones: %v", err)
	}
	total := 0
	for _, archivo := range archivos {
		b, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		var cambios map[string]struct {
			Previa, Nueva string
			Opciones      map[string]string
		}
		if err = json.Unmarshal(b, &cambios); err != nil {
			t.Fatal(err)
		}
		for tipo, c := range cambios {
			total++
			t.Run(tipo, func(t *testing.T) {
				leer := func(version string) []byte {
					b, err := os.ReadFile(filepath.Join(base, tipo, "v"+version+".docx"))
					if err != nil {
						t.Fatal(err)
					}
					return b
				}
				antes, despues := leer(c.Previa), leer(c.Nueva)
				if !maps.Equal(marcadoresRevision(t, antes), marcadoresRevision(t, despues)) {
					t.Fatal("contrato de marcadores cambiado")
				}
				if err := ValidarCambio(antes, despues); err != nil {
					t.Fatal(err)
				}
				a, d := partes(t, antes), partes(t, despues)
				if len(a) != len(d) {
					t.Fatal("partes del paquete cambiadas")
				}
				for nombre, original := range a {
					nuevo, ok := d[nombre]
					if !ok {
						t.Fatalf("parte ausente: %s", nombre)
					}
					if strings.HasPrefix(nombre, "word/") && strings.HasSuffix(nombre, ".xml") {
						if !reflect.DeepEqual(estructuraTraduccion(t, original, false, c.Opciones), estructuraTraduccion(t, nuevo, true, c.Opciones)) {
							t.Errorf("formato o campos cambiados: %s", nombre)
						}
					} else if !bytes.Equal(original, nuevo) {
						t.Errorf("parte no textual cambiada: %s", nombre)
					}
				}
			})
		}
	}
	if total != 12 {
		t.Fatalf("traducciones comprobadas: %d", total)
	}
}

const nsWordTraduccion = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

func estructuraTraduccion(t *testing.T, b []byte, exigirIngles bool, opciones map[string]string) []any {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(b))
	var salida []any
	enTexto := false
	for {
		token, err := d.Token()
		if err == io.EOF {
			return salida
		}
		if err != nil {
			t.Fatal(err)
		}
		switch v := token.(type) {
		case xml.StartElement:
			enTexto = v.Name.Space == nsWordTraduccion && v.Name.Local == "t"
			var attrs []xml.Attr
			for _, a := range v.Attr {
				if !exigirIngles && v.Name.Space == nsWordTraduccion && v.Name.Local == "listItem" && a.Name.Space == nsWordTraduccion && a.Name.Local == "displayText" {
					if traduccion, ok := opciones[a.Value]; ok {
						a.Value = traduccion
					}
				}
				if enTexto && a.Name.Space == "http://www.w3.org/XML/1998/namespace" && a.Name.Local == "space" {
					continue
				}
				if v.Name.Space == nsWordTraduccion && v.Name.Local == "lang" && a.Name.Space == nsWordTraduccion {
					if exigirIngles && a.Value != "en-GB" {
						t.Errorf("idioma de corrección: %s", a.Value)
					}
					a.Value = ""
				}
				attrs = append(attrs, a)
			}
			sort.Slice(attrs, func(i, j int) bool {
				return attrs[i].Name.Space+attrs[i].Name.Local < attrs[j].Name.Space+attrs[j].Name.Local
			})
			v.Attr = attrs
			salida = append(salida, v)
		case xml.EndElement:
			enTexto = false
			salida = append(salida, v)
		case xml.CharData:
			if !enTexto && strings.TrimSpace(string(v)) != "" {
				salida = append(salida, string(v))
			}
		case xml.Comment:
			salida = append(salida, string(v))
		}
	}
}
