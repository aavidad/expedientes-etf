package expediente

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

type ConsultaIntervencion struct{}

var _ salida.ConsultaIntervencion = ConsultaIntervencion{}

func (ConsultaIntervencion) IntervencionVigente(ctx context.Context, id, corte string) (dominio.HistoriaIntervencion, error) {
	if !dominio.IDValido(id) {
		return dominio.HistoriaIntervencion{}, dominio.ErrPeticion
	}
	if f, err := time.Parse("2006-01-02", corte); err != nil || f.Year() < 1 {
		return dominio.HistoriaIntervencion{}, dominio.ErrIntervencion
	}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.HistoriaIntervencion{}, err
	}
	// Comprueba visibilidad también cuando no hay revisión al corte.
	var visible string
	if err = e.QueryRow(ctx, `SELECT id::text FROM expediente.expediente WHERE id=$1`, id).Scan(&visible); err != nil { //etf:tecnico
		if errors.Is(err, pgx.ErrNoRows) {
			err = dominio.ErrNoEncontrada
		}
		return dominio.HistoriaIntervencion{}, err
	}
	r, err := (&Repositorio{}).HistoriaIntervencion(ctx, id, corte)
	if err != nil {
		return dominio.HistoriaIntervencion{}, err
	}
	_, err = e.Exec(ctx, `SELECT auditoria.registrar(transaction_timestamp(),$1,$2,$3::uuid,$3::uuid,$4)`, "expediente.leer_historia_intervencion", "expediente", id, "ok") //etf:tecnico
	if err != nil {
		return dominio.HistoriaIntervencion{}, err
	}
	return r, nil
}
