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

var _ salida.Familias = (*Repositorio)(nil)

func (r *Repositorio) BloquearFamilia(ctx context.Context, id string) (dominio.FichaFamilia, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.FichaFamilia{}, err
	}
	var version int
	if err = e.QueryRow(ctx, `SELECT version FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&version); err != nil { //etf:tecnico
		if errors.Is(err, pgx.ErrNoRows) {
			return dominio.FichaFamilia{}, dominio.ErrNoEncontrada
		}
		return dominio.FichaFamilia{}, err
	}
	return r.ObtenerFamilia(ctx, id)
}
func (*Repositorio) ObtenerFamilia(ctx context.Context, id string) (dominio.FichaFamilia, error) {
	var r dominio.FichaFamilia
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var b []byte
	var equipo *string
	var fecha *time.Time
	err = e.QueryRow(ctx, `SELECT id::text,equipo_id::text,estado,version,fecha_aceptacion,familia FROM expediente.expediente WHERE id=$1`, id).Scan(&r.ID, &equipo, &r.Estado, &r.Version, &fecha, &b) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, dominio.ErrNoEncontrada
	}
	if err != nil {
		return r, err
	}
	if equipo == nil || fecha == nil {
		return r, dominio.ErrEstado
	}
	r.EquipoID = *equipo
	r.Aceptacion = *fecha
	if err = json.Unmarshal(b, &r.Familia); err != nil {
		return r, err
	}
	// Los datos estadísticos actuales salen de la última vigencia, incluso
	// después de una corrección retroactiva. El contacto nunca va a la historia.
	var estadistica []byte
	err = e.QueryRow(ctx, `SELECT valor FROM expediente.consulta_familia_vigencia WHERE expediente_id=$1 AND vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<vigente_hasta)`, id).Scan(&estadistica) //etf:tecnico
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if err == nil {
		if err = json.Unmarshal(estadistica, &r.Familia); err != nil {
			return r, err
		}
	}
	r.Adultos = []dominio.Adulto{}
	rows, err := e.Query(ctx, `SELECT a.datos,to_jsonb(v) FROM expediente.adulto a JOIN expediente.consulta_adulto v ON v.id=a.id
 WHERE v.expediente_id=$1 AND v.vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (v.vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<v.vigente_hasta) ORDER BY v.orden`, id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var datos, estadistica []byte
		var a dominio.Adulto
		if err := rows.Scan(&datos, &estadistica); err != nil {
			return r, err
		}
		if err := json.Unmarshal(datos, &a); err != nil {
			return r, err
		}
		if err := json.Unmarshal(estadistica, &a); err != nil {
			return r, err
		}
		r.Adultos = append(r.Adultos, a)
	}
	return r, rows.Err()
}
func (*Repositorio) RepetirFamilia(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoFamilia, bool, error) {
	return repeticion[dominio.ResultadoFamilia](ctx, usuario, clave, ruta, huella, ahora)
}
func (*Repositorio) RecordarFamilia(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoFamilia, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}
func (*Repositorio) GuardarFamilia(ctx context.Context, id string, p dominio.CambioFamilia, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var valor any
	if p.Familia != nil {
		valor = p.Familia
	} // nil conserva el contacto cuando se guardan adultos.
	b, err := json.Marshal(valor)
	if err != nil {
		return err
	}
	var equipo string
	err = e.QueryRow(ctx, `UPDATE expediente.expediente SET familia=CASE WHEN $2::jsonb='null'::jsonb THEN familia ELSE $2::jsonb END,
 version=version+1,modificado_por=$3,modificado_en=$4 WHERE id=$1 AND version=$5 RETURNING equipo_id::text`, id, b, usuario, ahora, p.Version).Scan(&equipo)
	if err != nil {
		return err
	}
	if p.Familia != nil {
		estadistica, err := json.Marshal(p.Familia.Estadistica())
		if err != nil {
			return err
		}
		_, err = e.Exec(ctx, `INSERT INTO expediente.familia_revision(expediente_id,equipo_id,version,vigente_desde,valor,registrado_por,registrado_en) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, equipo, p.Version+1, p.Desde, estadistica, usuario, ahora) //etf:tecnico
		return err
	}
	// Los identificadores nuevos se generan en aplicación. Nunca se permite
	// trasladar un adulto existente a otro expediente, ni siquiera del mismo equipo.
	adultos, err := json.Marshal(*p.Adultos)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.adulto(id,expediente_id,equipo_id,papel,sexo,datos)
 SELECT (j->>'id')::uuid,$1::uuid,$2::uuid,j->>'papel',j->>'sexo',j FROM jsonb_array_elements($3::jsonb) j
 ON CONFLICT(id) DO UPDATE SET papel=EXCLUDED.papel,sexo=EXCLUDED.sexo,datos=EXCLUDED.datos
 WHERE adulto.expediente_id=EXCLUDED.expediente_id AND adulto.equipo_id=EXCLUDED.equipo_id`, id, equipo, adultos)
	if err != nil {
		return err
	}
	var revision string
	err = e.QueryRow(ctx, `INSERT INTO expediente.adultos_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, id, equipo, p.Version+1, p.Desde, usuario, ahora).Scan(&revision) //etf:tecnico
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.adulto_revision(revision_id,adulto_id,expediente_id,equipo_id,orden,papel,sexo,fecha_nacimiento,edad_declarada,nivel_educativo,situacion_laboral,es_tutor)
 SELECT $1::uuid,a.id,$2::uuid,$3::uuid,a.orden,a.papel,a.sexo,a.fecha_nacimiento,a.edad_declarada,a.nivel_educativo,a.situacion_laboral,a.es_tutor
 FROM jsonb_to_recordset($4::jsonb) AS a(id uuid,orden integer,papel text,sexo text,fecha_nacimiento date,edad_declarada integer,nivel_educativo text,situacion_laboral text,es_tutor boolean)`, revision, id, equipo, adultos)
	if err != nil {
		return err
	}
	// Las personas retiradas mantienen ID y datos estadísticos para evaluación,
	// pero dejan de conservar nombres, contactos y narrativas sin uso vigente.
	_, err = e.Exec(ctx, `UPDATE expediente.adulto a SET datos=datos-ARRAY['nombre','apellidos','telefono','email','otros_datos','situacion_laboral_texto']
 WHERE a.expediente_id=$1 AND NOT EXISTS(SELECT 1 FROM expediente.consulta_adulto v WHERE v.id=a.id AND v.expediente_id=a.expediente_id
 AND v.vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (v.vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<v.vigente_hasta))`, id)

	return err
}
func (*Repositorio) HistoriaFamilia(ctx context.Context, id, corte string) (dominio.HistoriaFamilia, error) {
	r := dominio.HistoriaFamilia{Adultos: []dominio.AdultoEstadistico{}}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var b []byte
	var desde time.Time
	var hasta *time.Time
	var version int
	err = e.QueryRow(ctx, `SELECT valor,vigente_desde,vigente_hasta,version FROM expediente.consulta_familia_vigencia
 WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta)`, id, corte).Scan(&b, &desde, &hasta, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if err == nil {
		if err = json.Unmarshal(b, &r.Familia); err != nil {
			return r, err
		}
		r.VigenciaFamilia = vigencia(desde, hasta, version)
	}
	err = e.QueryRow(ctx, `SELECT vigente_desde,vigente_hasta,version FROM expediente.consulta_adultos_vigencia
 WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta)`, id, corte).Scan(&desde, &hasta, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if err == nil {
		r.VigenciaAdultos = vigencia(desde, hasta, version)
	}
	rows, err := e.Query(ctx, `SELECT to_jsonb(v) FROM expediente.consulta_adulto v WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta) ORDER BY orden`, id, corte) //etf:tecnico
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var a dominio.AdultoEstadistico
		if err = rows.Scan(&b); err != nil {
			return r, err
		}
		if err = json.Unmarshal(b, &a); err != nil {
			return r, err
		}
		r.Adultos = append(r.Adultos, a)
	}
	return r, rows.Err()
}
func vigencia(desde time.Time, hasta *time.Time, version int) *dominio.Vigencia {
	r := &dominio.Vigencia{Desde: desde.Format("2006-01-02"), Revision: version}
	if hasta != nil {
		f := hasta.Format("2006-01-02")
		r.Hasta = &f
	}
	return r
}

func (*Repositorio) AdultosPertenecen(ctx context.Context, id string, ids []string) (bool, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return false, err
	}
	var n int
	err = e.QueryRow(ctx, `SELECT count(*) FROM expediente.adulto WHERE expediente_id=$1 AND id=ANY($2::uuid[])`, id, ids).Scan(&n) //etf:tecnico
	return n == len(ids), err
}
