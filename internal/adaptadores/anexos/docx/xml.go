package docx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

const nsWord = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const nsRelacion = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
const nsCasilla = "http://schemas.microsoft.com/office/word/2010/wordml"
const nsActiveX = "http://schemas.microsoft.com/office/2006/activeX"

type nodo struct {
	nombre    xml.Name
	atributos []xml.Attr
	hijos     []*nodo
	texto     []string
}

// leerXML no resuelve entidades ni relaciones. Limita profundidad y nodos antes
// de construir el árbol; rechaza DTD y declaraciones de entidades.
func leerXML(b []byte) (*nodo, error) {
	if err := RevisarMarcado(b); err != nil {
		return nil, err
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	var pila []*nodo
	var raiz *nodo
	cuenta := 0
	tokens := 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("anexo.docx_xml_invalido")
		}
		tokens++
		if tokens > 500000 {
			return nil, errors.New("anexo.docx_limite_xml")
		}
		switch v := t.(type) {
		case xml.Directive:
			return nil, errors.New("anexo.docx_xml_inseguro")
		case xml.StartElement:
			cuenta++
			if len(pila) >= 64 || cuenta > 150000 {
				return nil, errors.New("anexo.docx_limite_xml")
			}
			n := &nodo{nombre: v.Name, atributos: v.Attr}
			if len(pila) > 0 {
				padre := pila[len(pila)-1]
				padre.hijos = append(padre.hijos, n)
			} else {
				if raiz != nil {
					return nil, errors.New("anexo.docx_xml_invalido")
				}
				raiz = n
			}
			pila = append(pila, n)
		case xml.EndElement:
			pila = pila[:len(pila)-1]
		case xml.CharData:
			if len(pila) == 0 && strings.TrimSpace(string(v)) != "" {
				return nil, errors.New("anexo.docx_xml_invalido")
			}
			if len(pila) > 0 && pila[len(pila)-1].es(nsWord, "t") {
				n := pila[len(pila)-1]
				n.texto = append(n.texto, string(v))
			}
		}
	}
	if raiz == nil || len(pila) != 0 {
		return nil, errors.New("anexo.docx_xml_invalido")
	}
	return raiz, nil
}

func (n *nodo) es(ns, local string) bool {
	return n != nil && n.nombre.Space == ns && n.nombre.Local == local
}
func (n *nodo) atributo(ns, local string) string {
	for _, a := range n.atributos {
		if a.Name.Space == ns && a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}
func (n *nodo) directos(local string) []*nodo {
	var res []*nodo
	for _, h := range n.hijos {
		if h.es(nsWord, local) {
			res = append(res, h)
		}
	}
	return res
}
func (n *nodo) buscar(ns, local string) []*nodo {
	var res []*nodo
	if n.es(ns, local) {
		res = append(res, n)
	}
	for _, h := range n.hijos {
		res = append(res, h.buscar(ns, local)...)
	}
	return res
}
func (n *nodo) contenido(estatico bool) string {
	var b strings.Builder
	var recorrer func(*nodo)
	recorrer = func(x *nodo) {
		if x.es(nsWord, "sdtPr") || x.es(nsWord, "del") || (estatico && x.es(nsWord, "sdt")) {
			return
		}
		if x.es(nsWord, "t") {
			for _, fragmento := range x.texto {
				b.WriteString(fragmento)
			}
		}
		if x.es(nsWord, "tab") || x.es(nsWord, "br") {
			b.WriteByte(' ')
		}
		for _, h := range x.hijos {
			recorrer(h)
		}
		if x.es(nsWord, "p") {
			b.WriteByte('\n')
		}
	}
	recorrer(n)
	return strings.TrimSpace(b.String())
}
func normalizar(s string) string { return strings.Join(strings.Fields(s), " ") }
