package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
)

func (r *Repositorio) ParaDocumentos(ctx context.Context, id string) (d.DatosParaDocumento, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.DatosParaDocumento{}, d.ErrNoDisponible
	}
	// La decisión y su colección de menores no cambian mientras se genera.
	var version *int
	var datos d.DatosParaDocumento
	var copia []byte
	err = tx.QueryRow(ctx, `SELECT expediente.bloquear_lectura_spm(id),to_char(fecha_recepcion AT TIME ZONE 'Europe/Madrid','YYYY-MM-DD'),to_jsonb(e)->'copia_traslado' FROM expediente.expediente e WHERE id=$1`, id).Scan(&version, &datos.FechaRecepcion, &copia) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.DatosParaDocumento{}, d.ErrNoEncontrada
	}
	if err != nil {
		return d.DatosParaDocumento{}, d.ErrNoDisponible
	}
	defer clear(copia)
	if len(copia) > 0 && string(copia) != "null" {
		if err = json.Unmarshal(copia, &datos.CopiaTraslado); err != nil {
			return d.DatosParaDocumento{}, d.ErrNoDisponible
		}
	}
	datos.Derivacion, err = r.Obtener(ctx, id)
	if err != nil {
		return datos, err
	}
	datos.Finalizacion, err = r.LeerFinalizacion(ctx, id)
	if err != nil {
		return datos, err
	}
	datos.ActaCierre, err = r.LeerActaCierre(ctx, id)
	return datos, err
}
