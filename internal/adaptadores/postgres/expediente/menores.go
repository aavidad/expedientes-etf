package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

var _ salida.Menores = (*Repositorio)(nil)

func (r *Repositorio) BloquearMenores(ctx context.Context, id string) (dominio.FichaMenores, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.FichaMenores{}, err
	}
	var version int
	if err = e.QueryRow(ctx, `SELECT version FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&version); err != nil { //etf:tecnico
		if errors.Is(err, pgx.ErrNoRows) {
			err = dominio.ErrNoEncontrada
		}
		return dominio.FichaMenores{}, err
	}
	return r.ObtenerMenores(ctx, id)
}
func (*Repositorio) ObtenerMenores(ctx context.Context, id string) (dominio.FichaMenores, error) {
	r := dominio.FichaMenores{Menores: []dominio.Menor{}}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var equipo *string
	var fecha *time.Time
	var subprograma *string
	err = e.QueryRow(ctx, `SELECT id::text,equipo_id::text,estado,version,fecha_aceptacion,subprograma FROM expediente.expediente WHERE id=$1`, id).Scan(&r.ID, &equipo, &r.Estado, &r.Version, &fecha, &subprograma) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, dominio.ErrNoEncontrada
	}
	if err != nil {
		return r, err
	}
	if equipo == nil || fecha == nil || subprograma == nil {
		return r, dominio.ErrEstado
	}
	r.EquipoID, r.Aceptacion, r.Subprograma = *equipo, *fecha, *subprograma
	rows, err := e.Query(ctx, `SELECT m.datos,to_jsonb(v) FROM expediente.menor m JOIN expediente.consulta_menor v ON (v.id,v.expediente_id,v.equipo_id)=(m.id,m.expediente_id,m.equipo_id)
 WHERE v.expediente_id=$1 AND v.vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (v.vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<v.vigente_hasta) ORDER BY v.orden`, id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var datos, estadistica []byte
		var m dominio.Menor
		if err = rows.Scan(&datos, &estadistica); err != nil {
			return r, err
		}
		if err = json.Unmarshal(datos, &m); err != nil {
			return r, err
		}
		if err = json.Unmarshal(estadistica, &m); err != nil {
			return r, err
		}
		m.DatosDesdeAnexo = string(datos) == "{}"
		r.Menores = append(r.Menores, m)
	}
	return r, rows.Err()
}
func (*Repositorio) MenoresPertenecen(ctx context.Context, id string, ids []string) (bool, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return false, err
	}
	var n int
	err = e.QueryRow(ctx, `SELECT count(*) FROM expediente.menor WHERE expediente_id=$1 AND id=ANY($2::uuid[])`, id, ids).Scan(&n) //etf:tecnico
	return n == len(ids), err
}
func (*Repositorio) RepetirMenores(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoMenores, bool, error) {
	return repeticion[dominio.ResultadoMenores](ctx, usuario, clave, ruta, huella, ahora)
}
func (*Repositorio) RecordarMenores(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoMenores, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}
func (*Repositorio) GuardarMenores(ctx context.Context, id string, p dominio.CambioMenores, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var equipo string
	err = e.QueryRow(ctx, `UPDATE expediente.expediente SET version=version+1,modificado_por=$2,modificado_en=$3 WHERE id=$1 AND version=$4 RETURNING equipo_id::text`, id, usuario, ahora, p.Version).Scan(&equipo) //etf:tecnico
	if err != nil {
		return err
	}
	datos, err := json.Marshal(*p.Menores)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.menor(id,expediente_id,equipo_id,orden,sexo,datos)
 SELECT (j->>'id')::uuid,$1::uuid,$2::uuid,(j->>'orden')::integer,j->>'sexo',j FROM jsonb_array_elements($3::jsonb) j
 ON CONFLICT(id) DO UPDATE SET datos=EXCLUDED.datos WHERE menor.expediente_id=EXCLUDED.expediente_id AND menor.equipo_id=EXCLUDED.equipo_id`, id, equipo, datos)
	if err != nil {
		return err
	}
	var revision string
	err = e.QueryRow(ctx, `INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, id, equipo, p.Version+1, p.Desde, usuario, ahora).Scan(&revision) //etf:tecnico
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.menor_revision(revision_id,menor_id,expediente_id,equipo_id,orden,sexo,fecha_nacimiento,nivel_escolarizacion,subprograma,centro_proteccion_codigo,propuesto_atencion_temprana)
 SELECT $1::uuid,m.id,$2::uuid,$3::uuid,m.orden,m.sexo,m.fecha_nacimiento,m.nivel_escolarizacion,m.subprograma,m.centro_proteccion_codigo,m.propuesto_atencion_temprana
 FROM jsonb_to_recordset($4::jsonb) AS m(id uuid,orden integer,sexo text,fecha_nacimiento date,nivel_escolarizacion text,subprograma text,centro_proteccion_codigo text,propuesto_atencion_temprana boolean)`, revision, id, equipo, datos)
	if err != nil {
		return err
	}
	// Retirar conserva la identidad y la estadística; limpia los textos sin uso.
	_, err = e.Exec(ctx, `UPDATE expediente.menor m SET datos=datos-ARRAY['nombre','apellidos','centro_educativo','centro_proteccion_texto']
 WHERE m.expediente_id=$1 AND NOT EXISTS(SELECT 1 FROM expediente.consulta_menor v WHERE (v.id,v.expediente_id,v.equipo_id)=(m.id,m.expediente_id,m.equipo_id)
 AND v.vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (v.vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<v.vigente_hasta))`, id)
	return err
}
func (*Repositorio) HistoriaMenores(ctx context.Context, id, corte string) (dominio.HistoriaMenores, error) {
	r := dominio.HistoriaMenores{Menores: []dominio.MenorEstadistico{}}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var desde time.Time
	var hasta *time.Time
	var version int
	err = e.QueryRow(ctx, `SELECT vigente_desde,vigente_hasta,version FROM expediente.consulta_menores_vigencia WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta)`, id, corte).Scan(&desde, &hasta, &version) //etf:tecnico
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if err == nil {
		r.Vigencia = vigencia(desde, hasta, version)
	}
	rows, err := e.Query(ctx, `SELECT to_jsonb(v) FROM expediente.consulta_menor v WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta) ORDER BY orden`, id, corte) //etf:tecnico
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var m dominio.MenorEstadistico
		if err = rows.Scan(&b); err != nil {
			return r, err
		}
		if err = json.Unmarshal(b, &m); err != nil {
			return r, err
		}
		r.Menores = append(r.Menores, m)
	}
	return r, rows.Err()
}
