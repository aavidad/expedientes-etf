package hoja

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"io"
	"path"
	"strconv"
	"strings"
)

const maxDescomprimido = 40_000_000

type textoXML struct {
	Texto string `xml:"t"`
	Runs  []struct {
		Texto string `xml:"t"`
	} `xml:"r"`
}

func (t textoXML) valor() string {
	var b strings.Builder
	b.WriteString(t.Texto)
	for _, r := range t.Runs {
		b.WriteString(r.Texto)
	}
	return b.String()
}

type relacionXML struct {
	ID      string `xml:"Id,attr"`
	Tipo    string `xml:"Type,attr"`
	Destino string `xml:"Target,attr"`
	Modo    string `xml:"TargetMode,attr"`
}

func leerXLSX(ctx context.Context, b []byte) ([]registro, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, d.ErrHojaFormato
	}
	if len(z.File) > 64 {
		return nil, d.ErrHojaLimites
	}
	partes := map[string][]byte{}
	total := 0
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n := f.Name
		if n != path.Clean(n) || strings.HasPrefix(n, "/") || strings.Contains(n, "\\") || strings.HasPrefix(n, "../") || !f.Mode().IsRegular() || f.Flags&1 != 0 {
			return nil, d.ErrHojaFormato
		}
		if _, repetida := partes[n]; repetida {
			return nil, d.ErrHojaFormato
		}
		permitida := n == "[Content_Types].xml" || n == "_rels/.rels" || n == "xl/workbook.xml" || n == "xl/_rels/workbook.xml.rels" || n == "xl/sharedStrings.xml" || n == "xl/styles.xml" || n == "xl/theme/theme1.xml" || n == "docProps/core.xml" || n == "docProps/app.xml" || strings.HasPrefix(n, "xl/worksheets/") && path.Ext(n) == ".xml" && !strings.Contains(strings.TrimPrefix(n, "xl/worksheets/"), "/")
		if !permitida {
			return nil, d.ErrHojaFormato
		}
		if f.UncompressedSize64 > maxDescomprimido || uint64(total)+f.UncompressedSize64 > maxDescomprimido {
			return nil, d.ErrHojaLimites
		}
		r, err := f.Open()
		if err != nil {
			return nil, d.ErrHojaFormato
		}
		v, err := io.ReadAll(io.LimitReader(r, int64(maxDescomprimido-total)+1))
		cerrar := r.Close()
		if err != nil || cerrar != nil {
			return nil, d.ErrHojaFormato
		}
		total += len(v)
		if total > maxDescomprimido {
			return nil, d.ErrHojaLimites
		}
		if err := revisarXML(ctx, v); err != nil {
			return nil, err
		}
		partes[n] = v
	}
	// Todas las relaciones se inspeccionan, incluidas las del paquete raíz.
	for nombre, v := range partes {
		if !strings.HasSuffix(nombre, ".rels") {
			continue
		}
		var relaciones struct {
			XMLName    xml.Name      `xml:"Relationships"`
			Relaciones []relacionXML `xml:"Relationship"`
		}
		if xml.Unmarshal(v, &relaciones) != nil {
			return nil, d.ErrHojaFormato
		}
		for _, r := range relaciones.Relaciones {
			if r.Modo != "" && r.Modo != "Internal" || strings.Contains(r.Destino, ":") || strings.Contains(r.Destino, "\\") || strings.Contains(r.Tipo, "externalLink") || strings.Contains(r.Tipo, "vbaProject") || strings.Contains(r.Tipo, "oleObject") {
				return nil, d.ErrHojaFormato
			}
		}
	}
	var tipos struct {
		XMLName   xml.Name `xml:"Types"`
		Overrides []struct {
			Parte string `xml:"PartName,attr"`
			Tipo  string `xml:"ContentType,attr"`
		} `xml:"Override"`
	}
	if xml.Unmarshal(partes["[Content_Types].xml"], &tipos) != nil {
		return nil, d.ErrHojaFormato
	}
	libroOK := false
	for _, t := range tipos.Overrides {
		if strings.Contains(strings.ToLower(t.Tipo), "macro") || strings.Contains(t.Tipo, "externalLink") {
			return nil, d.ErrHojaFormato
		}
		if t.Parte == "/xl/workbook.xml" && t.Tipo == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml" {
			libroOK = true
		}
	}
	var libro struct {
		XMLName xml.Name `xml:"workbook"`
		Hojas   []struct {
			ID     string `xml:"id,attr"`
			Estado string `xml:"state,attr"`
		} `xml:"sheets>sheet"`
	}
	if !libroOK || xml.Unmarshal(partes["xl/workbook.xml"], &libro) != nil || len(libro.Hojas) != 1 || libro.Hojas[0].Estado != "" && libro.Hojas[0].Estado != "visible" {
		return nil, d.ErrHojaFormato
	}
	var rels struct {
		Relaciones []relacionXML `xml:"Relationship"`
	}
	if xml.Unmarshal(partes["xl/_rels/workbook.xml.rels"], &rels) != nil {
		return nil, d.ErrHojaFormato
	}
	destino := ""
	for _, r := range rels.Relaciones {
		if r.ID == libro.Hojas[0].ID && strings.HasSuffix(r.Tipo, "/worksheet") {
			if destino != "" {
				return nil, d.ErrHojaFormato
			}
			if strings.HasPrefix(r.Destino, "/") {
				destino = strings.TrimPrefix(r.Destino, "/")
			} else {
				destino = path.Join("xl", r.Destino)
			}
		}
	}
	if !strings.HasPrefix(destino, "xl/worksheets/") || partes[destino] == nil {
		return nil, d.ErrHojaFormato
	}
	for nombre := range partes {
		if strings.HasPrefix(nombre, "xl/worksheets/") && nombre != destino {
			return nil, d.ErrHojaFormato
		}
	}
	compartidos := []string{}
	if v, ok := partes["xl/sharedStrings.xml"]; ok {
		var ss struct {
			XMLName xml.Name   `xml:"sst"`
			Items   []textoXML `xml:"si"`
		}
		if xml.Unmarshal(v, &ss) != nil || len(ss.Items) > 101*len(columnas) {
			return nil, d.ErrHojaLimites
		}
		for _, t := range ss.Items {
			compartidos = append(compartidos, t.valor())
		}
	}
	// Lectura incremental: no se materializa un árbol XML para la hoja completa.
	dec := xml.NewDecoder(bytes.NewReader(partes[destino]))
	rs := []registro{}
	previa := 0
	raiz := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, d.ErrHojaFormato
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !raiz {
			if start.Name.Local != "worksheet" {
				return nil, d.ErrHojaFormato
			}
			raiz = true
		}
		if start.Name.Local == "hyperlink" || start.Name.Local == "externalLink" || start.Name.Local == "oleObject" {
			return nil, d.ErrHojaFormato
		}
		if start.Name.Local == "col" {
			for _, attr := range start.Attr {
				if attr.Name.Local == "hidden" && (attr.Value == "1" || attr.Value == "true") {
					return nil, d.ErrHojaFormato
				}
			}
		}
		if start.Name.Local != "row" {
			continue
		}
		var fila struct {
			R      int    `xml:"r,attr"`
			Oculta string `xml:"hidden,attr"`
			Celdas []struct {
				R       string   `xml:"r,attr"`
				Tipo    string   `xml:"t,attr"`
				Valor   string   `xml:"v"`
				Formula *string  `xml:"f"`
				Texto   textoXML `xml:"is"`
			} `xml:"c"`
		}
		if dec.DecodeElement(&fila, &start) != nil {
			return nil, d.ErrHojaFormato
		}
		if len(fila.Celdas) == 0 {
			continue
		}
		if len(rs) >= 101 || len(fila.Celdas) > len(columnas) {
			return nil, d.ErrHojaLimites
		}
		if fila.R <= previa || fila.R > 1048576 || fila.Oculta == "1" || fila.Oculta == "true" {
			return nil, d.ErrHojaFormato
		}
		valores := []string{}
		colPrevia := 0
		for _, c := range fila.Celdas {
			col, row, err := coordenada(c.R)
			if err != nil || row != fila.R || col <= colPrevia || c.Formula != nil {
				return nil, d.ErrHojaFormato
			}
			if col > len(columnas) {
				return nil, d.ErrHojaLimites
			}
			for len(valores) < col {
				valores = append(valores, "")
			}
			v := c.Valor
			switch c.Tipo {
			case "inlineStr":
				v = c.Texto.valor()
			case "s":
				i, err := strconv.Atoi(c.Valor)
				if err != nil || i < 0 || i >= len(compartidos) {
					return nil, d.ErrHojaFormato
				}
				v = compartidos[i]
			case "", "n", "str", "d":
			default:
				return nil, d.ErrHojaFormato
			}
			valores[col-1] = v
			colPrevia = col
		}
		rs = append(rs, registro{fila: fila.R, valores: valores})
		previa = fila.R
	}
	if !raiz {
		return nil, d.ErrHojaFormato
	}
	return rs, nil
}

// El recorrido previo limita nodos y atributos antes de materializar listas de
// cadenas compartidas, fragmentos de texto o celdas. También evita que un solo
// token XML enorme amplifique memoria en el propio decodificador.
func revisarXML(ctx context.Context, b []byte) error {
	for i := 0; i < len(b); i++ {
		if b[i] != '<' {
			continue
		}
		inicio := i
		var fin []byte
		switch {
		case bytes.HasPrefix(b[i:], []byte("<!--")):
			fin = []byte("-->")
		case bytes.HasPrefix(b[i:], []byte("<![CDATA[")):
			j := bytes.Index(b[i+9:], []byte("]]>"))
			if j < 0 {
				return d.ErrHojaFormato
			}
			if j > d.MaxBytes {
				return d.ErrHojaLimites
			}
			i += 9 + j + 2
			continue
		case bytes.HasPrefix(b[i:], []byte("<?")):
			fin = []byte("?>")
		case bytes.HasPrefix(b[i:], []byte("<!")):
			return d.ErrHojaFormato
		}
		if fin != nil {
			j := bytes.Index(b[i:], fin)
			if j < 0 {
				return d.ErrHojaFormato
			}
			if j > 16384 {
				return d.ErrHojaLimites
			}
			i += j + len(fin) - 1
			continue
		}
		var comilla byte
		for i++; i < len(b); i++ {
			if i-inicio > 16384 {
				return d.ErrHojaLimites
			}
			if comilla != 0 {
				if b[i] == comilla {
					comilla = 0
				}
			} else if b[i] == '\'' || b[i] == '"' {
				comilla = b[i]
			} else if b[i] == '>' {
				break
			}
		}
		if i == len(b) {
			return d.ErrHojaFormato
		}
	}
	dec := xml.NewDecoder(bytes.NewReader(b))
	nodos, cadenas, filas, celdas, celdasFila, fragmentos := 0, 0, 0, 0, 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return d.ErrHojaFormato
		}
		nodos++
		if nodos > 50000 {
			return d.ErrHojaLimites
		}
		inicio, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if len(inicio.Attr) > 32 {
			return d.ErrHojaLimites
		}
		switch inicio.Name.Local {
		case "f", "hyperlink", "externalLink", "oleObject":
			return d.ErrHojaFormato
		case "si":
			cadenas++
			fragmentos = 0
		case "is":
			fragmentos = 0
		case "r":
			fragmentos++
		case "row":
			filas++
			celdasFila = 0
		case "c":
			celdas++
			celdasFila++
		}
		if cadenas > 101*len(columnas) || filas > 101 || celdas > 101*len(columnas) || celdasFila > len(columnas) || fragmentos > 64 {
			return d.ErrHojaLimites
		}
	}
}

func coordenada(s string) (int, int, error) {
	i, col := 0, 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		col = col*26 + int(s[i]-'A') + 1
		i++
		if col > 16384 {
			return 0, 0, d.ErrHojaFormato
		}
	}
	row, err := strconv.Atoi(s[i:])
	if i == 0 || err != nil || row < 1 {
		return 0, 0, d.ErrHojaFormato
	}
	return col, row, nil
}
