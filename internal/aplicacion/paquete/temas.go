package paquete

import (
	"context"
	"maps"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

func (c *Consulta) tema(ctx context.Context, nombre string) (entrada.Tema, map[string]string, error) {
	var tema entrada.Tema
	if err := ctx.Err(); err != nil {
		return tema, nil, err
	}
	if !dominio.NombreTemaValido(nombre) {
		return tema, nil, dominio.ErrNoDisponible
	}
	tema, err := c.temas.Tema(ctx, nombre)
	if err != nil {
		return tema, nil, err
	}
	base, err := c.temas.Tokens(ctx, dominio.TemaPredeterminado)
	if err != nil {
		return tema, nil, err
	}
	if nombre != dominio.TemaPredeterminado {
		propios, err := c.temas.Tokens(ctx, nombre)
		if err != nil {
			return tema, nil, err
		}
		base = maps.Clone(base)
		maps.Copy(base, propios)
	}
	if !dominio.TokensValidos(base, tema.Tipo) {
		return tema, nil, dominio.ErrNoDisponible
	}
	return tema, base, nil
}
func (c *Consulta) Temas(ctx context.Context) ([]entrada.Tema, error) {
	lista, err := c.temas.Temas(ctx)
	if err != nil {
		return nil, err
	}
	validos := []entrada.Tema{}
	for _, tema := range lista {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, _, err := c.tema(ctx, tema.Identificador); err == nil {
			validos = append(validos, tema)
		}
	}
	return validos, nil
}
func (c *Consulta) RecursoTema(ctx context.Context, nombre, ruta string) (entrada.Recurso, error) {
	if _, _, err := c.tema(ctx, nombre); err != nil {
		return entrada.Recurso{}, err
	}
	return c.temas.RecursoTema(ctx, nombre, ruta)
}

func (c *Consulta) Tema(ctx context.Context, nombre string) (entrada.Tema, error) {
	tema, _, err := c.tema(ctx, nombre)
	if err == nil {
		return tema, nil
	}
	if ctx.Err() != nil {
		return tema, ctx.Err()
	}
	if !dominio.NombreTemaValido(nombre) || nombre == dominio.TemaPredeterminado {
		return tema, err
	}
	tema, _, err = c.tema(ctx, dominio.TemaPredeterminado)
	return tema, err
}
