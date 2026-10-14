package documento

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/jackc/pgx/v5"
)

func (Repositorio) LeerRegistro(ctx context.Context, id string) (d.Emision, d.RegistroSalida, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Emision{}, d.RegistroSalida{}, err
	}
	_, err = tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='20s'`)
	if err != nil {
		return d.Emision{}, d.RegistroSalida{}, d.ErrNoDisponible
	}
	e, err := leer(tx.QueryRow(ctx, `SELECT `+columnas+` FROM documento.emision e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.id=$1`, id)) //etf:tecnico
	if err != nil || e.ID == "" || !d.EsOficio(e.Tipo) {
		return e, d.RegistroSalida{}, err
	}
	var r d.RegistroSalida
	err = tx.QueryRow(ctx, `SELECT emision_id::text,adaptador,estado,coalesce(numero_registro,''),coalesce(to_char(fecha_registro,'YYYY-MM-DD'),''),coalesce(referencia_externa,''),coalesce(anotado_por::text,''),coalesce(anotado_en,'0001-01-01'::timestamptz),coalesce(error_clave,''),version FROM documento.registro_salida WHERE emision_id=$1 FOR UPDATE`, id).Scan(&r.EmisionID, &r.Adaptador, &r.Estado, &r.Numero, &r.Fecha, &r.ReferenciaExterna, &r.AnotadoPor, &r.AnotadoEn, &r.ErrorClave, &r.Version) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Emision{}, d.RegistroSalida{}, d.ErrAcceso
	}
	if err != nil {
		return d.Emision{}, d.RegistroSalida{}, d.ErrNoDisponible
	}
	return e, r, nil
}
func (Repositorio) GuardarRegistro(ctx context.Context, r d.RegistroSalida, version int) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE documento.registro_salida SET adaptador=$2,estado=$3,numero_registro=$4,fecha_registro=$5::date,referencia_externa=nullif($6,''),anotado_por=$7,anotado_en=$8,error_clave=nullif($9,''),version=$10 WHERE emision_id=$1 AND version=$11`, r.EmisionID, r.Adaptador, r.Estado, r.Numero, r.Fecha, r.ReferenciaExterna, r.AnotadoPor, r.AnotadoEn, r.ErrorClave, r.Version, version) //etf:tecnico
	if err != nil || tag.RowsAffected() != 1 {
		return d.ErrNoDisponible
	}
	return nil
}

func (Repositorio) PeticionRegistro(ctx context.Context, a d.Actor, p d.SolicitudRegistroSalida, huella string) (d.RegistroSalida, bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.RegistroSalida{}, false, d.ErrNoDisponible
	}
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "documento.registro:"+a.UsuarioID+":"+p.Clave) //etf:tecnico
	if err != nil {
		return d.RegistroSalida{}, false, d.ErrNoDisponible
	}
	var anterior, ruta string
	var respuesta []byte
	err = tx.QueryRow(ctx, `SELECT encode(huella_peticion,'hex'),ruta,respuesta FROM documento.registro_peticion WHERE usuario_id=$1 AND clave=$2 AND creada_en>clock_timestamp()-interval '24 hours'`, a.UsuarioID, p.Clave).Scan(&anterior, &ruta, &respuesta) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.RegistroSalida{}, false, nil
	}
	if err != nil {
		return d.RegistroSalida{}, false, d.ErrNoDisponible
	}
	if anterior != huella || ruta != "/api/documentos/emisiones/"+p.EmisionID+"/registro-salida" {
		return d.RegistroSalida{}, false, d.ErrIdempotencia
	}
	var r d.RegistroSalida
	if json.Unmarshal(respuesta, &r) != nil {
		return r, false, d.ErrNoDisponible
	}
	return r, true, nil
}
func (Repositorio) RecordarRegistro(ctx context.Context, a d.Actor, p d.SolicitudRegistroSalida, huella string, r d.RegistroSalida) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.ErrNoDisponible
	}
	b, err := json.Marshal(r)
	if err != nil {
		return d.ErrNoDisponible
	}
	tag, err := tx.Exec(ctx, `INSERT INTO documento.registro_peticion(usuario_id,clave,ruta,emision_id,huella_peticion,respuesta,creada_en) VALUES($1,$2,$3,$4,decode($5,'hex'),$6,clock_timestamp()) ON CONFLICT(usuario_id,clave) DO UPDATE SET ruta=EXCLUDED.ruta,emision_id=EXCLUDED.emision_id,huella_peticion=EXCLUDED.huella_peticion,respuesta=EXCLUDED.respuesta,creada_en=EXCLUDED.creada_en WHERE documento.registro_peticion.creada_en<=clock_timestamp()-interval '24 hours'`, a.UsuarioID, p.Clave, "/api/documentos/emisiones/"+p.EmisionID+"/registro-salida", p.EmisionID, huella, b) //etf:tecnico
	if err != nil || tag.RowsAffected() != 1 {
		return d.ErrNoDisponible
	}
	return nil
}
