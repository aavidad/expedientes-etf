package ficheros

import (
	"context"
	"encoding/json"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"os"
	"strings"
)

// CatalogosMenores recibe la ruta elegida por composición.
type CatalogosMenores struct{ Ruta string }

func (c CatalogosMenores) Obtener(ctx context.Context) (dominio.CatalogosMenores, error) {
	r := dominio.CatalogosMenores{Opciones: map[string][]string{}}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	b, err := os.ReadFile(c.Ruta)
	if err != nil {
		return r, dominio.ErrNoDisponible
	}
	var textos map[string]string
	if json.Unmarshal(b, &textos) != nil {
		return r, dominio.ErrNoDisponible
	}
	for clave := range textos {
		lista, codigo, ok := strings.Cut(clave, ".")
		if !ok || codigo == "" {
			return r, dominio.ErrNoDisponible
		}
		r.Opciones[lista] = append(r.Opciones[lista], codigo)
	}
	return r, nil
}
