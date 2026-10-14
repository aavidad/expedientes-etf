package catalogo

import (
	"context"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
)

var _ salida.MotivosCierre = (*Repositorio)(nil)

//etf:tecnico
func (r *Repositorio) CierreVigente(ctx context.Context, fecha time.Time) (dominio.CatalogoCierre, error) {
	version, err := r.Vigente(ctx, "equivalencias", fecha)
	if err != nil {
		return dominio.CatalogoCierre{}, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.CatalogoCierre{}, err
	}
	filas, err := tx.Query(ctx, `SELECT codigo,acta,spaf,admite_texto,sin_consentimiento
 FROM catalogo.motivo_cierre WHERE version_id=$1 ORDER BY orden`, version.ID)
	if err != nil {
		return dominio.CatalogoCierre{}, dominio.ErrPersistencia
	}
	defer filas.Close()
	resultado := dominio.CatalogoCierre{VersionID: version.ID, Motivos: []dominio.MotivoCierre{}}
	for filas.Next() {
		var motivo dominio.MotivoCierre
		if err := filas.Scan(&motivo.Codigo, &motivo.Acta, &motivo.SPAF, &motivo.AdmiteTexto, &motivo.SinConsentimiento); err != nil {
			return dominio.CatalogoCierre{}, dominio.ErrPersistencia
		}
		resultado.Motivos = append(resultado.Motivos, motivo)
	}
	if filas.Err() != nil {
		return dominio.CatalogoCierre{}, dominio.ErrPersistencia
	}
	if len(resultado.Motivos) == 0 {
		return dominio.CatalogoCierre{}, dominio.ErrNoDisponible
	}
	return resultado, nil
}
