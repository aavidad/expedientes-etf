package ficheros

import (
	"context"
	"encoding/json"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"os"
	"strings"
)

// CatalogosFamilia recibe rutas del arranque; no fija idioma ni textos en código.
type CatalogosFamilia struct{ Listas, Familia, Parametros string }

func (c CatalogosFamilia) Obtener(ctx context.Context) (dominio.CatalogosFamilia, error) {
	r := dominio.CatalogosFamilia{Opciones: map[string][]string{}}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	for _, ruta := range []string{c.Listas, c.Familia} {
		b, err := os.ReadFile(ruta)
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
	}
	b, err := os.ReadFile(c.Parametros)
	if err != nil {
		return r, dominio.ErrNoDisponible
	}
	var p struct {
		Version string                 `json:"version"`
		Tramos  []dominio.TramoFamilia `json:"tramos"`
	}
	if json.Unmarshal(b, &p) != nil || p.Version == "" || len(p.Tramos) == 0 {
		return r, dominio.ErrNoDisponible
	}
	for i, t := range p.Tramos {
		if !r.Admite("tramo_ingresos_smi", &t.Codigo) || t.Desde < 0 || i == 0 && t.Desde != 0 || t.Hasta != nil && *t.Hasta <= t.Desde || i > 0 && (p.Tramos[i-1].Hasta == nil || *p.Tramos[i-1].Hasta != t.Desde) || i < len(p.Tramos)-1 && t.Hasta == nil || i == len(p.Tramos)-1 && t.Hasta != nil {
			return r, dominio.ErrNoDisponible
		}
	}
	r.Tramos = p.Tramos
	return r, nil
}
