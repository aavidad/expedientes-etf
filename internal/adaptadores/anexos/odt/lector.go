// Package odt lee en memoria los Anexos 3 y 4 convertidos por LibreOffice.
package odt

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx"
)

const MIME = "application/vnd.oasis.opendocument.text"
const nsOffice = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
const nsTable = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
const nsText = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
const nsForm = "urn:oasis:names:tc:opendocument:xmlns:form:1.0"
const nsDraw = "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"

//go:embed perfiles.json
var perfilesJSON []byte

//go:embed rellenos.json
var rellenosJSON []byte

type regla struct {
	Codigo                           string
	Tabla, Fila, Celda, Desde, Hasta int
	Prefijo, Relleno, Modo           string
	Opciones                         []string
	Repetir                          bool
}
type etiqueta struct {
	Tabla, Fila, Celda int
	Valor              string
}
type perfil struct {
	Etiquetas      []etiqueta
	Titulo         string
	Cabeceras      []string
	Escolarizacion map[string]string
	Tipo, Version  string
	Tablas         [][][]int
	Reglas         []regla
	Estudios       map[string]string
}
type nodo struct {
	nombre xml.Name
	attrs  []xml.Attr
	hijos  []*nodo
	partes []any
}

func (n *nodo) attr(ns, clave string) string {
	for _, a := range n.attrs {
		if a.Name.Space == ns && a.Name.Local == clave {
			return a.Value
		}
	}
	return ""
}
func (n *nodo) buscar(ns, clave string) []*nodo {
	var r []*nodo
	if n.nombre.Space == ns && n.nombre.Local == clave {
		r = append(r, n)
	}
	for _, h := range n.hijos {
		r = append(r, h.buscar(ns, clave)...)
	}
	return r
}
func (n *nodo) directos(ns, clave string) []*nodo {
	var r []*nodo
	for _, h := range n.hijos {
		if h.nombre.Space == ns && h.nombre.Local == clave {
			r = append(r, h)
		}
	}
	return r
}
func (n *nodo) texto(casillas map[string]string) string {
	if n.nombre.Space == nsText {
		switch n.nombre.Local {
		case "s", "tab", "line-break":
			return " "
		case "note":
			return ""
		}
	}
	if n.nombre.Space == nsDraw && n.nombre.Local == "control" {
		if estado, ok := casillas[n.attr(nsDraw, "control")]; ok {
			return estado + " "
		}
		return "? "
	}
	var s strings.Builder
	for _, parte := range n.partes {
		switch v := parte.(type) {
		case string:
			s.WriteString(v)
		case *nodo:
			s.WriteString(v.texto(casillas))
		}
	}
	return s.String()
}

// Validar comprueba todo el contenedor, incluso las partes que no se extraen.
// No ejecuta controles, macros ni enlaces, ni escribe recursos en el disco.
func Validar(b []byte) error { _, err := abrir(b); return err }
func abrir(b []byte) (*nodo, error) {
	if len(b) > docx.MaxBytes {
		return nil, errors.New("anexo.odt_tamano_excedido")
	}
	if !bytes.HasPrefix(b, []byte{'P', 'K', 3, 4}) {
		return nil, errors.New("anexo.odt_invalido")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, errors.New("anexo.odt_zip_invalido")
	}
	if len(z.File) > 256 {
		return nil, errors.New("anexo.odt_limite_zip")
	}
	vistos := map[string]bool{}
	var total uint64
	var raiz *nodo
	tipo, manifest := false, false
	for _, f := range z.File {
		nombre := f.Name
		canon := strings.TrimSuffix(nombre, "/")
		if canon == "." || canon != path.Clean(canon) || strings.HasPrefix(canon, "/") || strings.HasPrefix(canon, "../") || strings.ContainsAny(nombre, "\\\x00:") || vistos[nombre] || (!f.Mode().IsRegular() && !f.FileInfo().IsDir()) {
			return nil, errors.New("anexo.odt_zip_invalido")
		}
		vistos[nombre] = true
		if !permitida(nombre) {
			return nil, errors.New("anexo.odt_contenido_inseguro")
		}
		size := f.UncompressedSize64
		if size > docx.MaxDescomprimido || total > docx.MaxDescomprimido-size || (size > 1<<20 && (f.CompressedSize64 == 0 || size/f.CompressedSize64 > 200)) || (strings.HasSuffix(nombre, ".xml") && size > 8<<20) {
			return nil, errors.New("anexo.odt_limite_descompresion")
		}
		total += size
		r, e := f.Open()
		if e != nil {
			return nil, errors.New("anexo.odt_zip_invalido")
		}
		datos, e := io.ReadAll(io.LimitReader(r, int64(size)+1))
		cierre := r.Close()
		if e != nil || cierre != nil || uint64(len(datos)) != size {
			clear(datos)
			return nil, errors.New("anexo.odt_zip_invalido")
		}
		if nombre == "mimetype" {
			tipo = string(datos) == MIME
		}
		if strings.HasSuffix(nombre, ".xml") || nombre == "manifest.rdf" {
			n, e := leerXML(datos)
			clear(datos)
			if e != nil {
				return nil, e
			}
			if nombre == "content.xml" {
				raiz = n
			}
			if nombre == "META-INF/manifest.xml" {
				manifest = true
			}
		} else {
			clear(datos)
		}
	}
	if !tipo || !manifest || raiz == nil || raiz.nombre != (xml.Name{Space: nsOffice, Local: "document-content"}) || len(raiz.buscar(nsOffice, "text")) != 1 {
		return nil, errors.New("anexo.odt_invalido")
	}
	return raiz, nil
}
func permitida(n string) bool {
	if strings.HasSuffix(n, "/") {
		return strings.HasPrefix(n, "Configurations2/")
	}
	switch n {
	case "mimetype", "content.xml", "styles.xml", "meta.xml", "settings.xml", "META-INF/manifest.xml", "manifest.rdf", "layout-cache", "Thumbnails/thumbnail.png":
		return true
	}
	if strings.HasPrefix(n, "Pictures/") {
		switch strings.ToLower(path.Ext(n)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".wmf", ".emf":
			return true
		}
	}
	return false
}
func leerXML(b []byte) (*nodo, error) {
	if err := docx.RevisarMarcado(b); err != nil {
		return nil, errors.New("anexo.odt_limite_xml")
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	var pila []*nodo
	var raiz *nodo
	for count := 0; ; count++ {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("anexo.odt_xml_invalido")
		}
		if count > 500000 {
			return nil, errors.New("anexo.odt_limite_xml")
		}
		switch v := t.(type) {
		case xml.Directive:
			return nil, errors.New("anexo.odt_contenido_inseguro")
		case xml.StartElement:
			if len(pila) >= 64 {
				return nil, errors.New("anexo.odt_limite_xml")
			}
			if v.Name.Local == "script" || v.Name.Local == "encryption-data" {
				return nil, errors.New("anexo.odt_contenido_inseguro")
			}
			for _, a := range v.Attr {
				if a.Name.Space == nsTable && (a.Name.Local == "number-rows-repeated" || a.Name.Local == "number-columns-repeated") && a.Value != "1" {
					return nil, errors.New("anexo.odt_limite_xml")
				}
			}
			n := &nodo{nombre: v.Name, attrs: v.Attr}
			if len(pila) > 0 {
				p := pila[len(pila)-1]
				p.hijos = append(p.hijos, n)
				p.partes = append(p.partes, n)
			} else {
				if raiz != nil {
					return nil, errors.New("anexo.odt_xml_invalido")
				}
				raiz = n
			}
			pila = append(pila, n)
		case xml.EndElement:
			pila = pila[:len(pila)-1]
		case xml.CharData:
			if len(pila) > 0 {
				p := pila[len(pila)-1]
				p.partes = append(p.partes, string(v))
			} else if strings.TrimSpace(string(v)) != "" {
				return nil, errors.New("anexo.odt_xml_invalido")
			}
		}
	}
	if raiz == nil || len(pila) != 0 {
		return nil, errors.New("anexo.odt_xml_invalido")
	}
	return raiz, nil
}

func Leer(b []byte) (docx.Lectura, error) {
	raiz, err := abrir(b)
	if err != nil {
		return docx.Lectura{}, err
	}
	estados := map[string]string{}
	for _, c := range raiz.buscar(nsForm, "checkbox") {
		estado := "☐"
		switch c.attr(nsForm, "current-state") {
		case "checked":
			estado = "☒"
		case "", "unchecked":
		default:
			estado = "?"
		}
		estados[c.attr(nsForm, "id")] = estado
	}
	var tablas [][][][]string
	for _, t := range raiz.buscar(nsTable, "table") {
		var filas [][][]string
		for _, f := range t.directos(nsTable, "table-row") {
			var celdas [][]string
			for _, c := range f.directos(nsTable, "table-cell") {
				var ps []string
				for _, p := range c.buscar(nsText, "p") {
					ps = append(ps, normalizar(p.texto(estados)))
				}
				celdas = append(celdas, ps)
			}
			filas = append(filas, celdas)
		}
		tablas = append(tablas, filas)
	}
	tipo := ""
	var perfiles []perfil
	if json.Unmarshal(perfilesJSON, &perfiles) != nil {
		return docx.Lectura{}, errors.New("anexo.odt_perfil_invalido")
	}
	for _, p := range raiz.buscar(nsText, "p") {
		for _, perfil := range perfiles {
			if normalizar(p.texto(nil)) == perfil.Titulo {
				tipo = perfil.Tipo
			}
		}
	}

	return ExtraerTablas(tablas, tipo)
}
func normalizar(s string) string { return strings.Join(strings.Fields(s), " ") }

// ExtraerTablas mantiene los códigos del lector Word y descarta filas sin persona.
func ExtraerTablas(tablas [][][][]string, tipo string) (docx.Lectura, error) {
	var perfiles []perfil
	if err := json.Unmarshal(perfilesJSON, &perfiles); err != nil {
		return docx.Lectura{}, errors.New("anexo.odt_perfil_invalido")
	}
	for _, p := range perfiles {
		if p.Tipo != tipo || len(tablas) != len(p.Tablas) {
			continue
		}
		correcto := true
		for i, t := range tablas {
			if (i != 3 && len(t) != len(p.Tablas[i])) || (i == 3 && len(t) < 1) {
				correcto = false
				break
			}
			for j, row := range t {
				ref := j
				if i == 3 && ref > 1 {
					ref = 1
				}
				if len(row) != p.Tablas[i][ref][0] {
					correcto = false
					break
				}
			}
		}
		for _, et := range p.Etiquetas {
			if correcto && normalizar(strings.Join(tablas[et.Tabla][et.Fila][et.Celda], " ")) != et.Valor {
				correcto = false
			}
		}
		if !correcto {
			continue
		}
		menores := 0
		for _, fila := range tablas[3][1:] {
			if !filaVacia(fila, 3) {
				menores++
			}
		}
		if menores > 15 {
			return docx.Lectura{}, errors.New("anexo.docx_demasiados_menores")
		}
		// Las cabeceras se verifican aparte de las dimensiones para evitar atribuir
		// campos a un documento de otro tipo con el mismo número de tablas.
		if normalizar(strings.Join(tablas[0][0][0], " ")) != p.Cabeceras[0] || normalizar(strings.Join(tablas[3][0][0], " ")) != p.Cabeceras[1] {
			continue
		}
		l := docx.Lectura{PlantillaReconocida: true, Tipo: tipo, Version: p.Version, Avisos: []string{"anexo.confirmar_odt"}}
		for i := 0; i < len(p.Reglas); {
			r := p.Reglas[i]
			fin := i + 1
			if r.Repetir {
				for fin < len(p.Reglas) && p.Reglas[fin].Repetir && p.Reglas[fin].Tabla == r.Tabla {
					fin++
				}
			}
			hasta := r.Fila + 1
			if r.Repetir {
				hasta = len(tablas[r.Tabla])
			}
			for fila := r.Fila; fila < hasta; fila++ {
				if (r.Tabla == 3 || r.Tabla == 5 || r.Tabla == 6) && filaVacia(tablas[r.Tabla][fila], r.Tabla) {
					continue
				}
				for _, rr := range p.Reglas[i:fin] {
					cell := tablas[rr.Tabla][fila][rr.Celda]
					desde, final := rr.Desde, rr.Hasta
					if desde > len(cell) {
						desde = len(cell)
					}
					if final > len(cell) {
						final = len(cell)
					}
					v := normalizar(strings.Join(cell[desde:final], " "))
					v = strings.TrimSpace(strings.TrimPrefix(v, rr.Prefijo))
					motivo := ""
					if rr.Modo == "casillas" {
						v, motivo = seleccion(v, rr.Opciones)
					}
					if relleno(v, rr.Relleno) {
						v = ""
						motivo = "anexo.texto_relleno"
					}
					if strings.HasSuffix(rr.Codigo, "nacimiento") || strings.HasSuffix(rr.Codigo, "fecha") || strings.Contains(rr.Codigo, ".fecha_") {
						v, motivo = fecha(v)
					}
					if rr.Codigo == "servicio.nombre" && strings.HasPrefix(v, "iServicios") {
						v = strings.TrimPrefix(v, "i")
					}
					if len(rr.Opciones) > 0 && v != "" {
						permitido := false
						for _, op := range rr.Opciones {
							if normalizar(op) == v {
								permitido = true
							}
						}
						if !permitido {
							motivo = "anexo.valor_fuera_lista"
						}
					}
					if rr.Codigo == "menores.persona.escolarizacion" {
						if destino, ok := p.Escolarizacion[v]; ok {
							v = destino
						}
					}
					if rr.Codigo == "sociodemografia.adulto.estudios" && v != "" {
						if destino, ok := p.Estudios[v]; ok {
							v = destino
						} else {
							motivo = "anexo.equivalencia_ausente"
						}
					}
					if v == "" && motivo == "" {
						motivo = "anexo.campo_ausente"
					}
					c := docx.Campo{Codigo: rr.Codigo, Valor: v, Dudoso: motivo != ""}
					if motivo != "" {
						c.Motivos = []string{motivo}
					}
					l.Campos = append(l.Campos, c)
				}
			}
			i = fin
		}
		return l, nil
	}
	return docx.Lectura{}, errors.New("anexo.plantilla_desconocida")
}
func filaVacia(row [][]string, tabla int) bool {
	inicio := 0
	if tabla == 5 {
		inicio = 1
	}
	for _, cell := range row[inicio:] {
		v := normalizar(strings.Join(cell, " "))
		if v != "" && !relleno(v, "") {
			return false
		}
	}
	return true
}
func relleno(v, propio string) bool {
	v = strings.ToLower(normalizar(v))
	if propio != "" && v == strings.ToLower(normalizar(propio)) {
		return true
	}
	var rellenos []string
	if json.Unmarshal(rellenosJSON, &rellenos) == nil {
		for _, r := range rellenos {
			if v == r {
				return true
			}
		}
	}
	return false
}
func fecha(v string) (string, string) {
	if v == "" {
		return "", "anexo.campo_ausente"
	}
	if len(v) > 0 && (v[0] == 'S' || v[0] == 'd') {
		v = v[1:]
	}
	layout := "02/01/2006"
	if len(v) == 8 {
		layout = "02/01/06"
	}
	t, err := time.Parse(layout, v)
	if err != nil {
		return v, "anexo.fecha_invalida"
	}
	return t.Format("02/01/2006"), ""
}
func seleccion(v string, opciones []string) (string, string) {
	elegido := ""
	encontradas := 0
	for _, opcion := range opciones {
		pos := strings.Index(v, opcion)
		if pos < 0 {
			continue
		}
		antes := strings.TrimSpace(v[:pos])
		if antes == "" {
			continue
		}
		ultima := []rune(antes)
		marca := ultima[len(ultima)-1]
		if marca == '☒' {
			if elegido != "" {
				return "", "anexo.valor_ambiguo"
			}
			elegido = opcion
			encontradas++
		} else if marca == '☐' {
			encontradas++
		}
	}
	if encontradas != len(opciones) {
		return "", "anexo.casilla_no_legible"
	}
	if elegido == "" {
		return "", "anexo.campo_ausente"
	}
	return elegido, ""
}

// NormalizarCampo aplica las equivalencias de la plantilla a texto sin controles.
// PDF conserva siempre la confirmación humana, incluso tras esta normalización.
func NormalizarCampo(codigo, valor string) string {
	valor = normalizar(valor)
	var perfiles []perfil
	if json.Unmarshal(perfilesJSON, &perfiles) == nil {
		for _, p := range perfiles {
			for _, r := range p.Reglas {
				if r.Codigo == codigo && relleno(valor, r.Relleno) {
					return ""
				}
			}
		}
	}
	if relleno(valor, "") {
		return ""
	}
	if strings.HasSuffix(codigo, "nacimiento") || strings.HasSuffix(codigo, "fecha") || strings.Contains(codigo, ".fecha_") {
		v, _ := fecha(valor)
		return v
	}
	if codigo == "servicio.nombre" && strings.HasPrefix(valor, "iServicios") {
		return strings.TrimPrefix(valor, "i")
	}
	if codigo == "menores.persona.escolarizacion" && len(perfiles) > 0 {
		if v, ok := perfiles[0].Escolarizacion[valor]; ok {
			return v
		}
	}
	if codigo == "sociodemografia.adulto.estudios" {
		var ps []perfil
		if json.Unmarshal(perfilesJSON, &ps) == nil {
			if v, ok := ps[0].Estudios[valor]; ok {
				return v
			}
		}
	}
	return valor
}

// Codigos devuelve los campos de la variante reconocida; PDF usa la misma lista.
func Codigos(tipo string) map[string]bool {
	var ps []perfil
	permitidos := map[string]bool{}
	if json.Unmarshal(perfilesJSON, &ps) == nil {
		for _, p := range ps {
			if p.Tipo == tipo {
				for _, r := range p.Reglas {
					permitidos[r.Codigo] = true
				}
			}
		}
	}
	return permitidos
}
