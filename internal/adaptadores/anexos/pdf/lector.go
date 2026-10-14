// Package pdf interpreta en memoria PDF de anexos mediante el motor evaluado en F0.17a.
package pdf

import (
	"bytes"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	dslipak "github.com/dslipak/pdf"
)

var (
	ErrCorrupto = errors.New("anexo.pdf_corrupto")
	ErrTamano   = errors.New("anexo.limite_bytes")
	ErrPaginas  = errors.New("anexo.limite_paginas")
	ErrObjetos  = errors.New("anexo.limite_objetos")
	ErrRecursos = errors.New("anexo.limite_recursos")
	ErrLimites  = errors.New("anexo.limites_invalidos")
	ErrMotor    = errors.New("anexo.motor_desconocido")
)

const MaxBytes = 10_000_000

type Limites struct {
	Paginas  int
	Objetos  int
	Lecturas int
	Glifos   int
}

type Glifo struct {
	X, Y, Ancho float64
	Texto       string
}

type Linea struct {
	Pagina int
	Y      float64
	Glifos []Glifo
}

func (l Linea) Texto() string { return textoEntre(l, 0, math.MaxFloat64) }

type Documento struct {
	Paginas  int
	Lineas   []Linea
	SinTexto bool
}

// Leer trabaja exclusivamente en memoria. Los límites son parámetros del ensayo.
// Los parsers no admiten cancelación durante Content: F1.03 necesita un trabajador
// aislado con límites de CPU/memoria, además de estas cotas de entrada y salida.
func Leer(datos []byte, motor string, limites Limites) (doc Documento, err error) {
	var entrada *lectorAcotado
	defer func() {
		if recover() != nil {
			doc = Documento{}
			err = ErrCorrupto
		}
		if entrada != nil && entrada.agotado {
			doc = Documento{}
			err = ErrRecursos
		}
	}()
	if limites.Paginas <= 0 || limites.Objetos <= 0 || limites.Lecturas <= 0 || limites.Glifos <= 0 {
		return doc, ErrLimites
	}
	if len(datos) > MaxBytes {
		return doc, ErrTamano
	}
	if motor != "dslipak" {
		return doc, ErrMotor
	}
	if err := prevalidar(datos, limites); err != nil {
		return doc, err
	}
	entrada = &lectorAcotado{datos: bytes.NewReader(datos), restantes: limites.Lecturas}
	var paginas int
	var pagina func(int) ([]Glifo, error)
	switch motor {
	case "dslipak":
		r, e := dslipak.NewReader(entrada, int64(len(datos)))
		if e != nil {
			return doc, clasificar(e)
		}
		if err := validarEstructura(r, limites); err != nil {
			return doc, err
		}
		paginas = r.NumPage()
		if r.Trailer().Key("Size").Int64() > int64(limites.Objetos) {
			return doc, ErrObjetos
		}
		pagina = func(n int) ([]Glifo, error) {
			p := r.Page(n)
			if p.V.IsNull() {
				return nil, ErrCorrupto
			}
			if p.V.Key("Contents").IsNull() {
				return nil, nil
			}
			var glifos []Glifo
			for _, t := range p.Content().Text {
				glifos = append(glifos, Glifo{t.X, t.Y, t.W, t.S})
			}
			return glifos, nil
		}
	}
	if paginas < 1 {
		return doc, ErrCorrupto
	}
	if paginas > limites.Paginas {
		return doc, ErrPaginas
	}
	doc.Paginas = paginas
	total := 0
	for n := 1; n <= paginas; n++ {
		glifos, e := pagina(n)
		if e != nil {
			return Documento{}, e
		}
		if entrada.agotado {
			return Documento{}, ErrRecursos
		}
		total += len(glifos)
		if total > limites.Glifos {
			return Documento{}, ErrRecursos
		}
		doc.Lineas = append(doc.Lineas, agrupar(n, glifos)...)
	}
	doc.SinTexto = true
	for _, l := range doc.Lineas {
		if strings.TrimSpace(l.Texto()) != "" {
			doc.SinTexto = false
			break
		}
	}
	return doc, nil
}

// Prevalidación conservadora de diccionarios visibles. No es un validador PDF
// completo ni controla la expansión de streams comprimidos de las bibliotecas.
func prevalidar(datos []byte, limites Limites) error {
	if len(datos) < 100 || !regexp.MustCompile(`^%PDF-1\.[0-7][\r\n]`).Match(datos) || !bytes.HasSuffix(bytes.TrimSpace(datos), []byte("%%EOF")) {
		return ErrCorrupto
	}
	for _, c := range regexp.MustCompile(`/\s*(Size|Count)\s+([0-9]+)`).FindAllSubmatch(datos, -1) {
		n, e := strconv.ParseUint(string(c[2]), 10, 64)
		if e != nil {
			return ErrRecursos
		}
		if string(c[1]) == "Size" && n > uint64(limites.Objetos) {
			return ErrObjetos
		}
		if string(c[1]) == "Count" && n > uint64(limites.Paginas) {
			return ErrPaginas
		}
	}
	if len(regexp.MustCompile(`(?m)^\s*[0-9]+\s+[0-9]+\s+obj\b`).FindAllIndex(datos, -1)) > limites.Objetos {
		return ErrObjetos
	}
	return nil
}

type lectorAcotado struct {
	datos     *bytes.Reader
	restantes int
	agotado   bool
}

func (l *lectorAcotado) ReadAt(p []byte, off int64) (int, error) {
	if l.restantes <= 0 {
		l.agotado = true
		panic(ErrRecursos)
	}
	l.restantes--
	n, err := l.datos.ReadAt(p, off)
	if n == 0 && len(p) > 0 {
		panic(ErrCorrupto)
	}
	return n, err
}

func clasificar(e error) error {
	if errors.Is(e, ErrRecursos) {
		return ErrRecursos
	}
	return ErrCorrupto
}

func agrupar(pagina int, glifos []Glifo) []Linea {
	// Se agrupa por baseline antes de ordenar X, evitando un comparador no transitivo.
	sort.SliceStable(glifos, func(i, j int) bool { return glifos[i].Y > glifos[j].Y })
	var lineas []Linea
	for _, g := range glifos {
		if strings.TrimSpace(g.Texto) == "" {
			continue
		}
		if len(lineas) == 0 || math.Abs(lineas[len(lineas)-1].Y-g.Y) > 1.5 {
			lineas = append(lineas, Linea{Pagina: pagina, Y: g.Y})
		}
		l := &lineas[len(lineas)-1]
		l.Glifos = append(l.Glifos, g)
	}
	for i := range lineas {
		sort.SliceStable(lineas[i].Glifos, func(a, b int) bool { return lineas[i].Glifos[a].X < lineas[i].Glifos[b].X })
	}
	return lineas
}

func textoEntre(l Linea, inicio, fin float64) string {
	var b strings.Builder
	ultimo := inicio
	for _, g := range l.Glifos {
		if g.X < inicio || g.X >= fin {
			continue
		}
		if b.Len() > 0 && g.X-ultimo > 1.5 {
			b.WriteByte(' ')
		}
		b.WriteString(g.Texto)
		ultimo = g.X + g.Ancho
	}
	return strings.TrimSpace(b.String())
}

var _ io.ReaderAt = (*lectorAcotado)(nil)
