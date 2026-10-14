package catalogo

import (
	"context"
	c "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"time"
)

// LecturaFirmantes es la parte de lectura del puerto público de catálogo.
type LecturaFirmantes interface {
	Firmantes(context.Context) ([]c.Firmante, error)
}
type Firmantes struct{ Catalogo LecturaFirmantes }

func (f Firmantes) Vigente(ctx context.Context, rol string, fecha time.Time) (d.Firmante, error) {
	if f.Catalogo == nil {
		return d.Firmante{}, d.ErrFirmante
	}
	lista, err := f.Catalogo.Firmantes(ctx)
	if err != nil {
		return d.Firmante{}, d.ErrNoDisponible
	}
	fecha = c.FechaCivil(fecha)
	var elegido d.Firmante
	for _, f := range lista {
		if (f.Tipo == "firmante" || rol == "jefe_spm" && f.Tipo == "destinatario") && f.Rol == rol && !fecha.Before(c.FechaCivil(f.VigenteDesde)) && (f.VigenteHasta == nil || fecha.Before(c.FechaCivil(*f.VigenteHasta))) {
			if elegido.ID != "" {
				return d.Firmante{}, d.ErrFirmante
			}
			elegido = d.Firmante{ID: f.ID, Version: f.Version, Nombre: f.Nombre}
		}
	}
	if !d.FirmanteValido(elegido) {
		return d.Firmante{}, d.ErrFirmante
	}
	return elegido, nil
}
