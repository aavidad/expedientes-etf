package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Errores internos del recorrido: nunca llegan a la respuesta HTTP.
type errorJSON string

func (e errorJSON) Error() string { return string(e) }

const (
	errLimiteJSON   errorJSON = "json_limite"
	errJSONInvalido errorJSON = "json_invalido"
)

// presupuestoJSON recorre un JSON sin materializarlo y falla antes de
// decodificar si algún array supera porArray elementos, si la suma de
// elementos de todos los arrays revisados supera total o si anida demasiado.
// Así un cuerpo pequeño de objetos vacíos no se convierte en millones de
// estructuras en memoria.
type presupuestoJSON struct {
	porArray, total, usados int
}

func (p *presupuestoJSON) revisar(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := p.valor(dec, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errJSONInvalido
	}
	return nil
}

func (p *presupuestoJSON) valor(dec *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return errLimiteJSON
	}
	t, err := dec.Token()
	if err != nil {
		return errJSONInvalido
	}
	delim, compuesto := t.(json.Delim)
	if !compuesto {
		return nil
	}
	n := 0
	for dec.More() {
		if delim == '{' {
			if _, err := dec.Token(); err != nil {
				return errJSONInvalido
			}
		} else {
			n++
			p.usados++
			if n > p.porArray || p.usados > p.total {
				return errLimiteJSON
			}
		}
		if err := p.valor(dec, profundidad+1); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil {
		return errJSONInvalido
	}
	return nil
}

// acotarJSON lee el cuerpo ya limitado por MaxBytesReader y comprueba su
// presupuesto de elementos antes de que el manejador lo decodifique y el
// servicio lo copie o serialice. Un JSON mal formado pasa para que el
// manejador responda su error habitual; sólo falla al leer o al exceder.
func acotarJSON(r *http.Request) bool {
	b, err := io.ReadAll(r.Body)
	if err != nil || errors.Is((&presupuestoJSON{porArray: 500, total: 20000}).revisar(b), errLimiteJSON) {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return true
}
