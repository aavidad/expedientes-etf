package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"maps"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

// DiferenciaMarcadores identifica el campo y su bloque dentro del contrato.
type DiferenciaMarcadores struct{ Faltan, Sobran []string }

func (*DiferenciaMarcadores) Error() string { return "docx.marcador_invalido" }

// ValidarCambio conserva el contrato de campos y bloques de la versión actual.
// El relleno vacío ensaya además las reglas estructurales del mismo motor.
func ValidarCambio(nueva, actual []byte) error {
	marcas, err := contratoPlantilla(nueva)
	if err != nil {
		return err
	}
	previos, err := marcadoresHistoricos(actual)
	if err != nil {
		return err
	}
	if !maps.Equal(previos, marcas) {
		faltan, sobran := []string{}, []string{}
		for clave := range previos {
			if !marcas[clave] {
				faltan = append(faltan, clave)
			}
		}
		for clave := range marcas {
			if !previos[clave] {
				sobran = append(sobran, clave)
			}
		}
		slices.Sort(faltan)
		slices.Sort(sobran)
		return &DiferenciaMarcadores{Faltan: faltan, Sobran: sobran}
	}
	datos := Datos{Campos: map[string]string{}, Filas: map[string][]map[string]string{}}
	for clave := range marcas {
		bloque, marca, _ := strings.Cut(clave, ":")
		if strings.HasPrefix(marca, "#") {
			if datos.Filas[marca[1:]] == nil {
				datos.Filas[marca[1:]] = []map[string]string{{}}
			}
			continue
		}
		if !strings.HasPrefix(marca, "/") {
			if bloque == "" {
				datos.Campos[marca] = ""
			} else {
				if datos.Filas[bloque] == nil {
					datos.Filas[bloque] = []map[string]string{{}}
				}
				datos.Filas[bloque][0][marca] = ""
			}
		}
	}
	b, err := Generar(nueva, datos)
	clear(b)
	return err
}

func contratoPlantilla(b []byte) (map[string]bool, error) {
	if len(b) > maxComprimido {
		return nil, fallo("docx.limite_comprimido", "")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil || len(z.File) > maxEntradas {
		return nil, fallo("docx.zip_invalido", "")
	}
	marcas := map[string]bool{}
	vistos := map[string]bool{}
	total := 0
	main := false
	tipos, defectos := map[string]string{}, map[string]string{}
	officeDocument := false
	nombres := map[string]bool{}
	for _, f := range z.File {
		nombres[f.Name] = true
	}
	for _, f := range z.File {
		nombre := strings.ToLower(f.Name)
		if !fs.ValidPath(f.Name) || vistos[f.Name] || !f.Mode().IsRegular() || strings.Contains(nombre, "vbaproject") || strings.HasPrefix(nombre, "word/embeddings/") || strings.HasPrefix(nombre, "word/activex/") {
			return nil, fallo("docx.estructura_invalida", "")
		}
		vistos[f.Name] = true
		if f.UncompressedSize64 > uint64(maxDescomprimido-total) {
			return nil, fallo("docx.limite_descomprimido", "")
		}
		r, err := f.Open()
		if err != nil {
			return nil, fallo("docx.zip_invalido", "")
		}
		parte, err := io.ReadAll(io.LimitReader(r, int64(maxDescomprimido-total)+1))
		cerrar := r.Close()
		total += len(parte)
		if err != nil || cerrar != nil || total > maxDescomprimido {
			clear(parte)
			return nil, fallo("docx.zip_invalido", "")
		}
		if strings.HasSuffix(nombre, ".xml") || strings.HasSuffix(nombre, ".rels") {
			raiz, err := leerXML(parte, nil)
			clear(parte)
			if err != nil {
				return nil, err
			}
			if f.Name == "[Content_Types].xml" {
				if raiz.hijos[0].nombre.Local != "Types" || raiz.hijos[0].uri != "http://schemas.openxmlformats.org/package/2006/content-types" {
					return nil, fallo("docx.estructura_invalida", "")
				}
				partes, extensiones := map[string]bool{}, map[string]bool{}
				for _, n := range raiz.hijos[0].hijos {
					if n.uri != raiz.hijos[0].uri || (n.nombre.Local != "Override" && n.nombre.Local != "Default") || len(n.hijos) != 0 {
						return nil, fallo("docx.estructura_invalida", "")
					}
					var tipo, parte, extension string
					for _, a := range n.atributos {
						switch a.Name.Local {
						case "ContentType":
							tipo = a.Value
						case "PartName":
							parte = a.Value
						case "Extension":
							extension = a.Value
						}
					}
					if tipo == "" {
						return nil, fallo("docx.estructura_invalida", "")
					}
					if n.nombre.Local == "Override" {
						if !strings.HasPrefix(parte, "/") || !nombres[strings.TrimPrefix(parte, "/")] || partes[parte] {
							return nil, fallo("docx.estructura_invalida", "")
						}
						partes[parte] = true
						tipos[strings.TrimPrefix(parte, "/")] = tipo
						if parte == "/word/document.xml" && tipo == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" {
							main = true
						}
					} else {
						if extension == "" || extensiones[extension] {
							return nil, fallo("docx.estructura_invalida", "")
						}
						extensiones[extension] = true
						defectos[extension] = tipo
					}
				}
			}
			if strings.HasSuffix(nombre, ".rels") {
				const relaciones = "http://schemas.openxmlformats.org/package/2006/relationships"
				if raiz.hijos[0].nombre.Local != "Relationships" || raiz.hijos[0].uri != relaciones {
					return nil, fallo("docx.estructura_invalida", "")
				}
				ids := map[string]bool{}
				for _, n := range raiz.hijos[0].hijos {
					if n.uri != relaciones || n.nombre.Local != "Relationship" || len(n.hijos) != 0 {
						return nil, fallo("docx.estructura_invalida", "")
					}
					var id, tipo, target, modo string
					atributos := map[string]bool{}
					for _, a := range n.atributos {
						if a.Name.Space != "" || atributos[a.Name.Local] {
							return nil, fallo("docx.estructura_invalida", "")
						}
						atributos[a.Name.Local] = true
						switch a.Name.Local {
						case "Id":
							id = a.Value
						case "Type":
							tipo = a.Value
						case "Target":
							target = a.Value
						case "TargetMode":
							modo = a.Value
						}
					}
					if id == "" || ids[id] || tipo == "" || target == "" || contenidoActivo(tipo) {
						return nil, fallo("docx.estructura_invalida", "")
					}
					ids[id] = true
					if modo == "External" {
						if tipo != "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" || !enlaceHTTPS(target) {
							return nil, fallo("docx.estructura_invalida", "")
						}
						continue
					}
					base := path.Dir(path.Dir(f.Name))
					destino := path.Clean(path.Join(base, target))
					if strings.HasPrefix(target, "/") {
						destino = strings.TrimPrefix(target, "/")
					}
					if (modo != "" && modo != "Internal") || strings.ContainsAny(target, `\:`) || strings.HasPrefix(target, "//") || !fs.ValidPath(destino) || !nombres[destino] {
						return nil, fallo("docx.estructura_invalida", "")
					}
					if f.Name == "_rels/.rels" && tipo == "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" {
						if destino != "word/document.xml" || officeDocument {
							return nil, fallo("docx.estructura_invalida", "")
						}
						officeDocument = true
					}
				}
			}
			if err := validarCamposWord(raiz); err != nil {
				return nil, err
			}
			var revisar func(*nodo, string) error
			revisar = func(n *nodo, bloque string) error {
				if n.es("fldSimple") || n.es("altChunk") || n.es("object") || n.es("control") || (n.uri == "urn:schemas-microsoft-com:office:office" && n.nombre.Local == "OLEObject") || n.uri == "http://schemas.microsoft.com/office/2006/activeX" {
					return fallo("docx.estructura_invalida", "")
				}
				for _, a := range n.atributos {
					if a.Name.Local == "ContentType" && contenidoActivo(a.Value) {
						return fallo("docx.macros_no_admitidas", "")
					}
				}
				for _, h := range n.hijos {
					if err := revisar(h, bloque); err != nil {
						return err
					}
				}
				return nil
			}
			if err := revisar(raiz, ""); err != nil {
				return nil, err
			}
			if err := extraerMarcadores(raiz, f.Name, marcas); err != nil {
				return nil, err
			}
		} else {
			clear(parte)
		}
	}
	for nombre := range nombres {
		if nombre == "[Content_Types].xml" {
			continue
		}
		tipo := tipos[nombre]
		if tipo == "" {
			tipo = defectos[strings.TrimPrefix(path.Ext(nombre), ".")]
		}
		if tipo == "" || (strings.HasSuffix(nombre, ".rels") && tipo != "application/vnd.openxmlformats-package.relationships+xml") {
			return nil, fallo("docx.estructura_invalida", "")
		}
		// Sólo las partes .xml y .rels pasan la revisión de campos, OLE y
		// altChunk: una parte declarada como XML con otra extensión se
		// rechaza para que no llegue sin revisar al documento emitido.
		minusculas := strings.ToLower(nombre)
		esXML := strings.HasSuffix(tipo, "+xml") || tipo == "application/xml" || tipo == "text/xml"
		if esXML && !strings.HasSuffix(minusculas, ".xml") && !strings.HasSuffix(minusculas, ".rels") {
			return nil, fallo("docx.estructura_invalida", "")
		}
	}
	if !main || !officeDocument || !vistos["word/document.xml"] || !vistos["_rels/.rels"] || len(marcas) == 0 {
		return nil, fallo("docx.estructura_invalida", "")
	}
	return marcas, nil
}

// La versión histórica sólo aporta el contrato del documento principal. No se
// ejecuta ni se publica, y sus relaciones y objetos no condicionan la candidata.
func marcadoresHistoricos(b []byte) (map[string]bool, error) {
	if len(b) > maxComprimido {
		return nil, fallo("docx.limite_comprimido", "")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil || len(z.File) > maxEntradas {
		return nil, fallo("docx.zip_invalido", "")
	}
	var documento *zip.File
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			if documento != nil || !f.Mode().IsRegular() || f.UncompressedSize64 > maxDescomprimido {
				return nil, fallo("docx.estructura_invalida", "")
			}
			documento = f
		}
	}
	if documento == nil {
		return nil, fallo("docx.estructura_invalida", "")
	}
	r, err := documento.Open()
	if err != nil {
		return nil, fallo("docx.zip_invalido", "")
	}
	parte, err := io.ReadAll(io.LimitReader(r, maxDescomprimido+1))
	cerrar := r.Close()
	defer clear(parte)
	if err != nil || cerrar != nil || len(parte) > maxDescomprimido {
		return nil, fallo("docx.zip_invalido", "")
	}
	raiz, err := leerXML(parte, nil)
	if err != nil {
		return nil, err
	}
	marcas := map[string]bool{}
	if err := extraerMarcadores(raiz, documento.Name, marcas); err != nil {
		return nil, err
	}
	return marcas, nil
}

func extraerMarcadores(raiz *nodo, parte string, marcas map[string]bool) error {
	patron := regexp.MustCompile(`\{\{([#/]?[a-z][a-z0-9_.]*)\}\}`)
	var visitar func(*nodo, string) error
	visitar = func(n *nodo, bloque string) error {
		if n.es("tr") {
			for _, m := range patron.FindAllStringSubmatch(textoNodo(n), -1) {
				if strings.HasPrefix(m[1], "#") {
					bloque = m[1][1:]
					break
				}
			}
		}
		if n.es("p") {
			texto := textoNodo(n)
			resto := patron.ReplaceAllString(texto, "")
			if strings.Contains(resto, "{{") || strings.Contains(resto, "}}") {
				return fallo("docx.marcador_invalido", "")
			}
			for _, m := range patron.FindAllStringSubmatch(texto, -1) {
				if parte != "word/document.xml" {
					return fallo("docx.marcador_invalido", "")
				}
				marcas[bloque+":"+m[1]] = true
			}
		}
		for _, h := range n.hijos {
			if err := visitar(h, bloque); err != nil {
				return err
			}
		}
		return nil
	}
	return visitar(raiz, "")
}

func contenidoActivo(valor string) bool {
	valor = strings.ToLower(valor)
	if strings.HasSuffix(valor, "/package") {
		return true
	}
	for _, prohibido := range []string{"macroenabled", "vbaproject", "activex", "oleobject", "/control"} {
		if strings.Contains(valor, prohibido) {
			return true
		}
	}
	return false
}

func enlaceHTTPS(destino string) bool {
	if !strings.HasPrefix(destino, "https://") || strings.ContainsAny(destino, "\\\r\n\t") {
		return false
	}
	u, err := url.Parse(destino)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Opaque == ""
}

// Las instrucciones pueden estar divididas entre varios runs. Se valida el
// campo completo; no se admiten anidamientos, opciones ni instrucciones ocultas
// tras el separador que inicia el resultado visible.
func validarCamposWord(raiz *nodo) error {
	abierto, resultado := false, false
	var instruccion strings.Builder
	permitida := func() bool {
		switch strings.TrimSpace(instruccion.String()) {
		case "PAGE", "NUMPAGES", "SECTIONPAGES":
			return true
		}
		return false
	}
	var visitar func(*nodo) error
	visitar = func(n *nodo) error {
		if n.es("fldChar") {
			atributos := map[string]bool{}
			for _, a := range n.atributos {
				clave := n.espacios[a.Name.Space] + ":" + a.Name.Local
				if atributos[clave] {
					return fallo("docx.estructura_invalida", "")
				}
				atributos[clave] = true
			}
			if len(n.hijos) != 0 {
				return fallo("docx.estructura_invalida", "")
			}
			switch n.atributo("fldCharType") {
			case "begin":
				if abierto {
					return fallo("docx.estructura_invalida", "")
				}
				abierto, resultado = true, false
				instruccion.Reset()
			case "separate":
				if !abierto || resultado || !permitida() {
					return fallo("docx.estructura_invalida", "")
				}
				resultado = true
			case "end":
				if !abierto || !permitida() {
					return fallo("docx.estructura_invalida", "")
				}
				abierto = false
			default:
				return fallo("docx.estructura_invalida", "")
			}
		}
		if n.es("instrText") {
			if !abierto || resultado || len(n.hijos) != 0 {
				return fallo("docx.estructura_invalida", "")
			}
			instruccion.WriteString(n.texto)
		}
		// Las revisiones pueden reaparecer al aceptar cambios en Word.
		if n.es("delInstrText") || n.es("fldData") {
			return fallo("docx.estructura_invalida", "")
		}
		for _, h := range n.hijos {
			if err := visitar(h); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visitar(raiz); err != nil {
		return err
	}
	if abierto {
		return fallo("docx.estructura_invalida", "")
	}
	return nil
}
