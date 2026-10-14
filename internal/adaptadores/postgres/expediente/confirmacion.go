package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

const rutaConfirmacion = "/api/coordinacion/derivaciones/confirmar"

func (*Repositorio) RepetirConfirmacion(ctx context.Context, usuario, clave, huella string, ahora time.Time) (dominio.ReciboDerivacion, bool, error) {
	return repeticion[dominio.ReciboDerivacion](ctx, usuario, clave, rutaConfirmacion, huella, ahora)
}
func (*Repositorio) RecordarConfirmacion(ctx context.Context, usuario, clave, huella string, r dominio.ReciboDerivacion, ahora time.Time) error {
	return recordar(ctx, usuario, clave, rutaConfirmacion, huella, r, ahora)
}

//etf:tecnico
func (*Repositorio) BloquearLectura(ctx context.Context, id string) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	if _, err := e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,697052))`, id); err != nil {
		return err
	} //etf:tecnico
	var existe bool
	if err := e.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expediente.lectura_anexo WHERE id=$1)`, id).Scan(&existe); err != nil {
		return err
	} //etf:tecnico
	if existe {
		return dominio.ErrEstado
	}
	return nil
}
