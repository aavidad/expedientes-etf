package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"maps"
	"strings"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
)

// Consulta conecta la lectura de paquetes con sus puertos, sin conocer el disco.
type Consulta struct {
	textos salida.Textos
	temas  salida.Temas
}

var _ entrada.Paquetes = (*Consulta)(nil)

func NuevaConsulta(textos salida.Textos, temas salida.Temas) *Consulta {
	return &Consulta{textos: textos, temas: temas}
}

func (c *Consulta) Textos(ctx context.Context, codigo string) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !dominio.CodigoIdiomaValido(codigo) {
		return nil, dominio.ErrNoDisponible
	}
	base, err := c.textos.Textos(ctx, dominio.IdiomaPredeterminado)
	if err != nil {
		return nil, err
	}
	if codigo == dominio.IdiomaPredeterminado {
		return base, nil
	}
	traduccion, err := c.textos.Textos(ctx, codigo)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return base, nil
	}
	base = maps.Clone(base)
	for clave, valor := range traduccion {
		if strings.TrimSpace(valor) != "" {
			base[clave] = valor
		}
	}
	return base, nil
}

func (c *Consulta) Tokens(ctx context.Context, nombre string) (map[string]string, error) {
	if !dominio.NombreTemaValido(nombre) {
		return nil, dominio.ErrNoDisponible
	}
	_, tokens, err := c.tema(ctx, nombre)
	if err == nil {
		return tokens, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if nombre == dominio.TemaPredeterminado {
		return nil, err
	}
	_, tokens, err = c.tema(ctx, dominio.TemaPredeterminado)
	return tokens, err
}

func (c *Consulta) Idiomas(ctx context.Context) ([]entrada.Idioma, error) {
	return c.textos.Idiomas(ctx)
}
