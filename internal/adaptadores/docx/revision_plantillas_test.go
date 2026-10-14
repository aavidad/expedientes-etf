package docx

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRevisionTodasLasPlantillas publica documentos sólo si se pide un directorio
// de evidencia. El motor real rellena cada versión, incluidas las históricas.
func TestRevisionTodasLasPlantillas(t *testing.T) {
	b, err := os.ReadFile("testdata/revision_datos.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture Datos
	if err = json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	patron := regexp.MustCompile(`\{\{([#/]?[a-z][a-z0-9_.]*)\}\}`)
	total := 0
	err = filepath.WalkDir("../../../idiomas", func(ruta string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() || !strings.HasSuffix(ruta, ".docx") || !strings.Contains(ruta, string(filepath.Separator)+"plantillas"+string(filepath.Separator)) {
			return nil
		}
		total++
		t.Run(strings.TrimPrefix(ruta, "../../../idiomas/"), func(t *testing.T) {
			plantilla, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}

			if strings.HasSuffix(ruta, "v1.0.1.docx") || strings.HasSuffix(ruta, "v1.1.1.docx") {
				anterior := strings.TrimSuffix(ruta, "1.docx") + "0.docx"
				base, err := os.ReadFile(anterior)
				if err != nil {
					t.Fatal(err)
				}
				if !maps.Equal(marcadoresRevision(t, plantilla), marcadoresRevision(t, base)) {
					t.Fatal("contrato de marcadores cambiado")
				}
			}
			datos := Datos{Campos: fixture.Campos, Filas: map[string][]map[string]string{}}
			if strings.Contains(ruta, string(filepath.Separator)+"memoria"+string(filepath.Separator)) {
				datos.Campos = camposRevisionMemoria(t, ruta, patron)
			}
			z, err := zip.NewReader(bytes.NewReader(plantilla), int64(len(plantilla)))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range z.File {
				if !strings.HasSuffix(f.Name, ".xml") {
					continue
				}
				parte := leerParteRevision(t, f)
				raiz, err := leerXML(parte, nil)
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range raiz.buscar("p") {
					for _, marca := range patron.FindAllStringSubmatch(textoNodo(p), -1) {
						clave := marca[1]
						if strings.HasPrefix(clave, "#") {
							datos.Filas[clave[1:]] = fixture.Filas[clave[1:]]
						} else if !strings.HasPrefix(clave, "/") {
							if _, ok := datos.Campos[clave]; !ok {
								t.Fatalf("dato inventado ausente: %s", clave)
							}
						}
					}
				}
			}
			for _, filas := range []bool{false, true} {
				caso := datos
				if !filas {
					caso.Filas = map[string][]map[string]string{}
					for bloque := range datos.Filas {
						caso.Filas[bloque] = datos.Filas[bloque][:1]
					}
				}
				salida, err := Generar(plantilla, caso)
				if err != nil {
					t.Fatal(err)
				}
				generado, err := zip.NewReader(bytes.NewReader(salida), int64(len(salida)))
				if err != nil {
					t.Fatal(err)
				}
				for _, f := range generado.File {
					if !strings.HasSuffix(f.Name, ".xml") {
						continue
					}
					raiz, err := leerXML(leerParteRevision(t, f), nil)
					if err != nil {
						t.Fatal(err)
					}
					for _, p := range raiz.buscar("p") {
						texto := textoNodo(p)
						if strings.Contains(texto, "{{") || strings.Contains(texto, "}}") {
							t.Fatalf("marcador sin sustituir: %s", f.Name)
						}
					}
				}
				if filas {
					if dir := os.Getenv("ETF_DOCX_EVIDENCIA_DIR"); dir != "" && (strings.HasSuffix(ruta, "v1.0.1.docx") || strings.HasSuffix(ruta, "v1.1.1.docx")) {
						relativo, err := filepath.Rel("../../../idiomas", ruta)
						if err != nil {
							t.Fatal(err)
						}
						destino := filepath.Join(dir, relativo)
						if err = os.MkdirAll(filepath.Dir(destino), 0700); err != nil {
							t.Fatal(err)
						}
						if err = os.WriteFile(destino, salida, 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			t.Log("DOCX-OK: una y dos filas por bloque; ningún marcador pendiente")
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 58 {
		t.Fatalf("inventario cambiado: %d versiones, revisar cobertura", total)
	}
	t.Logf("PLANTILLAS-OK: %d versiones, %d generaciones", total, total*2)
}

func camposRevisionMemoria(t *testing.T, ruta string, patron *regexp.Regexp) map[string]string {
	t.Helper()
	b, err := os.ReadFile("testdata/revision_memoria_datos.json")
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]string
	if err = json.Unmarshal(b, &campos); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(strings.TrimSuffix(ruta, ".docx") + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var presentacion struct {
		Campos map[string]string `json:"campos"`
		Celdas []struct {
			Campo string `json:"campo"`
		} `json:"celdas"`
	}
	if err = json.Unmarshal(b, &presentacion); err != nil {
		t.Fatal(err)
	}
	for _, celda := range presentacion.Celdas {
		campos[celda.Campo] = "3"
	}
	for clave, texto := range presentacion.Campos {
		campos[clave] = patron.ReplaceAllStringFunc(texto, func(marca string) string {
			referencia := marca[2 : len(marca)-2]
			valor, ok := campos[referencia]
			if !ok {
				t.Fatalf("dato de Memoria ausente: %s", referencia)
			}
			return valor
		})
	}
	return campos
}

func leerParteRevision(t *testing.T, f *zip.File) []byte {
	t.Helper()
	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	cierre := r.Close()
	if err != nil || cierre != nil {
		t.Fatalf("lectura ZIP: %v; %v", err, cierre)
	}
	return b
}

// El contrato incluye todas las partes, también cabeceras y pies.
func marcadoresRevision(t *testing.T, b []byte) map[string]int {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	contrato := map[string]int{}
	patron := regexp.MustCompile(`\{\{([#/]?[a-z][a-z0-9_.]*)\}\}`)
	for _, f := range z.File {
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		raiz, err := leerXML(leerParteRevision(t, f), nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range raiz.buscar("p") {
			for _, marca := range patron.FindAllString(textoNodo(p), -1) {
				contrato[f.Name+":"+marca]++
			}
		}
	}
	return contrato
}
