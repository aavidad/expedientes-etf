package pdf

import (
	"regexp"
	"strings"
)

// Perfil describe una versión de plantilla como datos del ensayo. Las etiquetas
// se leen del perfil; nunca se devuelven como textos de interfaz.
type Perfil struct {
	Tipo           string
	CamposManuales map[string][]string
	Encabezado     string
	Secciones      []Seccion
}

type Seccion struct {
	Codigo      string
	Inicio, Fin string
	Escalares   []Escalar
	Tablas      []Tabla
}

type Escalar struct {
	Codigo, Ancla, Patron string
	Desde, Hasta          float64
	Continuaciones        int
}

type Tabla struct {
	Codigo, Inicio, Fin string
	FinFilas            string
	Antes               float64
	Clave               int
	Patron              string
	Columnas            []Columna
}

type Columna struct {
	Codigo       string
	Desde, Hasta float64
	Separador    string
}

type Campo struct {
	Codigo  string
	Valor   string
	Pagina  int
	Dudoso  bool
	Motivos []string
}

type Lectura struct {
	PlantillaReconocida bool
	Campos              []Campo
	Avisos              []string
}

// Extraer propone valores y conserva la página de origen. Todo PDF requiere
// confirmación humana, también cuando un valor coincide exactamente con la tabla.
func Extraer(doc Documento, perfil Perfil) Lectura {
	var lectura Lectura
	if doc.SinTexto {
		lectura.Avisos = []string{"anexo.pdf_sin_texto"}
		return lectura
	}
	for _, l := range doc.Lineas {
		if coincide(perfil.Encabezado, l.Texto()) {
			lectura.PlantillaReconocida = true
			break
		}
	}
	if !lectura.PlantillaReconocida {
		lectura.Avisos = []string{"anexo.plantilla_desconocida"}
		return lectura
	}
	for _, s := range perfil.Secciones {
		lineas := region(doc.Lineas, s.Inicio, s.Fin)
		for _, regla := range s.Escalares {
			c := Campo{Codigo: s.Codigo + "." + regla.Codigo, Dudoso: true, Motivos: []string{"anexo.confirmar_pdf"}}
			for i, l := range lineas {
				if !coincide(regla.Ancla, l.Texto()) {
					continue
				}
				v := textoEntre(l, regla.Desde, regla.Hasta)
				if regla.Patron != "" {
					m, e := regexp.Compile(regla.Patron)
					if e != nil {
						continue
					}
					partes := m.FindStringSubmatch(v)
					if len(partes) < 2 {
						continue
					}
					v = partes[1]
				}
				for j := 1; j <= regla.Continuaciones && i+j < len(lineas); j++ {
					siguiente := lineas[i+j]
					otra := false
					for _, r := range s.Escalares {
						if coincide(r.Ancla, siguiente.Texto()) {
							otra = true
						}
					}
					if otra {
						break
					}
					v = strings.TrimSpace(v + " " + textoEntre(siguiente, regla.Desde, regla.Hasta))
				}
				if c.Pagina != 0 {
					c.Valor = ""
					c.Motivos = append(c.Motivos, "anexo.campo_ambiguo")
					break
				}
				c.Valor = strings.TrimSpace(v)
				c.Pagina = l.Pagina
			}
			if c.Valor == "" {
				c.Motivos = append(c.Motivos, "anexo.campo_no_leido")
			}
			lectura.Campos = append(lectura.Campos, c)
		}
		for _, tabla := range s.Tablas {
			filas := region(lineas, tabla.Inicio, tabla.Fin)
			for i, fila := range filas {
				if coincide(tabla.FinFilas, fila.Texto()) {
					filas = filas[:i]
					break
				}
			}
			if tabla.Clave < 0 || tabla.Clave >= len(tabla.Columnas) {
				lectura.Avisos = append(lectura.Avisos, "anexo.perfil_invalido")
				continue
			}
			clave := tabla.Columnas[tabla.Clave]
			var inicios []int
			for i, l := range filas {
				if coincide(tabla.Patron, textoEntre(l, clave.Desde, clave.Hasta)) {
					inicio := i
					for inicio > 0 && filas[inicio-1].Pagina == l.Pagina && filas[inicio-1].Y-l.Y <= tabla.Antes {
						inicio--
					}
					inicios = append(inicios, inicio)
				}
			}
			if len(inicios) == 0 {
				lectura.Avisos = append(lectura.Avisos, "anexo.tabla_no_leida")
				continue
			}
			for f, inicio := range inicios {
				fin := len(filas)
				if f+1 < len(inicios) {
					fin = inicios[f+1]
				}
				for _, columna := range tabla.Columnas {
					var valores []string
					for _, l := range filas[inicio:fin] {
						if v := textoEntre(l, columna.Desde, columna.Hasta); v != "" {
							valores = append(valores, v)
						}
					}
					c := Campo{Codigo: s.Codigo + "." + tabla.Codigo + "." + columna.Codigo, Valor: strings.Join(valores, columna.Separador), Pagina: filas[inicio].Pagina, Dudoso: true, Motivos: []string{"anexo.confirmar_pdf"}}
					if c.Valor == "" {
						c.Motivos = append(c.Motivos, "anexo.campo_no_leido")
					}
					lectura.Campos = append(lectura.Campos, c)
				}
			}
		}
	}
	return lectura
}

func coincide(patron, texto string) bool {
	if patron == "" {
		return false
	}
	r, e := regexp.Compile(patron)
	return e == nil && r.MatchString(texto)
}

func region(lineas []Linea, inicio, fin string) []Linea {
	for i, l := range lineas {
		if !coincide(inicio, l.Texto()) {
			continue
		}
		for j := i + 1; j < len(lineas); j++ {
			if coincide(fin, lineas[j].Texto()) {
				return lineas[i+1 : j]
			}
		}
		return lineas[i+1:]
	}
	return nil
}
