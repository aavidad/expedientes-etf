package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"strconv"
	"strings"
)

// PrepararUbicacion publica nombres y grafías juntos, sin sobrescribir archivos.
func (p *Publicacion) PrepararUbicacion(ctx context.Context, clave string, textos map[string]string, variantes map[string][]string, esperadas map[string]int) (dominio.Preparada, error) {
	if err := p.referencias.Bloquear(ctx); err != nil {
		return dominio.Preparada{}, err
	}
	c, err := p.Catalogos(ctx)
	if err != nil {
		return dominio.Preparada{}, err
	}
	preparada := dominio.Preparada{Anteriores: map[string]int{}}
	for _, codigo := range []string{"es", "en"} {
		if c.Referencias[codigo] != esperadas[codigo] {
			return dominio.Preparada{}, &dominio.Conflicto{Actual: c.Referencias[codigo], Codigo: codigo}
		}
		c.Textos[codigo][clave] = textos[codigo]
		if grafias, ok := variantes[codigo]; ok {
			prefijo := clave + ".variante."
			for k := range c.Textos[codigo] {
				if strings.HasPrefix(k, prefijo) {
					delete(c.Textos[codigo], k)
				}
			}
			for i, texto := range grafias {
				c.Textos[codigo][prefijo+"v"+strconv.Itoa(i+1)] = texto
			}
		}
		r, err := p.disco.EscribirCatalogo(ctx, codigo, c.Textos[codigo])
		if err != nil {
			return dominio.Preparada{}, err
		}
		preparada.Anteriores[codigo] = esperadas[codigo]
		preparada.Referencias = append(preparada.Referencias, r)
	}
	return preparada, nil
}
