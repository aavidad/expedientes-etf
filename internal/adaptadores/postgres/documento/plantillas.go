package documento

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r Repositorio) Actual(ctx context.Context, tipo, idioma string) (d.ReferenciaPlantilla, error) {
	var referencia d.ReferenciaPlantilla
	var p d.Plantilla
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return referencia, err
	}
	// Una sola consulta fija plantilla y revisión de la misma instantánea.
	err = tx.QueryRow(ctx, `SELECT p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante,r.version FROM documento.plantilla p JOIN documento.referencia_activa r ON r.plantilla_id=p.id WHERE r.tipo_documento=$1 AND r.idioma=$2`, tipo, idioma).Scan(&p.ID, &p.Tipo, &p.Idioma, &p.Version, &p.Ruta, &p.Huella, &p.RolFirmante, &referencia.Revision) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.ReferenciaPlantilla{}, d.ErrPlantilla
	}
	if err != nil {
		return d.ReferenciaPlantilla{}, d.ErrNoDisponible
	}
	referencia.Plantilla = p
	return referencia, nil
}
func (Repositorio) Publicar(ctx context.Context, p d.Plantilla, previa int64) (d.ReferenciaPlantilla, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.ReferenciaPlantilla{}, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='20s'`); err != nil {
		return d.ReferenciaPlantilla{}, d.ErrNoDisponible
	}
	var aplicada bool
	var revision int64
	err = tx.QueryRow(ctx, `SELECT aplicada,revision FROM documento.publicar_plantilla($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.Tipo, p.Idioma, p.Version, p.Ruta, p.Huella, previa).Scan(&aplicada, &revision) //etf:tecnico
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return d.ReferenciaPlantilla{}, d.ErrVersionPlantilla
	}
	if err != nil {
		return d.ReferenciaPlantilla{}, d.ErrNoDisponible
	}
	if !aplicada {
		return d.ReferenciaPlantilla{}, &d.ConflictoPlantilla{VersionActual: revision}
	}
	return d.ReferenciaPlantilla{Plantilla: p, Revision: revision}, nil
}
func (Repositorio) Registrada(ctx context.Context, p d.Plantilla) (bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='6s'`); err != nil {
		return false, d.ErrNoDisponible
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "documento.plantilla:"+p.Tipo+":"+p.Idioma); err != nil { //etf:tecnico
		return false, d.ErrNoDisponible
	}
	var registrada bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documento.plantilla WHERE id=$1 AND ruta_relativa=$2)`, p.ID, p.Ruta).Scan(&registrada) //etf:tecnico
	if err != nil {
		return false, d.ErrNoDisponible
	}
	return registrada, nil
}
