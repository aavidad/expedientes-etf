package docx

import (
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
)

const espacioWord = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// Los intervalos originales conservan prefijos, declaraciones de espacios,
// atributos de compatibilidad y partes que el motor no necesita interpretar.
type nodo struct {
	nombre                        xml.Name
	uri                           string
	atributos                     []xml.Attr
	espacios                      map[string]string
	inicio, apertura, cierre, fin int
	hijos                         []*nodo
	texto                         string
	reemplazo                     []byte
}

func leerXML(original []byte, espacios map[string]string) (*nodo, error) {
	raiz := &nodo{fin: len(original), espacios: espacios}
	pila := []*nodo{raiz}
	copiados := 0
	d := xml.NewDecoder(bytes.NewReader(original))
	for {
		inicio := int(d.InputOffset())
		token, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fallo("docx.xml_invalido", "")
		}
		padre := pila[len(pila)-1]
		switch v := token.(type) {
		case xml.StartElement:
			if len(pila) > 128 {
				return nil, fallo("docx.xml_demasiado_profundo", "")
			}
			ns, err := heredarEspacios(padre.espacios, v.Attr, &copiados)
			if err != nil {
				return nil, err
			}
			n := &nodo{nombre: v.Name, uri: ns[v.Name.Space], atributos: v.Attr, espacios: ns, inicio: inicio, apertura: int(d.InputOffset())}
			padre.hijos = append(padre.hijos, n)
			pila = append(pila, n)
		case xml.EndElement:
			if len(pila) == 1 || padre.nombre != v.Name {
				return nil, fallo("docx.xml_invalido", "")
			}
			padre.cierre, padre.fin = inicio, int(d.InputOffset())
			pila = pila[:len(pila)-1]
		case xml.CharData:
			if len(pila) == 1 && len(bytes.TrimSpace(v)) != 0 {
				return nil, fallo("docx.xml_invalido", "")
			}
			padre.texto += string(v)
		case xml.Directive:
			return nil, fallo("docx.xml_directiva_no_admitida", "")
		}
	}
	if len(pila) != 1 || len(raiz.hijos) != 1 {
		return nil, fallo("docx.xml_invalido", "")
	}
	return raiz, nil
}

// Límites de las declaraciones de espacios de nombres. Un documento de Word
// declara unas cuarenta en la raíz; los nodos que no declaran ninguna comparten
// el mapa del padre, así que sólo se copia cuando un nodo añade declaraciones.
const (
	maxEspaciosPorNodo  = 128
	maxEspaciosCopiados = 1 << 17
)

func heredarEspacios(padre map[string]string, atributos []xml.Attr, copiados *int) (map[string]string, error) {
	declarados := 0
	for _, a := range atributos {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			declarados++
		}
	}
	if declarados == 0 {
		return padre, nil
	}
	if declarados > maxEspaciosPorNodo || len(padre)+declarados > maxEspaciosCopiados-*copiados {
		return nil, fallo("docx.limite_espacios_xml", "")
	}
	*copiados += len(padre) + declarados
	ns := make(map[string]string, len(padre)+declarados)
	for k, valor := range padre {
		ns[k] = valor
	}
	for _, a := range atributos {
		if a.Name.Space == "xmlns" {
			ns[a.Name.Local] = a.Value
		}
		if a.Name.Space == "" && a.Name.Local == "xmlns" {
			ns[""] = a.Value
		}
	}
	return ns, nil
}

func (n *nodo) es(nombre string) bool { return n.uri == espacioWord && n.nombre.Local == nombre }

func (n *nodo) atributo(nombre string) string {
	for _, a := range n.atributos {
		if n.espacios[a.Name.Space] == espacioWord && a.Name.Local == nombre {
			return a.Value
		}
	}
	return ""
}

func (n *nodo) buscar(nombre string) []*nodo {
	var encontrados []*nodo
	if n.es(nombre) {
		encontrados = append(encontrados, n)
	}
	for _, h := range n.hijos {
		encontrados = append(encontrados, h.buscar(nombre)...)
	}
	return encontrados
}

func (n *nodo) hijo(nombre string) *nodo {
	for _, h := range n.hijos {
		if h.es(nombre) {
			return h
		}
	}
	return nil
}

func (n *nodo) escribir(b *bytes.Buffer, original []byte) {
	if n.reemplazo != nil {
		b.Write(n.reemplazo)
		return
	}
	pos := n.inicio
	for _, h := range n.hijos {
		b.Write(original[pos:h.inicio])
		h.escribir(b, original)
		pos = h.fin
	}
	b.Write(original[pos:n.fin])
}

func (n *nodo) bytes(original []byte) []byte {
	var b bytes.Buffer
	n.escribir(&b, original)
	return b.Bytes()
}

// Cuenta también las copias y la expansión máxima del escape XML antes de
// asignar memoria. Es deliberadamente conservador, no una cuota de negocio.
type presupuestoXML struct{ usado int }

func (p *presupuestoXML) reservar(cantidad int) error {
	if cantidad > maxDescomprimido-p.usado {
		return fallo("docx.limite_resultado", "")
	}
	p.usado += cantidad
	return nil
}

func (n *nodo) ponerTexto(original []byte, valor string, presupuesto *presupuestoXML) error {
	if err := presupuesto.reservar(len(valor)*6 + n.apertura - n.inicio + len(n.nombre.Space) + len(n.nombre.Local) + 40); err != nil {
		return err
	}
	apertura := bytes.TrimSuffix(original[n.inicio:n.apertura], []byte("/>"))
	if len(apertura) == n.apertura-n.inicio {
		apertura = apertura[:len(apertura)-1]
	}
	// Sustituye xml:space conservando el resto de atributos del nodo de texto.
	conEspacio := false
	for _, a := range n.atributos {
		if a.Name.Space == "xml" && a.Name.Local == "space" {
			conEspacio = true
			apertura = regexp.MustCompile(`xml:space\s*=\s*("[^"]*"|'[^']*')`).ReplaceAll(apertura, []byte(`xml:space="preserve"`))
		}
	}
	var b bytes.Buffer
	b.Write(apertura)
	if !conEspacio {
		b.WriteString(` xml:space="preserve"`) //etf:tecnico
	}
	b.WriteByte('>')
	_ = xml.EscapeText(&b, []byte(valor)) // bytes.Buffer nunca devuelve error.
	b.WriteString("</")                   //etf:tecnico
	if n.nombre.Space != "" {
		b.WriteString(n.nombre.Space)
		b.WriteByte(':')
	}
	b.WriteString(n.nombre.Local)
	b.WriteByte('>')
	n.reemplazo = b.Bytes()
	return nil
}
