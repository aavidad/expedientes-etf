package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxComprimido    = 10 << 20
	maxDescomprimido = 100 << 20
	maxEntradas      = 1000
	maxFilas         = 15
)

// Datos contiene valores ya formateados por el llamador. Controles se indexa
// por w:id de la versión de plantilla; Filas por el nombre de su bloque.
type Datos struct {
	Campos      map[string]string
	Filas       map[string][]map[string]string
	Controles   map[string]string
	LimiteFilas int
}

// Error permite traducir el fallo mediante el futuro puerto Textos. Campo es
// sólo un identificador de plantilla, nunca el valor personal suministrado.
type Error struct{ Clave, Campo string }

func (e *Error) Error() string {
	if e.Campo == "" {
		return e.Clave
	}
	return e.Clave + ":" + e.Campo
}
func fallo(clave, campo string) error { return &Error{Clave: clave, Campo: campo} }

// Generar no lee ni escribe ficheros, no ejecuta procesos y no crea emisiones.
// Conserva el orden ZIP y sus metadatos para reproducir los mismos bytes.
func Generar(plantilla []byte, datos Datos) ([]byte, error) {
	if len(plantilla) > maxComprimido {
		return nil, fallo("docx.limite_comprimido", "")
	}
	if err := validarDatos(datos); err != nil {
		return nil, err
	}
	z, err := zip.NewReader(bytes.NewReader(plantilla), int64(len(plantilla)))
	if err != nil {
		return nil, fallo("docx.zip_invalido", "")
	}
	if len(z.File) > maxEntradas {
		return nil, fallo("docx.limite_entradas", "")
	}
	partes := make(map[string][]byte, len(z.File))
	total := 0
	for _, f := range z.File {
		if _, existe := partes[f.Name]; existe {
			return nil, fallo("docx.parte_duplicada", "")
		}
		if strings.Contains(strings.ToLower(f.Name), "vbaproject") {
			return nil, fallo("docx.macros_no_admitidas", "")
		}
		if f.UncompressedSize64 > uint64(maxDescomprimido-total) {
			return nil, fallo("docx.limite_descomprimido", "")
		}
		r, err := f.Open()
		if err != nil {
			return nil, fallo("docx.zip_invalido", "")
		}
		contenido, err := io.ReadAll(io.LimitReader(r, int64(maxDescomprimido-total)+1))
		cerrar := r.Close()
		if err != nil || cerrar != nil {
			return nil, fallo("docx.zip_invalido", "")
		}
		total += len(contenido)
		if total > maxDescomprimido {
			return nil, fallo("docx.limite_descomprimido", "")
		}
		partes[f.Name] = contenido
	}
	for _, nombre := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		if len(partes[nombre]) == 0 {
			return nil, fallo("docx.estructura_invalida", "")
		}
	}
	original := partes["word/document.xml"]
	raiz, err := leerXML(original, nil)
	if err != nil {
		return nil, err
	}
	if !raiz.hijos[0].es("document") {
		return nil, fallo("docx.estructura_invalida", "")
	}
	presupuesto := &presupuestoXML{usado: len(original)}
	if err = rellenar(raiz, original, datos, presupuesto); err != nil {
		return nil, err
	}
	contenido := raiz.bytes(original)
	if len(contenido) > maxDescomprimido {
		return nil, fallo("docx.limite_resultado", "")
	}
	var salida bytes.Buffer
	w := zip.NewWriter(&salida)
	for _, f := range z.File {
		if f.Name != "word/document.xml" {
			if err = w.Copy(f); err != nil {
				return nil, fallo("docx.zip_invalido", "")
			}
			continue
		}
		cabecera := f.FileHeader
		r, err := w.CreateHeader(&cabecera)
		if err != nil {
			return nil, fallo("docx.zip_invalido", "")
		}
		if _, err = r.Write(contenido); err != nil {
			return nil, fallo("docx.zip_invalido", "")
		}
	}
	if err = w.Close(); err != nil {
		return nil, fallo("docx.zip_invalido", "")
	}
	return salida.Bytes(), nil
}

func validarDatos(datos Datos) error {
	total := 0
	validar := func(campos map[string]string) error {
		for _, campo := range claves(campos) {
			valor := campos[campo]
			if !utf8.ValidString(valor) {
				return fallo("docx.valor_xml_invalido", "")
			}
			for _, r := range valor {
				if !(r == '\t' || r == '\n' || r == '\r' || r >= 0x20 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff) {
					return fallo("docx.valor_xml_invalido", "")
				}
			}
			total += len(valor)
			if total > maxComprimido {
				return fallo("docx.limite_datos", "")
			}
		}
		return nil
	}
	if err := validar(datos.Campos); err != nil {
		return err
	}
	if err := validar(datos.Controles); err != nil {
		return err
	}
	for _, nombre := range claves(datos.Filas) {
		filas := datos.Filas[nombre]
		limite := maxFilas
		if datos.LimiteFilas < 0 || datos.LimiteFilas > 1000 {
			return fallo("docx.limite_filas", "")
		}
		if datos.LimiteFilas > 0 {
			limite = datos.LimiteFilas
		}
		if len(filas) > limite {
			return fallo("docx.limite_filas", "")
		}
		for _, fila := range filas {
			if err := validar(fila); err != nil {
				return err
			}
		}
	}
	return nil
}

func claves[V any](m map[string]V) []string {
	lista := make([]string, 0, len(m))
	for k := range m {
		lista = append(lista, k)
	}
	sort.Strings(lista)
	return lista
}

func rellenar(raiz *nodo, original []byte, datos Datos, presupuesto *presupuestoXML) error {
	// Sólo filas completas con apertura y cierre en la misma fila.
	for _, fila := range raiz.buscar("tr") {
		texto := textoNodo(fila)
		marcas := regexp.MustCompile(`\{\{#([a-z][a-z0-9_.]*)\}\}`).FindAllStringSubmatch(texto, -1)
		if len(marcas) == 0 {
			continue
		}
		if len(marcas) != 1 {
			return fallo("docx.bloque_invalido", "")
		}
		nombre := marcas[0][1]
		fin := "{{/" + nombre + "}}"
		if strings.Count(texto, fin) != 1 || strings.Index(texto, fin) < strings.Index(texto, marcas[0][0]) {
			return fallo("docx.bloque_invalido", nombre)
		}
		filas, existe := datos.Filas[nombre]
		if !existe {
			return fallo("docx.bloque_desconocido", nombre)
		}
		if len(filas) == 0 {
			// Sin valores de fila no puede validarse su esquema: cualquier campo
			// local exige una fila, incluso si su valor es vacío.
			return fallo("docx.bloque_sin_filas", nombre)
		}
		var repetidas bytes.Buffer
		for _, campos := range filas {
			copia := original[fila.inicio:fila.fin]
			if err := presupuesto.reservar(len(copia)); err != nil {
				return err
			}
			r, err := leerXML(copia, fila.espacios)
			if err != nil {
				return err
			}
			valores := make(map[string]string, len(datos.Campos)+len(campos)+2)
			for k, v := range datos.Campos {
				valores[k] = v
			}
			for k, v := range campos {
				valores[k] = v
			}
			valores["#"+nombre], valores["/"+nombre] = "", ""
			if len(r.buscar("sdt")) != 0 {
				return fallo("docx.control_en_fila_no_admitido", "")
			}
			if err = rellenarParrafos(r, copia, valores, presupuesto); err != nil {
				return err
			}
			repetidas.Write(r.bytes(copia))
		}
		fila.reemplazo = repetidas.Bytes()
	}
	if err := rellenarControles(raiz, original, datos.Controles, presupuesto); err != nil {
		return err
	}
	return rellenarParrafos(raiz, original, datos.Campos, presupuesto)
}

func textoNodo(n *nodo) string {
	var b strings.Builder
	for _, t := range n.buscar("t") {
		b.WriteString(t.texto)
	}
	return b.String()
}

func rellenarParrafos(n *nodo, original []byte, campos map[string]string, presupuesto *presupuestoXML) error {
	if n.reemplazo != nil {
		return nil
	}
	if n.es("p") {
		textos := n.buscar("t")
		texto := textoNodo(n)
		marcas := regexp.MustCompile(`\{\{([^{}]+)\}\}`).FindAllStringSubmatchIndex(texto, -1)
		residuo := regexp.MustCompile(`\{\{([^{}]+)\}\}`).ReplaceAllString(texto, "")
		if strings.Contains(residuo, "{{") || strings.Contains(residuo, "}}") {
			return fallo("docx.marcador_invalido", "")
		}
		tamano := len(texto)
		valores := make([]string, len(textos))
		inicios := make([]int, len(textos))
		pos := 0
		for i, t := range textos {
			valores[i], inicios[i] = t.texto, pos
			pos += len(t.texto)
		}
		for j := len(marcas) - 1; j >= 0; j-- {
			m := marcas[j]
			campo := texto[m[2]:m[3]]
			if !regexp.MustCompile(`^[#/]?[a-z][a-z0-9_.]*$`).MatchString(campo) {
				return fallo("docx.marcador_invalido", "")
			}
			valor, existe := campos[campo]
			if !existe {
				return fallo("docx.marcador_desconocido", campo)
			}
			tamano += len(valor) - (m[1] - m[0])
			if tamano > maxDescomprimido {
				return fallo("docx.limite_resultado", "")
			}
			for i, t := range textos {
				a, b := max(m[0], inicios[i]), min(m[1], inicios[i]+len(t.texto))
				if a >= b {
					continue
				}
				insertar := ""
				if a == m[0] {
					insertar = valor
				}
				valores[i] = valores[i][:a-inicios[i]] + insertar + valores[i][b-inicios[i]:]
			}
		}
		for i, t := range textos {
			if valores[i] != t.texto {
				if t.reemplazo != nil {
					return fallo("docx.control_con_marcador_no_admitido", "")
				}
				if err := t.ponerTexto(original, valores[i], presupuesto); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, h := range n.hijos {
		if err := rellenarParrafos(h, original, campos, presupuesto); err != nil {
			return err
		}
	}
	return nil
}

func rellenarControles(raiz *nodo, original []byte, valores map[string]string, presupuesto *presupuestoXML) error {
	encontrados := make(map[string]bool, len(valores))
	for _, control := range raiz.buscar("sdt") {
		pr := control.hijo("sdtPr")
		if pr == nil {
			continue
		}
		id := pr.hijo("id")
		if id == nil {
			continue
		}
		clave := id.atributo("val")
		valor, existe := valores[clave]
		if !existe {
			continue
		}
		if encontrados[clave] {
			return fallo("docx.control_duplicado", "")
		}
		encontrados[clave] = true
		contenido := control.hijo("sdtContent")
		if contenido == nil {
			return fallo("docx.control_sin_contenido", "")
		}
		textos := contenido.buscar("t")
		if len(textos) == 0 {
			return fallo("docx.control_sin_texto", "")
		}
		for i, t := range textos {
			v := ""
			if i == 0 {
				v = valor
			}
			if err := t.ponerTexto(original, v, presupuesto); err != nil {
				return err
			}
		}
		for _, h := range pr.hijos {
			if h.es("showingPlcHdr") || h.es("lock") {
				h.reemplazo = []byte{}
			}
			// El texto no debe ser sobrescrito por un enlace a customXml.
			if h.es("dataBinding") {
				return fallo("docx.control_vinculado_no_admitido", "")
			}
		}
	}
	for _, clave := range claves(valores) {
		if !encontrados[clave] {
			return fallo("docx.control_desconocido", "")
		}
	}
	return nil
}
