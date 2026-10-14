// Package ubicacion propone un municipio para que la coordinación lo confirme.
package ubicacion

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Municipio contiene sólo los datos del catálogo necesarios para la búsqueda.
// Nombre se resuelve mediante ClaveTexto fuera del dominio; Variantes incluye
// las grafías admitidas por el catálogo vigente. No se deducen centro ni equipo.
type Municipio struct {
	Codigo           string
	ClaveTexto       string
	Nombre           string
	Variantes        []string
	AmbitoDiputacion bool
}

type Campo uint8

const (
	CampoDireccion Campo = iota + 1
	CampoValoracionDiagnostico
)

type Estado uint8

const (
	SinCandidatos Estado = iota
	CandidatoUnico
	Ambiguo
)

// Candidato conserva la grafía original de la coincidencia. Inicio y Fin son
// posiciones en bytes dentro del campo de origen: texto[Inicio:Fin] == Fragmento.
// FueraAmbitoDiputacion procede del catálogo, nunca del nombre del municipio.
type Candidato struct {
	MunicipioID           string
	CentroID              *string
	EquipoID              *string
	Codigo                string
	ClaveTexto            string
	FueraAmbitoDiputacion bool
	Campo                 Campo
	Fragmento             string
	Inicio                int
	Fin                   int
}

// Propuesta reúne municipios distintos, ordenados por su aparición en el texto.
// Incluso un candidato único necesita confirmación humana antes de asignarse.
type Propuesta struct {
	Estado     Estado
	Candidatos []Candidato
}

// ProponerMunicipio busca primero en Dirección. Sólo consulta Valoración y
// diagnóstico si Dirección no tiene candidatos. Una Dirección ambigua no se
// resuelve mediante Valoración.
// Las coincidencias largas ocultan las cortas contenidas en ese mismo fragmento;
// una mención independiente de otro municipio conserva la ambigüedad.
func ProponerMunicipio(direccion, valoracion string, municipios []Municipio) Propuesta {
	candidatos := buscar(direccion, CampoDireccion, municipios)
	if len(candidatos) == 0 {
		candidatos = buscar(valoracion, CampoValoracionDiagnostico, municipios)
	}
	estado := SinCandidatos
	if len(candidatos) == 1 {
		estado = CandidatoUnico
	} else if len(candidatos) > 1 {
		estado = Ambiguo
	}
	return Propuesta{Estado: estado, Candidatos: candidatos}
}

type palabra struct {
	texto       string
	inicio, fin int
}

type coincidencia struct {
	municipio   Municipio
	inicio, fin int // Índices de palabras; fin es exclusivo.
	longitud    int // Longitud normalizada, independiente de tildes y signos.
}

func buscar(texto string, campo Campo, municipios []Municipio) []Candidato {
	palabras := normalizar(texto)
	var coincidencias []coincidencia
	for _, municipio := range municipios {
		// No modificar ni retener las variantes del catálogo recibido.
		grafias := append([]string{municipio.Nombre}, municipio.Variantes...)
		for _, grafia := range grafias {
			nombre := normalizar(grafia)
			if len(nombre) == 0 {
				continue
			}
			longitud := len(nombre) - 1
			for _, p := range nombre {
				longitud += utf8.RuneCountInString(p.texto)
			}
			for inicio := 0; inicio+len(nombre) <= len(palabras); inicio++ {
				coincide := true
				for j, p := range nombre {
					if palabras[inicio+j].texto != p.texto {
						coincide = false
						break
					}
				}
				if coincide {
					coincidencias = append(coincidencias, coincidencia{municipio, inicio, inicio + len(nombre), longitud})
				}
			}
		}
	}
	sort.Slice(coincidencias, func(i, j int) bool {
		a, b := coincidencias[i], coincidencias[j]
		if a.longitud != b.longitud {
			return a.longitud > b.longitud
		}
		if a.inicio != b.inicio {
			return a.inicio < b.inicio
		}
		return a.municipio.Codigo < b.municipio.Codigo
	})
	var candidatos []Candidato
	vistos := make(map[string]bool)
	// Cada posición recuerda hasta dónde la cubre una coincidencia más larga.
	// Procesar longitudes por grupos conserva los alias compartidos como ambiguos
	// y evita comparar cada repetición con todas las anteriores.
	cubiertaHasta := make([]int, len(palabras))
	for primero := 0; primero < len(coincidencias); {
		ultimo := primero + 1
		for ultimo < len(coincidencias) && coincidencias[ultimo].longitud == coincidencias[primero].longitud {
			ultimo++
		}
		for _, c := range coincidencias[primero:ultimo] {
			if cubiertaHasta[c.inicio] >= c.fin || vistos[c.municipio.Codigo] {
				continue
			}
			vistos[c.municipio.Codigo] = true
			inicio, fin := palabras[c.inicio].inicio, palabras[c.fin-1].fin
			candidatos = append(candidatos, Candidato{
				Codigo: c.municipio.Codigo, ClaveTexto: c.municipio.ClaveTexto,
				FueraAmbitoDiputacion: !c.municipio.AmbitoDiputacion,
				Campo:                 campo, Fragmento: texto[inicio:fin], Inicio: inicio, Fin: fin,
			})
		}
		// Las repeticiones también ocultan nombres cortos en su propio fragmento.
		for _, c := range coincidencias[primero:ultimo] {
			for pos := c.inicio; pos < c.fin; pos++ {
				cubiertaHasta[pos] = max(cubiertaHasta[pos], c.fin)
			}
		}
		primero = ultimo
	}
	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].Inicio != candidatos[j].Inicio {
			return candidatos[i].Inicio < candidatos[j].Inicio
		}
		return candidatos[i].Codigo < candidatos[j].Codigo
	})
	return candidatos
}

// normalizar separa palabras por signos y espacios, conserva sus posiciones
// originales y elimina diacríticos latinos, incluso los descompuestos en Unicode.
func normalizar(texto string) []palabra {
	var palabras []palabra
	var actual strings.Builder
	inicio, fin := 0, 0
	terminar := func() {
		if actual.Len() > 0 {
			palabras = append(palabras, palabra{actual.String(), inicio, fin})
			actual.Reset()
		}
	}
	for pos, r := range texto {
		if unicode.IsMark(r) {
			if actual.Len() > 0 {
				fin = pos + utf8.RuneLen(r)
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			terminar()
			continue
		}
		if actual.Len() == 0 {
			inicio = pos
		}
		fin = pos + utf8.RuneLen(r)
		r = unicode.ToLower(r)
		switch r {
		case 'á', 'à', 'ä', 'â', 'ã', 'å':
			r = 'a'
		case 'é', 'è', 'ë', 'ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô', 'õ':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û':
			r = 'u'
		case 'ñ':
			r = 'n'
		case 'ç':
			r = 'c'
		}
		actual.WriteRune(r)
	}
	terminar()
	return palabras
}
