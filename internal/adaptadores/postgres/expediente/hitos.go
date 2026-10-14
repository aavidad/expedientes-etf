package expediente

import (
	"context"
	"errors"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
)

var _ salida.Hitos = (*Repositorio)(nil)

func (*Repositorio) RepeticionHito(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoHito, bool, error) {
	return repeticion[dominio.ResultadoHito](ctx, usuario, clave, ruta, huella, ahora)
}
func (*Repositorio) RecordarHito(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoHito, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}
func (*Repositorio) BloquearHito(ctx context.Context, id string) (dominio.SituacionHitos, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.SituacionHitos{}, err
	}
	var d dominio.SituacionHitos
	err = e.QueryRow(ctx, `SELECT id::text,estado,coalesce(equipo_id::text,''),coalesce(subprograma,''),coalesce(to_char(fecha_aceptacion,'YYYY-MM-DD'),''),version
 FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.Estado, &d.EquipoID, &d.Subprograma, &d.FechaAceptacion, &d.Version) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d, dominio.ErrNoEncontrada
	}
	return d, err
}
func (*Repositorio) ConfirmarHito(ctx context.Context, d dominio.SituacionHitos, p dominio.ConfirmacionHito, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	// El bloqueo del expediente serializa tanto el hito como la comprobación de duplicados.
	var existe bool
	if err := e.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=$1 AND tipo=$2)`, d.ID, p.Tipo).Scan(&existe); err != nil { //etf:tecnico
		return err
	}
	if existe {
		return dominio.ErrEstado
	}
	fecha, _ := time.Parse("2006-01-02", p.Fecha)
	tag, err := e.Exec(ctx, `INSERT INTO expediente.hito(expediente_id,equipo_id,episodio_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 SELECT $1,equipo_id,id,$2,$3,'documento_externo',$4,$5 FROM expediente.episodio WHERE expediente_id=$1 AND cierre_fecha IS NULL`, d.ID, p.Tipo, fecha, usuario, ahora)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return dominio.ErrEstado
	}
	estado := d.Estado
	if p.Tipo == "consentimiento" {
		estado = dominio.EnIntervencion
	}
	tag, err = e.Exec(ctx, `UPDATE expediente.expediente SET estado=$2,version=version+1,modificado_por=$3,modificado_en=$4
 WHERE id=$1 AND version=$5 AND estado=$6`, d.ID, estado, usuario, ahora, p.Version, d.Estado)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return dominio.ErrEstado
	}
	if estado != d.Estado {
		_, err = e.Exec(ctx, `INSERT INTO expediente.estado_cambio(expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
 VALUES($1,$2,$3,$4,$5,$6)`, d.ID, d.EquipoID, d.Estado, estado, usuario, ahora)
	}
	return err
}
func (*Repositorio) ConsultarHitos(ctx context.Context, id string) ([]dominio.Hito, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := e.Query(ctx, `SELECT id::text,tipo,to_char(fecha,'YYYY-MM-DD'),origen,confirmado_por::text,confirmado_en
 FROM expediente.hito WHERE expediente_id=$1 ORDER BY fecha,confirmado_en,id`, id) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	hitos := []dominio.Hito{}
	for filas.Next() {
		var h dominio.Hito
		if err := filas.Scan(&h.ID, &h.Tipo, &h.Fecha, &h.Origen, &h.ConfirmadoPor, &h.ConfirmadoEn); err != nil {
			return nil, err
		}
		h.ConfirmadoEn = h.ConfirmadoEn.UTC()
		hitos = append(hitos, h)
	}
	return hitos, filas.Err()
}
