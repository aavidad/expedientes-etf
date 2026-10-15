package odt

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/actas"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// LeerActa reutiliza la validación ZIP/XML de RESP4 y el perfil común de actas.
// Conserva el orden de los párrafos, también dentro de las tablas, en memoria.
func LeerActa(b []byte) (d.Extraccion, error) {
	raiz, err := abrir(b)
	if err != nil {
		return d.Extraccion{}, err
	}
	var lineas []string
	var recorrer func(*nodo)
	recorrer = func(n *nodo) {
		if n.nombre.Space == nsText && (n.nombre.Local == "p" || n.nombre.Local == "h") {
			lineas = append(lineas, normalizar(n.texto(nil)))
			return
		}
		for _, h := range n.hijos {
			recorrer(h)
		}
	}
	recorrer(raiz)
	return actas.Extraer(lineas), nil
}
