package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
	"time"
)

// Publicacion comparte el contexto de transacción del caso de uso propietario.
// Los ficheros se preparan antes de la transacción que activa sus referencias.
type Publicacion struct {
	disco       salida.FicherosCatalogo
	referencias salida.Referencias
}

func NuevaPublicacion(d salida.FicherosCatalogo, r salida.Referencias) *Publicacion {
	return &Publicacion{d, r}
}
func (p *Publicacion) Catalogos(ctx context.Context) (dominio.Catalogos, error) {
	c := dominio.Catalogos{Referencias: map[string]int{}, Textos: map[string]map[string]string{}}
	for _, codigo := range []string{"es", "en"} {
		r, err := p.referencias.Consultar(ctx, codigo)
		if err != nil {
			return dominio.Catalogos{}, err
		}
		m, err := p.disco.LeerCatalogo(ctx, r)
		if err != nil {
			return dominio.Catalogos{}, err
		}
		c.Referencias[codigo] = r.Version
		c.Textos[codigo] = m
	}
	return c, nil
}
func (p *Publicacion) Preparar(ctx context.Context, clave string, textos map[string]string, esperadas map[string]int) (dominio.Preparada, error) {
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
		r, err := p.disco.EscribirCatalogo(ctx, codigo, c.Textos[codigo])
		if err != nil {
			return dominio.Preparada{}, err
		}
		preparada.Anteriores[codigo] = esperadas[codigo]
		preparada.Referencias = append(preparada.Referencias, r)
	}
	return preparada, nil
}
func (p *Publicacion) Activar(ctx context.Context, c dominio.Preparada) error {
	for _, r := range c.Referencias {
		// Verificar la huella antes de registrar o servir el contenido.
		if _, err := p.disco.LeerCatalogo(ctx, r); err != nil {
			return err
		}
		if err := p.referencias.Activar(ctx, r, c.Anteriores[r.Codigo]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Publicacion) TextosVersion(ctx context.Context, codigo, contenido string) (map[string]string, error) {
	r, err := p.referencias.Contenido(ctx, codigo, contenido)
	if err != nil {
		return nil, err
	}
	return p.disco.LeerCatalogo(ctx, r)
}

// La limpieza conserva todo contenido registrado, aunque ya no sea el activo.
// La antigüedad mínima evita borrar una preparación aún pendiente de activar.
func (p *Publicacion) LimpiarHuerfanos(ctx context.Context, antes time.Time) error {
	if err := p.referencias.Bloquear(ctx); err != nil {
		return err
	}
	conservadas, err := p.referencias.Conservadas(ctx)
	if err != nil {
		return err
	}
	for _, cod := range []string{"es", "en"} {
		if err := p.disco.LimpiarCatalogo(ctx, cod, conservadas[cod], antes); err != nil {
			return err
		}
	}
	return nil
}
