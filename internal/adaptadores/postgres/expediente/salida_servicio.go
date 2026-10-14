package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

func (r *Repositorio) BloquearSalida(ctx context.Context, id string) (d.Derivacion, error) { //etf:tecnico
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Derivacion{}, err
	}
	var version int
	var coincide bool
	err = tx.QueryRow(ctx, `SELECT actual,coincide FROM expediente.version_salida_servicio($1,NULL,false)`, id).Scan(&version, &coincide)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Message == d.ErrNoEncontrada.Error() {
			return d.Derivacion{}, d.ErrNoEncontrada
		}
		return d.Derivacion{}, err
	}
	return r.Obtener(ctx, id)
}

func (r *Repositorio) DatosSalida(ctx context.Context, id string) (d.DatosParaDocumento, error) { //etf:tecnico
	e, err := r.BloquearSalida(ctx, id)
	if err != nil {
		return d.DatosParaDocumento{}, err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.DatosParaDocumento{}, err
	}
	datos := d.DatosParaDocumento{Derivacion: e}
	err = tx.QueryRow(ctx, `SELECT to_char(fecha_recepcion AT TIME ZONE 'Europe/Madrid','YYYY-MM-DD') FROM expediente.expediente WHERE id=$1`, id).Scan(&datos.FechaRecepcion) //etf:tecnico
	return datos, err
}

func (*Repositorio) RepeticionSalida(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (d.SalidaServicio, bool, error) { //etf:tecnico
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.SalidaServicio{}, false, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,697052))`, usuario+":"+clave); err != nil {
		return d.SalidaServicio{}, false, err
	} //etf:tecnico
	if _, err = tx.Exec(ctx, `DELETE FROM expediente.salida_servicio_peticion WHERE usuario_id=$1 AND clave=$2 AND creada_en<=$3`, usuario, clave, ahora.Add(-24*time.Hour)); err != nil {
		return d.SalidaServicio{}, false, err
	} //etf:tecnico
	var anteriorRuta, anteriorHuella string
	var b []byte
	err = tx.QueryRow(ctx, `SELECT ruta,huella,respuesta FROM expediente.salida_servicio_peticion WHERE usuario_id=$1 AND clave=$2`, usuario, clave).Scan(&anteriorRuta, &anteriorHuella, &b) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.SalidaServicio{}, false, nil
	}
	if err != nil {
		return d.SalidaServicio{}, false, err
	}
	if ruta != anteriorRuta || huella != anteriorHuella {
		return d.SalidaServicio{}, false, d.ErrIdempotencia
	}
	var recibo d.SalidaServicio
	err = json.Unmarshal(b, &recibo)
	return recibo, err == nil, err
}

func (*Repositorio) RecordarSalida(ctx context.Context, usuario, clave, ruta, huella string, r d.SalidaServicio, ahora time.Time) error { //etf:tecnico
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO expediente.salida_servicio_peticion(usuario_id,clave,equipo_id,expediente_id,ruta,huella,respuesta,creada_en) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(usuario_id,clave) DO NOTHING`, usuario, clave, r.EquipoID, r.ExpedienteID, ruta, huella, b, ahora)
	if err == nil && tag.RowsAffected() != 1 {
		return d.ErrIdempotencia
	}
	return err
}

func incrementarSalida(ctx context.Context, r d.SalidaServicio) error { //etf:tecnico
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var version int
	var coincide bool
	if err = tx.QueryRow(ctx, `SELECT actual,coincide FROM expediente.version_salida_servicio($1,$2,true)`, r.ExpedienteID, r.VersionExpediente-1).Scan(&version, &coincide); err != nil {
		return err
	}
	if !coincide {
		return d.Conflicto{VersionActual: version}
	}
	return nil
}

func (*Repositorio) CrearSalida(ctx context.Context, r d.SalidaServicio, usuario string, ahora time.Time) error { //etf:tecnico
	if err := incrementarSalida(ctx, r); err != nil {
		return err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.salida_servicio(id,expediente_id,equipo_id,servicio_destino,catalogo_id,motivo,creada_por,creada_en) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.ID, r.ExpedienteID, r.EquipoID, r.Servicio, r.CatalogoID, r.Motivo, usuario, ahora)
	return err
}

func (*Repositorio) LeerSalida(ctx context.Context, expedienteID, id string) (d.SalidaServicio, error) { //etf:tecnico
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.SalidaServicio{}, err
	}
	var r d.SalidaServicio
	err = tx.QueryRow(ctx, `SELECT id::text,expediente_id::text,equipo_id::text,tipo,servicio_destino,catalogo_id::text,motivo,coalesce(to_char(fecha,'YYYY-MM-DD'),''),estado,version FROM expediente.salida_servicio WHERE id=$1 AND expediente_id=$2`, id, expedienteID).Scan(&r.ID, &r.ExpedienteID, &r.EquipoID, &r.Tipo, &r.Servicio, &r.CatalogoID, &r.Motivo, &r.Fecha, &r.Estado, &r.Version) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, d.ErrNoEncontrada
	}
	return r, err
}

func (*Repositorio) ConfirmarSalida(ctx context.Context, r d.SalidaServicio, usuario string, ahora time.Time) error { //etf:tecnico
	if err := incrementarSalida(ctx, r); err != nil {
		return err
	}
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	fecha, err := time.Parse("2006-01-02", r.Fecha)
	if err != nil {
		return d.ErrPeticion
	}
	tag, err := tx.Exec(ctx, `UPDATE expediente.salida_servicio SET fecha=$2,estado='emitida',version=version+1,confirmada_por=$3,confirmada_en=$4 WHERE id=$1 AND version=$5 AND estado='borrador'`, r.ID, fecha, usuario, ahora, r.Version-1)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return d.ErrEstado
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.salida_servicio_hito(salida_id,expediente_id,equipo_id,tipo,fecha,origen,confirmado_por,confirmado_en) VALUES($1,$2,$3,'derivacion_otro_servicio',$4,'documento_externo',$5,$6)`, r.ID, r.ExpedienteID, r.EquipoID, fecha, usuario, ahora)
	return err
}
