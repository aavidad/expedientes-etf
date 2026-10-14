package evaluacion

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repositorio struct{}

func traducir(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return d.ErrPeticion
		case "23503", "23514", "22P02":
			return d.ErrSujeto
		case "42501":
			return d.ErrAcceso
		}
	}
	return d.ErrNoDisponible
}

//etf:tecnico
func (*Repositorio) Consultar(ctx context.Context, id string) (d.Registro, error) {
	tx, e := postgres.DesdeContexto(ctx)
	if e != nil {
		return d.Registro{}, e
	}
	var b []byte
	e = tx.QueryRow(ctx, `SELECT jsonb_build_object(
 'instrumentos',coalesce((SELECT jsonb_agg(i.datos ORDER BY i.id) FROM evaluacion.instrumento_aplicado i WHERE i.expediente_id=$1),'[]'::jsonb),
 'escalas',coalesce((SELECT jsonb_agg(a.datos || jsonb_build_object('version',a.version,'puntuaciones',
 coalesce((SELECT jsonb_agg(jsonb_strip_nulls(jsonb_build_object('item',r.item,'valor_numerico',r.valor_numerico,'valor_codigo',r.valor_codigo)) ORDER BY r.item)
 FROM evaluacion.respuesta r WHERE r.aplicacion_id=a.id AND r.expediente_id=a.expediente_id AND r.equipo_id=a.equipo_id),'[]'::jsonb)) ORDER BY a.id)
 FROM evaluacion.aplicacion_escala a WHERE a.expediente_id=$1),'[]'::jsonb))`, id).Scan(&b)
	if e != nil {
		return d.Registro{}, traducir(e)
	}
	var r d.Registro
	if json.Unmarshal(b, &r) != nil {
		return d.Registro{}, d.ErrNoDisponible
	}
	return r, nil
}

//etf:tecnico
func (*Repositorio) Version(ctx context.Context, exp, id string) (int, error) {
	tx, e := postgres.DesdeContexto(ctx)
	if e != nil {
		return 0, e
	}
	var v int
	e = tx.QueryRow(ctx, `SELECT version FROM evaluacion.aplicacion_escala WHERE id=$1 AND expediente_id=$2 UNION ALL SELECT 1 FROM evaluacion.instrumento_aplicado WHERE id=$1 AND expediente_id=$2`, id, exp).Scan(&v)
	if errors.Is(e, pgx.ErrNoRows) {
		return 0, nil
	}
	return v, traducir(e)
}

//etf:tecnico
func (*Repositorio) Guardar(ctx context.Context, c d.Cambio) error {
	tx, e := postgres.DesdeContexto(ctx)
	if e != nil {
		return e
	}
	if c.Instrumento != nil {
		ins := c.Instrumento
		b, e := json.Marshal(ins)
		if e != nil {
			return d.ErrPeticion
		}
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.instrumento_aplicado(id,expediente_id,equipo_id,datos) VALUES($1,$2,$3,$4)`, ins.ID, ins.ExpedienteID, ins.EquipoID, b)
		return traducir(e)
	}
	ap := *c.Aplicacion
	respuestas := ap.Puntuaciones
	ap.Puntuaciones = nil
	b, e := json.Marshal(ap)
	if e != nil {
		return d.ErrPeticion
	}
	if c.VersionEsperada == 0 {
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.aplicacion_escala(id,expediente_id,equipo_id,adulto_id,instrumento,momento,sujeto,datos,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, ap.ID, ap.ExpedienteID, ap.EquipoID, nulo(ap.Sujeto.AdultoID), ap.Instrumento, ap.Momento, ap.Sujeto, b, ap.Version)
	} else {
		var id string
		e = tx.QueryRow(ctx, `UPDATE evaluacion.aplicacion_escala SET datos=$4,version=version+1 WHERE id=$1 AND expediente_id=$2 AND version=$3 AND instrumento=$5 AND momento=$6 AND sujeto=$7 RETURNING id::text`, ap.ID, ap.ExpedienteID, c.VersionEsperada, b, ap.Instrumento, ap.Momento, ap.Sujeto).Scan(&id)
		if errors.Is(e, pgx.ErrNoRows) {
			return d.ErrPeticion
		}
	}
	if e != nil {
		return traducir(e)
	}
	_, e = tx.Exec(ctx, `DELETE FROM evaluacion.respuesta WHERE aplicacion_id=$1 AND expediente_id=$2`, ap.ID, ap.ExpedienteID)
	if e != nil {
		return traducir(e)
	}
	for _, p := range respuestas {
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.respuesta(aplicacion_id,expediente_id,equipo_id,item,valor_numerico,valor_codigo) VALUES($1,$2,$3,$4,$5,$6)`, ap.ID, ap.ExpedienteID, ap.EquipoID, p.Item, p.ValorNumerico, p.ValorCodigo)
		if e != nil {
			return traducir(e)
		}
	}
	return nil
}
func nulo(v string) any {
	if v == "" {
		return nil
	}
	return v
}

//etf:tecnico
func (*Repositorio) Repetir(ctx context.Context, usuario, id, clave, huella string) (c d.Cambio, existe bool, err error) {
	tx, e := postgres.DesdeContexto(ctx)
	if e != nil {
		return c, false, e
	}
	var h string
	var b []byte
	e = tx.QueryRow(ctx, `SELECT huella,resultado FROM evaluacion.peticion WHERE usuario_id=$1 AND expediente_id=$2 AND clave=$3`, usuario, id, clave).Scan(&h, &b)
	if errors.Is(e, pgx.ErrNoRows) {
		return c, false, nil
	}
	if e != nil {
		return c, false, traducir(e)
	}
	if h != huella {
		return c, false, d.ErrIdempotencia
	}
	if json.Unmarshal(b, &c) != nil {
		return c, false, d.ErrNoDisponible
	}
	return c, true, nil
}

//etf:tecnico
func (*Repositorio) Recordar(ctx context.Context, usuario, id, clave, huella string, c d.Cambio) error {
	tx, e := postgres.DesdeContexto(ctx)
	if e != nil {
		return e
	}
	b, e := json.Marshal(c)
	if e != nil {
		return d.ErrPeticion
	}
	_, e = tx.Exec(ctx, `INSERT INTO evaluacion.peticion(usuario_id,expediente_id,equipo_id,clave,huella,resultado) SELECT $1::uuid,$2::uuid,equipo_id,$3::text,$4::text,$5::jsonb FROM evaluacion.instrumento_aplicado WHERE id=$6 UNION ALL SELECT $1::uuid,$2::uuid,equipo_id,$3::text,$4::text,$5::jsonb FROM evaluacion.aplicacion_escala WHERE id=$6`, usuario, id, clave, huella, b, objetivo(c))
	return traducir(e)
}
func objetivo(c d.Cambio) string {
	if c.Aplicacion != nil {
		return c.Aplicacion.ID
	}
	return c.Instrumento.ID
}
