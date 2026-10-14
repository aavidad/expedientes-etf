package pdf

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/odt"
)

//go:embed libreoffice.json
var libreOfficeJSON []byte

type reglaLibreOffice struct {
	Codigo, Inicio, Fin, Prefijo string
	AmbitoInicio, AmbitoFin      string
	Desde, Hasta                 float64
	Omitir                       int
	Opciones                     []string
	EliminarHasta                string
	Antes                        float64
	PrimeraOpcion, Fecha         bool
}
type perfilLibreOffice struct {
	Marcador string
	Perfil   Perfil
	Reglas   []reglaLibreOffice
	Manuales []string
}

// extraerLibreOffice reconoce la variante convertida por sus cabeceras, no por
// datos familiares. Los valores no legibles o los controles perdidos quedan vacíos.
func extraerLibreOffice(doc Documento) (Lectura, bool) {
	var p perfilLibreOffice
	if json.Unmarshal(libreOfficeJSON, &p) != nil {
		return Lectura{}, false
	}
	reconocida := false
	for _, linea := range doc.Lineas {
		if linea.Pagina == 1 && strings.HasPrefix(linea.Texto(), p.Marcador) {
			reconocida = true
		}
	}
	if !reconocida {
		return Lectura{}, false
	}
	e := Extraer(doc, p.Perfil)
	if !e.PlantillaReconocida {
		return e, true
	}
	reemplazar := func(c Campo) {
		for i, v := range e.Campos {
			if v.Codigo == c.Codigo {
				e.Campos[i] = c
				return
			}
		}
		e.Campos = append(e.Campos, c)
	}
	for _, r := range p.Reglas {
		c := Campo{Codigo: r.Codigo, Dudoso: true, Motivos: []string{"anexo.confirmar_pdf"}}
		encontrado := false
		lineas := doc.Lineas
		if r.AmbitoInicio != "" {
			lineas = region(lineas, r.AmbitoInicio, r.AmbitoFin)
		}
		for i, linea := range lineas {
			if !coincide(r.Inicio, linea.Texto()) {
				continue
			}
			if encontrado {
				c.Valor = ""
				c.Motivos = append(c.Motivos, "anexo.campo_ambiguo")
				break
			}
			encontrado = true
			c.Pagina = linea.Pagina
			var valores []string
			inicio := i
			if r.Antes > 0 && i > 0 && lineas[i-1].Pagina == linea.Pagina && lineas[i-1].Y-linea.Y <= r.Antes {
				inicio = i - 1
			}
			for j := inicio; j < len(lineas); j++ {
				l := lineas[j]
				if j > i && coincide(r.Fin, l.Texto()) {
					break
				}
				if j > i && j+1 < len(lineas) && lineas[j+1].Pagina == l.Pagina && l.Y-lineas[j+1].Y < 3 && coincide(r.Fin, lineas[j+1].Texto()) {
					break
				}
				if (j < i+r.Omitir && j >= i) || l.Y > 790 {
					continue
				}
				v := textoEntre(l, r.Desde, r.Hasta)
				if j == i && r.Prefijo != "" {
					v = strings.TrimSpace(strings.TrimPrefix(v, r.Prefijo))
				}
				if v != "" {
					valores = append(valores, v)
				}
			}
			c.Valor = strings.Join(valores, " ")
			if r.EliminarHasta != "" {
				_, c.Valor, _ = strings.Cut(c.Valor, r.EliminarHasta)
				c.Valor = strings.TrimSpace(c.Valor)
			}
			if len(r.Opciones) > 0 {
				raw := c.Valor
				c.Valor = ""
				for _, op := range r.Opciones {
					op = strings.TrimSpace(op)
					if (strings.HasSuffix(raw, op) || (r.PrimeraOpcion && strings.Contains(raw, op))) && len(op) > len(c.Valor) {
						c.Valor = op
					}
				}
			}
		}
		if r.Fecha {
			re := regexp.MustCompile(`[Sd]?\d{2}/\d{2}/(?:\d{4}|\d{2})`)
			c.Valor = re.FindString(c.Valor)
		}
		if c.Valor == "" {
			c.Motivos = append(c.Motivos, "anexo.campo_no_leido")
		}
		reemplazar(c)
	}
	estructura := ""
	for _, c := range e.Campos {
		if c.Codigo == "sociodemografia.estructura" {
			estructura = c.Valor
		}
	}
	if estructura != "" {
		for _, r := range p.Reglas {
			if r.Codigo == "sociodemografia.estructura" {
				ls := region(doc.Lineas, r.Inicio, r.Fin)
				var vs []string
				for _, l := range ls {
					if v := textoEntre(l, r.Desde, r.Hasta); v != "" {
						vs = append(vs, v)
					}
				}
				reemplazar(Campo{Codigo: "sociodemografia.otras_formas", Valor: strings.TrimSpace(strings.TrimPrefix(strings.Join(vs, " "), estructura)), Dudoso: true, Motivos: []string{"anexo.confirmar_pdf"}})
			}
		}
	}
	for _, codigo := range p.Manuales {
		reemplazar(Campo{Codigo: codigo, Dudoso: true, Motivos: []string{"anexo.confirmar_pdf", "anexo.campo_no_leido"}})
	}
	tipo := ""
	for _, l := range doc.Lineas {
		if m := regexp.MustCompile(p.Perfil.Tipo).FindStringSubmatch(l.Texto()); len(m) == 2 {
			tipo = "anexo" + m[1]
			break
		}
	}
	permitidos := odt.Codigos(tipo)
	campos := e.Campos[:0]
	for _, c := range e.Campos {
		if permitidos[c.Codigo] {
			campos = append(campos, c)
		}
	}
	e.Campos = campos
	for i := range e.Campos {
		e.Campos[i].Valor = odt.NormalizarCampo(e.Campos[i].Codigo, e.Campos[i].Valor)
	}
	return e, true
}
