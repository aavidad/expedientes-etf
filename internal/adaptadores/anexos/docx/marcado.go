package docx

import (
	"bytes"
	"errors"
)

// Límites del marcado de cada parte XML, comprobados sobre los bytes antes de
// decodificar: el decodificador de Go reserva todos los atributos de una
// etiqueta de una vez, así que la cota tiene que ir por delante.
const (
	maxAtributosEtiqueta = 128
	maxBytesEtiqueta     = 64 << 10
	maxAtributosParte    = 300000
)

// RevisarMarcado recorre las etiquetas de una parte XML contando atributos
// (signos «=» fuera de comillas) y el tamaño de cada etiqueta. Los
// comentarios, las secciones CDATA y las instrucciones de proceso se saltan.
func RevisarMarcado(b []byte) error {
	limite := errors.New("anexo.docx_limite_xml")
	total := 0
	for i := 0; i < len(b); {
		inicio := bytes.IndexByte(b[i:], '<')
		if inicio < 0 {
			return nil
		}
		i += inicio
		resto := b[i:]
		saltar := func(fin string) bool {
			j := bytes.Index(resto, []byte(fin))
			if j < 0 {
				return false
			}
			i += j + len(fin)
			return true
		}
		switch {
		case bytes.HasPrefix(resto, []byte("<!--")):
			if !saltar("-->") {
				return nil
			}
			continue
		case bytes.HasPrefix(resto, []byte("<![CDATA[")):
			if !saltar("]]>") {
				return nil
			}
			continue
		}
		atributos, comilla := 0, byte(0)
		j := 1
		for ; j < len(resto); j++ {
			c := resto[j]
			if comilla != 0 {
				if c == comilla {
					comilla = 0
				}
				continue
			}
			if c == '"' || c == '\'' {
				comilla = c
			} else if c == '=' {
				atributos++
			} else if c == '>' {
				break
			}
		}
		total += atributos
		if j > maxBytesEtiqueta || atributos > maxAtributosEtiqueta || total > maxAtributosParte {
			return limite
		}
		i += j + 1
	}
	return nil
}
