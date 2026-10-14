package catalogo

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
)

func (*Repositorio) duracionesFase(ctx context.Context, id string) ([]dominio.DuracionFase, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=$1 ORDER BY subprograma,fase`, id) //etf:tecnico
	if err != nil {
		return nil, dominio.ErrPersistencia
	}
	defer rows.Close()
	r := []dominio.DuracionFase{}
	for rows.Next() {
		var f dominio.DuracionFase
		if err := rows.Scan(&f.Subprograma, &f.Fase, &f.DiasOrientativos, &f.DiasAmpliable); err != nil {
			return nil, dominio.ErrPersistencia
		}
		r = append(r, f)
	}
	if rows.Err() != nil {
		return nil, dominio.ErrPersistencia
	}
	return r, nil
}
