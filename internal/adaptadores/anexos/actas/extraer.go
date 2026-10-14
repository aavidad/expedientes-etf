// Package actas interpreta propuestas de datos, siempre pendientes de revisión.
package actas

import (
	_ "embed"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"regexp"
	"strings"
	"time"
)

//go:embed perfil.json
var perfilJSON []byte

type regla struct {
	Codigo, Patron string
	Siguiente      bool
	Fin            string
}
type perfil struct {
	Titulos map[string]string
	Campos  []regla
}

// Extraer conserva únicamente los campos reconocidos. Nunca devuelve el texto
// completo ni interpreta la fecha de lectura como fecha del acta.
func Extraer(lineas []string) d.Extraccion {
	enumeracion := regexp.MustCompile(`^\s*[0-9]+\.\s+`)
	limpias := make([]string, len(lineas))
	for i, l := range lineas {
		limpias[i] = enumeracion.ReplaceAllString(l, "")
	}
	lineas = limpias
	var p perfil
	if json.Unmarshal(perfilJSON, &p) != nil {
		return d.Extraccion{}
	}
	r := d.Extraccion{Version: "actas-1", Campos: []d.Campo{}, Avisos: []string{}}
	texto := strings.ToLower(strings.Join(lineas, " "))
	for tipo, patron := range p.Titulos {
		if regexp.MustCompile(patron).MatchString(texto) {
			if r.PlantillaReconocida {
				return d.Extraccion{Version: "actas-1", Campos: []d.Campo{}, Avisos: []string{}}
			}
			r.Tipo = tipo
			r.PlantillaReconocida = true
		}
	}
	if !r.PlantillaReconocida {
		return r
	}
	menor := false
	patrones := make([]*regexp.Regexp, len(p.Campos))
	finales := make([]*regexp.Regexp, len(p.Campos))
	for i, c := range p.Campos {
		patrones[i] = regexp.MustCompile(c.Patron)
		if c.Fin != "" {
			finales[i] = regexp.MustCompile(c.Fin)
		}
	}
	numero := regexp.MustCompile(`\b[0-9]{2}-[0-9]{2}-[0-9]{2}\b`)
	for i, linea := range lineas {
		if len(r.Campos) >= 100 {
			break
		}
		linea = strings.TrimSpace(linea)
		for j, regla := range p.Campos {
			m := patrones[j].FindStringSubmatch(linea)
			if len(m) != 2 {
				continue
			}
			valor := strings.TrimSpace(m[1])
			if finales[j] != nil {
				for k := i + 1; k < len(lineas) && !finales[j].MatchString(strings.TrimSpace(lineas[k])); k++ {
					if len(valor)+len(lineas[k]) > 4000 {
						break
					}
					valor = strings.TrimSpace(valor + " " + lineas[k])
				}
			}
			if valor == "" && regla.Siguiente && i+1 < len(lineas) {
				valor = strings.TrimSpace(lineas[i+1])
			}
			if regla.Codigo == "menor.nombre" {
				menor = true
			}
			if strings.HasPrefix(regla.Codigo, "menor.") && !menor {
				continue
			}
			if regla.Codigo == "menor.nombre" && len(r.Campos) >= 100 {
				continue
			}
			if valor == "" || strings.Contains(valor, "{{") || len(valor) > 4000 {
				continue
			}
			if regla.Codigo == "fecha" || regla.Codigo == "menor.nacimiento" {
				valor = Fecha(valor)
				if valor == "" {
					continue
				}
			}
			if regla.Codigo == "numero" {
				valor = numero.FindString(valor)
				if valor == "" {
					continue
				}
			}
			r.Campos = append(r.Campos, d.Campo{Codigo: regla.Codigo, Valor: valor, Dudoso: true, Motivos: []string{"carga.actas.revisar"}})
		}
	}
	return d.DatosPermitidos(r)
}

// Fecha admite fechas completas; un año del número de expediente no es una fecha.
func Fecha(s string) string {
	for _, formato := range []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006"} {
		if f, err := time.Parse(formato, strings.TrimSpace(s)); err == nil && f.Year() > 0 {
			return f.Format("2006-01-02")
		}
	}
	return ""
}
