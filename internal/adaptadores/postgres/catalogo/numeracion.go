package catalogo

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"time"
)

var _ salida.Numeracion = (*Repositorio)(nil)

func (r *Repositorio) ReglaVigente(ctx context.Context, fecha time.Time) (dominio.ReglaNumeracion, error) {
	v, err := r.Vigente(ctx, "numeracion", dominio.FechaCivil(fecha))
	if err != nil {
		return dominio.ReglaNumeracion{}, err
	}
	return r.regla(ctx, v)
}
func (*Repositorio) regla(ctx context.Context, v dominio.Version) (dominio.ReglaNumeracion, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.ReglaNumeracion{}, err
	}
	r := dominio.ReglaNumeracion{Version: v}
	err = e.QueryRow(ctx, `SELECT componentes,cifras_equipo,cifras_correlativo,cifras_anio,separador FROM catalogo.regla_numeracion WHERE version_id=$1`, v.ID).Scan(&r.Componentes, &r.CifrasEquipo, &r.CifrasCorrelativo, &r.CifrasAnio, &r.Separador) //etf:tecnico
	if err != nil {
		return dominio.ReglaNumeracion{}, dominio.ErrPersistencia
	}
	return r, nil
}
