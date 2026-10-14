package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

//etf:tecnico
func (*Repositorio) LeerSPM(ctx context.Context, id string, bloquear bool) (*d.SalidaSPM, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	if bloquear {
		// También serializa el alta y la consulta de cierre cuando no hay salida.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,5301))`, id); err != nil {
			return nil, err
		} //etf:tecnico
		var version *int
		if err := tx.QueryRow(ctx, `SELECT expediente.bloquear_lectura_spm($1)`, id).Scan(&version); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, d.ErrNoEncontrada
			}
			return nil, err
		} //etf:tecnico
		if version == nil {
			return nil, d.ErrNoEncontrada
		}
	}
	var r d.SalidaSPM
	err = tx.QueryRow(ctx, `SELECT id::text,expediente_id::text,equipo_id::text,version,motivo,propuesta_medida,
 coalesce(to_char(fecha_derivacion,'YYYY-MM-DD'),''),respuesta,coalesce(to_char(fecha_respuesta,'YYYY-MM-DD'),'')
 FROM expediente.salida_spm WHERE expediente_id=$1`, id).Scan(&r.ID, &r.ExpedienteID, &r.EquipoID, &r.Version, &r.Motivo, &r.PropuestaMedida, &r.FechaDerivacion, &r.Respuesta, &r.FechaRespuesta) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.Respuesta == d.SPMAlaEspera {
		err = tx.QueryRow(ctx, `SELECT id::text,motivo FROM expediente.spm_forzado WHERE salida_id=$1 AND version_salida=$2`, r.ID, r.Version).Scan(&r.ForzadoID, &r.MotivoForzado)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	return &r, nil
}

//etf:tecnico
func (*Repositorio) GuardarSPM(ctx context.Context, accion string, r d.SalidaSPM, p d.PeticionSPM, usuario string, ahora time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	if accion == "crear" {
		_, err = tx.Exec(ctx, `INSERT INTO expediente.salida_spm(id,expediente_id,equipo_id,version,motivo,propuesta_medida,respuesta,creado_por,creado_en,modificado_por,modificado_en)
  VALUES($1,$2,$3,1,$4,$5,'a_la_espera',$6,$7,$6,$7)`, r.ID, r.ExpedienteID, r.EquipoID, r.Motivo, r.PropuestaMedida, usuario, ahora)
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE expediente.salida_spm SET version=$2,fecha_derivacion=nullif($3,'')::date,
 respuesta=$4,fecha_respuesta=nullif($5,'')::date,modificado_por=$6,modificado_en=$7 WHERE id=$1 AND version=$8`, r.ID, r.Version, r.FechaDerivacion, r.Respuesta, r.FechaRespuesta, usuario, ahora, p.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return d.ErrEstado
	}
	if accion == "forzar" {
		_, err = tx.Exec(ctx, `INSERT INTO expediente.spm_forzado(id,salida_id,expediente_id,equipo_id,version_salida,motivo,registrado_por,registrado_en)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.ForzadoID, r.ID, r.ExpedienteID, r.EquipoID, r.Version, p.Motivo, usuario, ahora)
	}
	if err == nil && (accion == "confirmar" || accion == "responder") {
		_, err = tx.Exec(ctx, `SELECT expediente.sincronizar_estado_spm($1)`, r.ExpedienteID)
	}
	return err
}

//etf:tecnico
func (*Repositorio) RepetirSPM(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (d.SalidaSPM, bool, error) {
	var r d.SalidaSPM
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, false, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,697051))`, usuario+":"+clave); err != nil {
		return r, false, err
	} //etf:tecnico
	if _, err := tx.Exec(ctx, `DELETE FROM expediente.peticion_spm WHERE usuario_id=$1 AND clave=$2 AND creada_en<=$3`, usuario, clave, ahora.Add(-24*time.Hour)); err != nil {
		return r, false, err
	} //etf:tecnico
	var anteriorRuta, anteriorHuella string
	var respuesta []byte
	err = tx.QueryRow(ctx, `SELECT ruta,huella_peticion,respuesta FROM expediente.peticion_spm WHERE usuario_id=$1 AND clave=$2`, usuario, clave).Scan(&anteriorRuta, &anteriorHuella, &respuesta) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	if anteriorRuta != ruta || anteriorHuella != huella {
		return r, false, d.ErrIdempotencia
	}
	err = json.Unmarshal(respuesta, &r)
	return r, true, err
}

//etf:tecnico
func (*Repositorio) RecordarSPM(ctx context.Context, usuario, clave, ruta, huella string, r d.SalidaSPM, ahora time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.peticion_spm(usuario_id,clave,ruta,huella_peticion,respuesta,creada_en) VALUES($1,$2,$3,$4,$5,$6)`, usuario, clave, ruta, huella, b, ahora)
	return err
}
