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

func (*Repositorio) RepeticionCierre(ctx context.Context, u, c, r, h string, t time.Time) (d.PropuestaCierre, bool, error) {
	return repeticion[d.PropuestaCierre](ctx, u, c, r, h, t)
}
func (*Repositorio) RecordarCierre(ctx context.Context, u, c, r, h string, p d.PropuestaCierre, t time.Time) error {
	return recordar(ctx, u, c, r, h, p, t)
}
func (*Repositorio) LeerFinalizacion(ctx context.Context, id string) (*d.FinalizacionGuardada, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	var b []byte
	err = e.QueryRow(ctx, `SELECT datos FROM expediente.finalizacion WHERE expediente_id=$1`, id).Scan(&b) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer clear(b)
	var f d.FinalizacionGuardada
	err = json.Unmarshal(b, &f)
	return &f, err
}
func (*Repositorio) GuardarFinalizacion(ctx context.Context, id, equipo string, f d.FinalizacionGuardada, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	defer clear(b)
	_, err = e.Exec(ctx, `INSERT INTO expediente.finalizacion(expediente_id,equipo_id,datos,modificado_por,modificado_en) VALUES($1,$2,$3,$4,$5)
 ON CONFLICT(expediente_id) DO UPDATE SET datos=EXCLUDED.datos,modificado_por=EXCLUDED.modificado_por,modificado_en=EXCLUDED.modificado_en`, id, equipo, b, usuario, ahora)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `UPDATE expediente.expediente SET version=version+1,modificado_por=$2,modificado_en=$3 WHERE id=$1`, id, usuario, ahora) //etf:tecnico
	return err
}
func (*Repositorio) Pendiente(ctx context.Context, id string) (bool, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return false, err
	}
	var existe bool
	err = e.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expediente.propuesta_cierre WHERE expediente_id=$1 AND estado='pendiente')`, id).Scan(&existe) //etf:tecnico
	return existe, err
}
func (*Repositorio) Proponer(ctx context.Context, p d.PropuestaCierre) (d.PropuestaCierre, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return p, err
	}
	err = e.QueryRow(ctx, `INSERT INTO expediente.propuesta_cierre(expediente_id,equipo_id,propuesta_por,propuesta_en,motivo_finalizacion,catalogo_version_id,motivo_acta,motivo_spaf)
 SELECT $1,$2,$3,$4,datos->>'motivo',(datos->>'catalogo_version_id')::uuid,datos->>'motivo_acta',datos->>'motivo_spaf'
 FROM expediente.finalizacion WHERE expediente_id=$1 RETURNING id::text`, p.ExpedienteID, p.EquipoID, p.PropuestaPor, p.PropuestaEn).Scan(&p.ID)
	return p, err
}
func (*Repositorio) Retirar(ctx context.Context, id, u string, t time.Time) (d.PropuestaCierre, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.PropuestaCierre{}, err
	}
	var p d.PropuestaCierre
	err = e.QueryRow(ctx, `UPDATE expediente.propuesta_cierre SET estado='retirada',resuelta_por=$2,resuelta_en=$3 WHERE expediente_id=$1 AND estado='pendiente'
 RETURNING id::text,expediente_id::text,equipo_id::text,propuesta_por::text,propuesta_en,estado,motivo_finalizacion`, id, u, t).Scan(&p.ID, &p.ExpedienteID, &p.EquipoID, &p.PropuestaPor, &p.PropuestaEn, &p.Estado, &p.Motivo)
	return p, err
}
func (*Repositorio) ListarPropuestas(ctx context.Context, l, o int) ([]d.PropuestaCierre, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := e.Query(ctx, `SELECT p.id::text,p.expediente_id::text,p.equipo_id::text,p.propuesta_por::text,p.propuesta_en,p.estado,p.motivo_finalizacion,e.numero,e.version
 FROM expediente.propuesta_cierre p JOIN expediente.expediente e ON e.id=p.expediente_id WHERE p.estado='pendiente' ORDER BY p.propuesta_en,p.id LIMIT $1 OFFSET $2`, l, o)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	r := []d.PropuestaCierre{}
	for filas.Next() {
		var p d.PropuestaCierre
		if err := filas.Scan(&p.ID, &p.ExpedienteID, &p.EquipoID, &p.PropuestaPor, &p.PropuestaEn, &p.Estado, &p.Motivo, &p.Numero, &p.Version); err != nil {
			return nil, err
		}
		r = append(r, p)
	}
	return r, filas.Err()
}
