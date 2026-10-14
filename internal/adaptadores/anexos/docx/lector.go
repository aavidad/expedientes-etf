// Package docx interpreta en memoria las versiones conocidas de los Anexos 3 y 4.
package docx

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
)

const MaxBytes = 10_000_000
const MaxDescomprimido = 32 << 20
const maxDocumento = 8 << 20

// Campo y Lectura conservan la estructura y códigos del ensayo PDF. Pagina es
// cero: DOCX no contiene una paginación estable. Los códigos repetidos conservan
// el orden de las filas, incluidos los valores vacíos.
type Campo struct {
	Codigo  string
	Valor   string
	Pagina  int
	Dudoso  bool
	Motivos []string
}

type Lectura struct {
	PlantillaReconocida bool
	Campos              []Campo
	Avisos              []string
	Tipo                string
	Version             string
}

//go:embed perfiles.json
var perfilesJSON string

type perfil struct {
	Tipo     string
	Version  string
	Titulos  []string
	Tablas   [][][][]string
	Reglas   []regla
	Estudios map[string]string
}

type regla struct {
	Codigo   string
	Tabla    int
	Fila     int
	Celda    int
	Control  int
	Etiqueta string
	Relleno  string
	Opciones []string
	Repetir  bool
	Modo     string
}

// Leer sólo procesa los bytes recibidos. Los errores son claves estables sin
// datos del documento y nunca van acompañados de una extracción parcial.
func Leer(b []byte) (Lectura, error) {
	partes, err := abrir(b)
	if err != nil {
		return Lectura{}, err
	}
	raiz, err := leerXML(partes["word/document.xml"])
	if err != nil {
		return Lectura{}, err
	}
	if !raiz.es(nsWord, "document") || len(raiz.directos("body")) != 1 {
		return Lectura{}, errors.New("anexo.docx_xml_invalido")
	}
	body := raiz.directos("body")[0]
	tablas := body.directos("tbl")
	var perfiles []perfil
	if err = json.Unmarshal([]byte(perfilesJSON), &perfiles); err != nil {
		return Lectura{}, errors.New("anexo.docx_perfil_invalido")
	}
	for _, p := range perfiles {
		if !reconocer(body, tablas, p) {
			continue
		}
		if len(tablas[3].directos("tr")) > 16 {
			return Lectura{}, errors.New("anexo.docx_demasiados_menores")
		}
		return extraer(tablas, p, partes)
	}
	return Lectura{}, errors.New("anexo.plantilla_desconocida")
}

func abrir(b []byte) (map[string][]byte, error) {
	if len(b) > MaxBytes {
		return nil, errors.New("anexo.docx_tamano_excedido")
	}
	if len(b) < 4 || !bytes.Equal(b[:4], []byte{'P', 'K', 3, 4}) {
		return nil, errors.New("anexo.docx_invalido")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, errors.New("anexo.docx_zip_invalido")
	}
	if len(z.File) > 256 {
		return nil, errors.New("anexo.docx_limite_zip")
	}
	partes := make(map[string][]byte)
	vistos := make(map[string]bool)
	var total uint64
	for _, f := range z.File {
		nombre := f.Name
		if nombre != path.Clean(nombre) || strings.HasPrefix(nombre, "/") || strings.HasPrefix(nombre, "../") || strings.Contains(nombre, "\\") || vistos[nombre] || !f.Mode().IsRegular() {
			return nil, errors.New("anexo.docx_zip_invalido")
		}
		vistos[nombre] = true
		if strings.Contains(strings.ToLower(nombre), "vbaproject") {
			return nil, errors.New("anexo.docx_macros")
		}
		if f.UncompressedSize64 > MaxDescomprimido || total > MaxDescomprimido-f.UncompressedSize64 {
			return nil, errors.New("anexo.docx_limite_descompresion")
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 1<<20 && (f.CompressedSize64 == 0 || f.UncompressedSize64/f.CompressedSize64 > 200) {
			return nil, errors.New("anexo.docx_limite_descompresion")
		}
		guardar := nombre == "word/document.xml" || nombre == "[Content_Types].xml" || nombre == "word/_rels/document.xml.rels" || (strings.HasPrefix(nombre, "word/activeX/") && strings.HasSuffix(nombre, ".xml"))
		limite := int64(f.UncompressedSize64)
		if guardar && limite > maxDocumento {
			return nil, errors.New("anexo.docx_limite_descompresion")
		}
		r, e := f.Open()
		if e != nil {
			return nil, errors.New("anexo.docx_zip_invalido")
		}
		var datos []byte
		var n int64
		if guardar {
			datos, e = io.ReadAll(io.LimitReader(r, limite+1))
			n = int64(len(datos))
		} else {
			n, e = io.Copy(io.Discard, io.LimitReader(r, limite+1))
		}
		cerrar := r.Close()
		if e != nil || cerrar != nil || n != limite {
			return nil, errors.New("anexo.docx_zip_invalido")
		}
		if guardar {
			partes[nombre] = datos
		}
	}
	if partes["word/document.xml"] == nil || partes["[Content_Types].xml"] == nil {
		return nil, errors.New("anexo.docx_invalido")
	}
	tipos, err := leerXML(partes["[Content_Types].xml"])
	if err != nil {
		return nil, err
	}
	const nsTipos = "http://schemas.openxmlformats.org/package/2006/content-types"
	ok := false
	for _, n := range tipos.buscar(nsTipos, "Override") {
		t := n.atributo("", "ContentType")
		if strings.Contains(strings.ToLower(t), "macroenabled") {
			return nil, errors.New("anexo.docx_macros")
		}
		if n.atributo("", "PartName") == "/word/document.xml" && t == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" {
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("anexo.docx_invalido")
	}
	return partes, nil
}

func reconocer(body *nodo, tablas []*nodo, p perfil) bool {
	if len(tablas) != len(p.Tablas) {
		return false
	}
	var titulos []string
	for _, n := range body.directos("p") {
		if s := normalizar(n.contenido(false)); s != "" {
			titulos = append(titulos, s)
		}
	}
	if len(titulos) < len(p.Titulos) {
		return false
	}
	for i, t := range p.Titulos {
		if titulos[i] != t {
			return false
		}
	}
	for ti, esperadas := range p.Tablas {
		filas := tablas[ti].directos("tr")
		if ti != 3 && len(filas) != len(esperadas) {
			return false
		}
		if ti == 3 && len(filas) < 1 {
			return false
		}
		for ri, f := range filas {
			indice := ri
			if ti == 3 && ri > 0 {
				indice = 1
			}
			if indice >= len(esperadas) {
				return false
			}
			celdas := f.directos("tc")
			if len(celdas) != len(esperadas[indice]) {
				return false
			}
			for ci, etiquetas := range esperadas[indice] {
				restante := normalizar(celdas[ci].contenido(true))
				for _, etiqueta := range etiquetas {
					posicion := strings.Index(restante, etiqueta)
					if posicion < 0 {
						return false
					}
					restante = restante[posicion+len(etiqueta):]
				}
			}
		}
	}
	return true
}
