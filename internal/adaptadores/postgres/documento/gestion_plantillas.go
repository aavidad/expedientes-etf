package documento

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (Repositorio) Listar(ctx context.Context) ([]d.VersionPlantilla, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := tx.Query(ctx, `SELECT p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante,
 p.id=r.plantilla_id,r.version,coalesce(p.subida_por::text,''),p.subida_en,p.antivirus,
 coalesce((SELECT jsonb_agg(jsonb_build_object('por',h.por,'en',h.en,'revision',h.revision) ORDER BY h.en DESC) FROM (
 SELECT a.activada_por::text por,a.activada_en en,a.revision FROM documento.activacion_plantilla a WHERE a.plantilla_id=p.id
 UNION ALL SELECT a.archivada_por::text,a.archivada_en,0::bigint FROM documento.archivo_plantilla a WHERE a.nueva_id=p.id
 ) h),'[]'::jsonb)
 FROM documento.plantilla p JOIN documento.referencia_activa r ON r.tipo_documento=p.tipo_documento AND r.idioma=p.idioma
 ORDER BY p.tipo_documento,p.idioma,p.subida_en DESC,p.id`)
	if err != nil {
		return nil, d.ErrNoDisponible
	} //etf:tecnico
	defer filas.Close()
	versiones := []d.VersionPlantilla{}
	for filas.Next() {
		var v d.VersionPlantilla
		var historia []byte
		p := &v.Plantilla
		if err := filas.Scan(&p.ID, &p.Tipo, &p.Idioma, &p.Version, &p.Ruta, &p.Huella, &p.RolFirmante, &v.Activa, &v.Revision, &v.SubidaPor, &v.SubidaEn, &v.Antivirus, &historia); err != nil {
			return nil, d.ErrNoDisponible
		}
		if err := json.Unmarshal(historia, &v.Activaciones); err != nil {
			return nil, d.ErrNoDisponible
		}
		v.Marcadores = "validos"
		versiones = append(versiones, v)
	}
	if filas.Err() != nil {
		return nil, d.ErrNoDisponible
	}
	return versiones, nil
}
func (Repositorio) Version(ctx context.Context, id string) (d.Plantilla, error) {
	var p d.Plantilla
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return p, err
	}
	err = tx.QueryRow(ctx, `SELECT id::text,tipo_documento,idioma,version,ruta_relativa,encode(huella_sha256,'hex'),rol_firmante FROM documento.plantilla WHERE id=$1`, id).Scan(&p.ID, &p.Tipo, &p.Idioma, &p.Version, &p.Ruta, &p.Huella, &p.RolFirmante) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return p, d.ErrPlantilla
	}
	if err != nil {
		return p, d.ErrNoDisponible
	}
	return p, nil
}
func (Repositorio) GuardarVersion(ctx context.Context, p d.Plantilla, previa int64) (d.ReferenciaPlantilla, error) {
	return cambiarPlantilla(ctx, p, previa, false)
}
func (Repositorio) Activar(ctx context.Context, p d.Plantilla, previa int64) (d.ReferenciaPlantilla, error) {
	return cambiarPlantilla(ctx, p, previa, true)
}
func cambiarPlantilla(ctx context.Context, p d.Plantilla, previa int64, activar bool) (d.ReferenciaPlantilla, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.ReferenciaPlantilla{}, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='20s'`); err != nil {
		return d.ReferenciaPlantilla{}, d.ErrNoDisponible
	}
	var aplicada bool
	var revision int64
	if activar {
		err = tx.QueryRow(ctx, `SELECT aplicada,revision FROM documento.activar_version_plantilla($1,$2)`, p.ID, previa).Scan(&aplicada, &revision) //etf:tecnico
	} else {
		err = tx.QueryRow(ctx, `SELECT aplicada,revision FROM documento.guardar_version_plantilla($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.Tipo, p.Idioma, p.Version, p.Ruta, p.Huella, previa).Scan(&aplicada, &revision) //etf:tecnico
	}
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
