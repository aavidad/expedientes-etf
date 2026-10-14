package docx

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/actas"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

// LeerActa reutiliza las cotas ZIP/XML del lector de anexos. No abre relaciones.
func LeerActa(b []byte) (d.Extraccion, error) {
	partes, err := abrir(b)
	if err != nil {
		return d.Extraccion{}, err
	}
	defer func() {
		for _, b := range partes {
			clear(b)
		}
	}()
	raiz, err := leerXML(partes["word/document.xml"])
	if err != nil {
		return d.Extraccion{}, err
	}
	var lineas []string
	for _, p := range raiz.buscar(nsWord, "p") {
		lineas = append(lineas, p.contenido(false))
	}
	return actas.Extraer(lineas), nil
}
