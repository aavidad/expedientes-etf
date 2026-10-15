package catalogo

import (
	"context"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

// NombresUbicaciones sirve las grafías publicadas al lector de anexos.
// El llamador proporciona la transacción y confirma su auditoría.
type NombresUbicaciones struct {
	Publicacion paquetes.PublicacionCatalogos
	Idioma      string
}

func (n NombresUbicaciones) Nombres(ctx context.Context) (map[string]string, error) {
	return n.NombresEn(ctx, n.Idioma)
}
func (n NombresUbicaciones) NombresEn(ctx context.Context, idioma string) (map[string]string, error) {
	c, err := n.Publicacion.Catalogos(ctx)
	if err != nil {
		return nil, err
	}
	textos, ok := c.Textos[idioma]
	if !ok {
		return nil, dominio.ErrNoDisponible
	}
	return textos, nil
}
