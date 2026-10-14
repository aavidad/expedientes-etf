package docx

import (
	"path"
	"strconv"
	"strings"
)

func extraer(tablas []*nodo, p perfil, partes map[string][]byte) (Lectura, error) {
	casillas, err := leerCasillas(partes)
	if err != nil {
		return Lectura{}, err
	}
	l := Lectura{PlantillaReconocida: true, Tipo: p.Tipo, Version: p.Version, Avisos: []string{"anexo.confirmar_docx"}}
	var reglas []regla
	for i := 0; i < len(p.Reglas); {
		r := p.Reglas[i]
		if !r.Repetir {
			reglas = append(reglas, r)
			i++
			continue
		}
		fin := i
		for fin < len(p.Reglas) && p.Reglas[fin].Repetir && p.Reglas[fin].Tabla == r.Tabla {
			fin++
		}
		for fila := r.Fila; fila < len(tablas[r.Tabla].directos("tr")); fila++ {
			for _, reglaFila := range p.Reglas[i:fin] {
				reglaFila.Fila = fila
				reglaFila.Repetir = false
				reglas = append(reglas, reglaFila)
			}
		}
		i = fin
	}
	for _, r := range reglas {
		filas := tablas[r.Tabla].directos("tr")
		fin := r.Fila + 1
		if r.Repetir {
			fin = len(filas)
		}
		for fi := r.Fila; fi < fin; fi++ {
			celda := filas[fi].directos("tc")[r.Celda]
			c := Campo{Codigo: r.Codigo}
			var motivo string
			switch r.Modo {
			case "casillas":
				c.Valor, motivo = seleccion(celda, r.Opciones, casillas)
			case "parrafo":
				parrafos := celda.directos("p")
				if r.Control < len(parrafos) {
					c.Valor = strings.TrimSpace(strings.TrimPrefix(normalizar(parrafos[r.Control].contenido(false)), r.Etiqueta))
				} else {
					motivo = "anexo.campo_ausente"
				}
			case "papel":
				c.Valor = celda.contenido(false)
			default:
				controles := celda.buscar(nsWord, "sdt")
				if r.Control >= 0 && r.Control < len(controles) {
					contenido := controles[r.Control].directos("sdtContent")
					if len(contenido) == 1 {
						c.Valor = contenido[0].contenido(false)
					} else {
						motivo = "anexo.campo_ausente"
					}
				} else if r.Control == 0 && len(controles) == 0 {
					c.Valor = strings.TrimSpace(strings.TrimPrefix(normalizar(celda.contenido(false)), r.Etiqueta))
				} else {
					motivo = "anexo.campo_ausente"
				}
			}
			v := normalizar(c.Valor)
			if motivo == "" {
				switch {
				case v == "":
					motivo = "anexo.campo_ausente"
				case v == normalizar(r.Relleno) && r.Relleno != "":
					motivo = "anexo.texto_relleno"
				case len(r.Opciones) > 0 && !enLista(v, r.Opciones):
					motivo = "anexo.valor_fuera_lista"
				}
			}
			if motivo != "" {
				c.Dudoso = true
				c.Motivos = []string{motivo}
			}
			if r.Codigo == "sociodemografia.adulto.estudios" && !c.Dudoso {
				if destino, ok := p.Estudios[v]; ok {
					c.Valor = destino
				} else {
					c.Dudoso = true
					c.Motivos = []string{"anexo.equivalencia_ausente"}
				}
			}
			l.Campos = append(l.Campos, c)
		}
	}
	return l, nil
}

func enLista(valor string, opciones []string) bool {
	for _, v := range opciones {
		if valor == normalizar(v) {
			return true
		}
	}
	return false
}

// Las casillas ActiveX del Anexo 3 se leen como propiedades XML, sin cargar ni
// ejecutar el control. Las referencias se resuelven sólo dentro del ZIP.
func leerCasillas(partes map[string][]byte) (map[string]string, error) {
	res := make(map[string]string)
	b := partes["word/_rels/document.xml.rels"]
	if b == nil {
		return res, nil
	}
	n, err := leerXML(b)
	if err != nil {
		return nil, err
	}
	const ns = "http://schemas.openxmlformats.org/package/2006/relationships"
	for _, r := range n.buscar(ns, "Relationship") {
		if r.atributo("", "Type") != "http://schemas.openxmlformats.org/officeDocument/2006/relationships/control" {
			continue
		}
		if r.atributo("", "TargetMode") == "External" {
			continue
		}
		destino := path.Clean(path.Join("word", r.atributo("", "Target")))
		if !strings.HasPrefix(destino, "word/activeX/") || !strings.HasSuffix(destino, ".xml") {
			continue
		}
		b = partes[destino]
		if b == nil {
			continue
		}
		x, e := leerXML(b)
		if e != nil {
			return nil, e
		}
		for _, prop := range x.buscar(nsActiveX, "ocxPr") {
			if prop.atributo(nsActiveX, "name") == "Value" {
				res[r.atributo("", "Id")] = prop.atributo(nsActiveX, "value")
			}
		}
	}
	return res, nil
}

func seleccion(celda *nodo, opciones []string, activeX map[string]string) (string, string) {
	var estados []string
	for _, n := range celda.buscar(nsCasilla, "checked") {
		v := n.atributo(nsCasilla, "val")
		if v == "" {
			v = n.atributo(nsWord, "val")
		}
		estados = append(estados, v)
	}
	if len(estados) == 0 {
		for _, n := range celda.buscar(nsWord, "control") {
			estados = append(estados, activeX[n.atributo(nsRelacion, "id")])
		}
	}
	if len(estados) != len(opciones) {
		return "", "anexo.casilla_no_legible"
	}
	valor := ""
	for i, v := range estados {
		b, e := strconv.ParseBool(v)
		if e != nil {
			return "", "anexo.casilla_no_legible"
		}
		if b {
			if valor != "" {
				return "", "anexo.valor_ambiguo"
			}
			valor = opciones[i]
		}
	}
	if valor == "" {
		return "", "anexo.campo_ausente"
	}
	return valor, ""
}
