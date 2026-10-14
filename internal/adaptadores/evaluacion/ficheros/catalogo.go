package ficheros

import (
	"context"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	"os"
	"path/filepath"
)

// Catalogo lee definiciones inmutables. No acepta definiciones de la petición.
type Catalogo struct{ Ruta string }

func (c Catalogo) leer() (struct {
	Instrumentos []string   `json:"codigos"`
	Escalas      []d.Escala `json:"escalas"`
}, error) {
	var r struct {
		Instrumentos []string   `json:"codigos"`
		Escalas      []d.Escala `json:"escalas"`
	}
	b, e := os.ReadFile(filepath.Clean(c.Ruta))
	if e != nil {
		return r, d.ErrNoDisponible
	}
	if json.Unmarshal(b, &r) != nil {
		return r, d.ErrNoDisponible
	}
	return r, nil
}
func (c Catalogo) Instrumento(_ context.Context, codigo string) (bool, error) {
	r, e := c.leer()
	if e != nil {
		return false, e
	}
	for _, v := range r.Instrumentos {
		if v == codigo {
			return true, nil
		}
	}
	return false, nil
}
func (c Catalogo) Escala(_ context.Context, id string) (d.Escala, error) {
	r, e := c.leer()
	if e != nil {
		return d.Escala{}, e
	}
	for _, v := range r.Escalas {
		if v.ID == id {
			if v.Validar() != nil {
				return d.Escala{}, d.ErrVersionEscala
			}
			return v, nil
		}
	}
	return d.Escala{}, d.ErrVersionEscala
}

func (c Catalogo) Definiciones(_ context.Context) (d.Definiciones, error) {
	r, err := c.leer()
	if err != nil {
		return d.Definiciones{}, err
	}
	for _, escala := range r.Escalas {
		if escala.Validar() != nil {
			return d.Definiciones{}, d.ErrVersionEscala
		}
	}
	return d.Definiciones{Instrumentos: r.Instrumentos, Escalas: r.Escalas}, nil
}
