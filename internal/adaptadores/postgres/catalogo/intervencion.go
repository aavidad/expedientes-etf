package catalogo

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

var _ salida.CatalogosIntervencion = (*Repositorio)(nil)

func (*Repositorio) ObtenerIntervencion(ctx context.Context, corte string) (dominio.CatalogosIntervencion, error) {
	r := dominio.CatalogosIntervencion{Opciones: map[string]map[string]bool{}}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	rows, err := e.Query(ctx, `SELECT o.lista,o.codigo,o.admite_texto FROM catalogo.intervencion_opcion o JOIN catalogo.version v ON v.id=o.version_id WHERE v.ambito='intervencion' AND v.estado='publicada' AND v.vigente_desde<=$1::date AND (v.vigente_hasta IS NULL OR $1::date<v.vigente_hasta) ORDER BY o.lista,o.orden`, corte) //etf:tecnico
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var lista, codigo string
		var texto bool
		if err = rows.Scan(&lista, &codigo, &texto); err != nil {
			return r, err
		}
		if r.Opciones[lista] == nil {
			r.Opciones[lista] = map[string]bool{}
		}
		r.Opciones[lista][codigo] = texto
	}
	if err = rows.Err(); err != nil {
		return r, err
	}
	if len(r.Opciones) != 5 {
		return r, dominio.ErrNoDisponible
	}
	return r, nil
}
