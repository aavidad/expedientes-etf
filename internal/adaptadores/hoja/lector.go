// Package hoja lee el formato de intercambio histórico v1 exclusivamente en memoria.
package hoja

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"io"
	"strings"
	"unicode/utf8"
)

type Lector struct{}
type registro struct {
	fila    int
	valores []string
}

// Columnas son identificadores del contrato, no etiquetas visibles.
var columnas = []string{"numero", "fecha_derivacion", "estado", "tipo_anexo", "fecha_aceptacion", "municipio_id", "centro_ss_id", "subprograma", "profesional_referencia_puesto", "profesional_referencia_id", "descripcion_situacion", "sexos_menores", "fecha_no_aceptacion", "causa_no_aceptacion", "campos_derivacion", "episodios", "familia", "menores"}

func (Lector) LeerHoja(ctx context.Context, formato string, b []byte) ([]d.FilaHoja, error) {
	if len(b) == 0 || len(b) > d.MaxBytes {
		return nil, d.ErrHojaLimites
	}
	var rs []registro
	var err error
	switch formato {
	case "csv":
		rs, err = leerCSV(ctx, b)
	case "xlsx":
		rs, err = leerXLSX(ctx, b)
	default:
		return nil, d.ErrHojaFormato
	}
	if err != nil {
		return nil, err
	}
	if len(rs) < 2 || len(rs) > 101 {
		return nil, d.ErrHojaLimites
	}
	indices := map[string]int{}
	permitidas := map[string]bool{}
	for _, c := range columnas {
		permitidas[c] = true
	}
	for i, c := range rs[0].valores {
		c = strings.TrimSpace(c)
		if _, repetida := indices[c]; !permitidas[c] || repetida {
			return nil, d.ErrHojaFormato
		}
		indices[c] = i
	}
	for _, c := range []string{"fecha_derivacion", "estado", "tipo_anexo"} {
		if _, existe := indices[c]; !existe {
			return nil, d.ErrHojaFormato
		}
	}
	// Ninguna celda del contrato necesita más de 64 KiB, y el JSON de todas
	// las celdas se cuenta antes de materializarlo: una cadena compartida o
	// un array de objetos vacíos no puede multiplicar la memoria del proceso.
	for _, r := range rs {
		for _, v := range r.valores {
			if len(v) > maxCelda {
				return nil, d.ErrHojaLimites
			}
		}
	}
	presupuesto := &presupuestoJSON{porArray: 500, total: 20000}
	excedido := false
	filas := make([]d.FilaHoja, 0, len(rs)-1)
	for _, r := range rs[1:] {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		f := d.FilaHoja{Numero: r.fila}
		valor := func(c string) string {
			i, ok := indices[c]
			if !ok || i >= len(r.valores) {
				return ""
			}
			return strings.TrimSpace(r.valores[i])
		}
		if len(r.valores) > len(rs[0].valores) {
			f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Clave: "expediente.hoja_columnas"})
		}
		c := d.CasoHistorico{NumeroHistorico: d.NumeroHistorico{Numero: valor("numero")}, FechaDerivacion: valor("fecha_derivacion"), Estado: valor("estado")}
		c.Datos.EsquemaVersion = "1"
		c.Datos.Extraccion.Tipo = valor("tipo_anexo")
		decode := func(columna string, destino any) {
			v := valor(columna)
			if v == "" || excedido {
				return
			}
			// Un JSON mal formado sigue siendo una incidencia de la fila.
			if errors.Is(presupuesto.revisar([]byte(v)), errLimiteJSON) {
				excedido = true
				return
			}
			var elementos []json.RawMessage
			if json.Unmarshal([]byte(v), &elementos) != nil || elementos == nil {
				f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Columna: columna, Clave: "expediente.hoja_json"})
				return
			}
			for _, elemento := range elementos {
				if bytes.Equal(bytes.TrimSpace(elemento), []byte("null")) {
					f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Columna: columna, Clave: "expediente.hoja_json"})
					return
				}
			}
			dec := json.NewDecoder(strings.NewReader(v))
			dec.DisallowUnknownFields()
			if err := dec.Decode(destino); err != nil || !errors.Is(dec.Decode(new(any)), io.EOF) || v == "null" {
				f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Columna: columna, Clave: "expediente.hoja_json"})
			}
		}
		decode("campos_derivacion", &c.Datos.Extraccion.Campos)
		decode("episodios", &c.Episodios)
		decode("familia", &c.Familia)
		decode("menores", &c.Menores)
		if c.Estado == d.NoAceptada {
			c.NoAceptacion = &d.NoAceptacion{Version: 1, Fecha: valor("fecha_no_aceptacion"), Motivo: valor("causa_no_aceptacion"), Subprograma: valor("subprograma")}
			for _, col := range []string{"fecha_aceptacion", "municipio_id", "centro_ss_id", "profesional_referencia_puesto", "profesional_referencia_id", "descripcion_situacion", "sexos_menores"} {
				if valor(col) != "" {
					f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Columna: col, Clave: "expediente.hoja_no_aceptada"})
				}
			}
		} else {
			a := &d.Aceptacion{Version: 1, FechaActa: valor("fecha_aceptacion"), MunicipioID: valor("municipio_id"), CentroID: valor("centro_ss_id"), Subprograma: valor("subprograma"), ProfesionalPuesto: valor("profesional_referencia_puesto"), Descripcion: valor("descripcion_situacion")}
			if id := valor("profesional_referencia_id"); id != "" {
				a.ProfesionalID = &id
			}
			decode("sexos_menores", &a.Menores)
			c.Aceptacion = a
			for _, col := range []string{"fecha_no_aceptacion", "causa_no_aceptacion"} {
				if valor(col) != "" {
					f.Incidencias = append(f.Incidencias, d.IncidenciaHoja{Fila: r.fila, Columna: col, Clave: "expediente.hoja_aceptada"})
				}
			}
		}
		if excedido {
			return nil, d.ErrHojaLimites
		}
		f.Caso = c
		filas = append(filas, f)
	}
	return filas, nil
}

const maxCelda = 64 << 10

func leerCSV(ctx context.Context, b []byte) ([]registro, error) {
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(b) || bytes.ContainsRune(b, 0) {
		return nil, d.ErrHojaFormato
	}
	nuevo := func(separador rune) *csv.Reader {
		r := csv.NewReader(bytes.NewReader(b))
		r.Comma = separador
		r.FieldsPerRecord = -1
		return r
	}
	r := nuevo(',')
	primera, err := r.Read()
	if err != nil {
		return nil, d.ErrHojaFormato
	}
	if len(primera) == 1 {
		r = nuevo(';')
		primera, err = r.Read()
		if err != nil {
			return nil, d.ErrHojaFormato
		}
	}
	rs := []registro{{fila: 1, valores: primera}}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		v, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, d.ErrHojaFormato
		}
		linea, _ := r.FieldPos(0)
		if len(rs) >= 101 || len(v) > len(columnas) {
			return nil, d.ErrHojaLimites
		}
		rs = append(rs, registro{fila: linea, valores: v})
	}
	return rs, nil
}
