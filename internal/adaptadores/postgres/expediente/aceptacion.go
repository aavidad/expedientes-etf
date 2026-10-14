package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ salida.Aceptaciones = (*Repositorio)(nil)

func (*Repositorio) Repeticion(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoAceptacion, bool, error) {
	return repeticion[dominio.ResultadoAceptacion](ctx, usuario, clave, ruta, huella, ahora)
}
func repeticion[T any](ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (T, bool, error) {
	var vacio T
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return vacio, false, err
	}
	// Bloqueo de transacción también cuando aún no existe la fila de respuesta.
	if _, err := e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,697051))`, usuario+":"+clave); err != nil { //etf:tecnico
		return vacio, false, err
	}
	if _, err := e.Exec(ctx, `DELETE FROM expediente.peticion_idempotente WHERE usuario_id=$1 AND clave=$2 AND creada_en<=$3`, usuario, clave, ahora.Add(-24*time.Hour)); err != nil { //etf:tecnico
		return vacio, false, err
	}
	var anteriorRuta, anteriorHuella string
	var respuesta []byte
	err = e.QueryRow(ctx, `SELECT ruta,huella_peticion,respuesta FROM expediente.peticion_idempotente WHERE usuario_id=$1 AND clave=$2`, usuario, clave).Scan(&anteriorRuta, &anteriorHuella, &respuesta) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return vacio, false, nil
	}
	if err != nil {
		return vacio, false, err
	}
	if anteriorRuta != ruta || anteriorHuella != huella {
		return vacio, false, dominio.ErrIdempotencia
	}
	var r T
	if err := json.Unmarshal(respuesta, &r); err != nil {
		return r, false, err
	}
	return r, true, nil
}
func (r *Repositorio) Bloquear(ctx context.Context, id string) (dominio.Derivacion, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Derivacion{}, err
	}
	var version int
	if err := e.QueryRow(ctx, `SELECT version FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&version); err != nil { //etf:tecnico
		if errors.Is(err, pgx.ErrNoRows) {
			return dominio.Derivacion{}, dominio.ErrNoEncontrada
		}
		return dominio.Derivacion{}, err
	}
	return r.Obtener(ctx, id)
}
func (*Repositorio) Siguiente(ctx context.Context, equipo string, anio, limite int) (int, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return 0, err
	}
	for {
		var n int
		err = e.QueryRow(ctx, `INSERT INTO expediente.contador_expediente(equipo_id,anio,ultimo) VALUES($1,$2,1)
 ON CONFLICT(equipo_id,anio) DO UPDATE SET ultimo=expediente.contador_expediente.ultimo+1
 WHERE expediente.contador_expediente.ultimo<$3 RETURNING ultimo`, equipo, anio, limite).Scan(&n)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, dominio.ErrNumeracion
		}
		if err != nil {
			return 0, err
		}
		var ocupado bool
		if err = e.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expediente.expediente WHERE equipo_id=$1 AND anio_numero=$2 AND correlativo=$3)`, equipo, anio, n).Scan(&ocupado); err != nil { //etf:tecnico
			return 0, err
		}
		if !ocupado {
			return n, nil
		}
	}
}
func (*Repositorio) Aceptar(ctx context.Context, p dominio.Aceptacion, r dominio.ResultadoAceptacion, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	fecha, _ := p.Fecha()
	var anterior string
	if err := e.QueryRow(ctx, `SELECT estado FROM expediente.expediente WHERE id=$1`, r.ID).Scan(&anterior); err != nil { //etf:tecnico
		return err
	}
	tag, err := e.Exec(ctx, `UPDATE expediente.expediente SET numero=nullif($2,''),equipo_id=$3,municipio_id=$4,centro_ss_id=$5,
 anio_numero=CASE WHEN $2='' THEN NULL ELSE $6::integer END,correlativo=nullif($7::integer,0),regla_numeracion_id=CASE WHEN $2='' THEN NULL ELSE nullif($8,'')::uuid END,fecha_aceptacion=$9,subprograma=$10,
 profesional_referencia_puesto=$11,profesional_referencia_id=$12,descripcion_situacion=$13,
 fase_intervencion='recepcion',estado='aceptada',version=version+1,modificado_por=$14,modificado_en=$15
 WHERE id=$1 AND version=$16 AND estado IN ('derivacion_recibida','lista_espera')`, r.ID, r.Numero, p.EquipoID, p.MunicipioID, p.CentroID, r.Anio, r.Correlativo, r.ReglaID, fecha, p.Subprograma, p.ProfesionalPuesto, p.ProfesionalID, p.Descripcion, usuario, ahora, p.Version)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return dominio.ErrNumeroDuplicado
		}
		return err
	}
	if tag.RowsAffected() != 1 {
		return dominio.ErrEstado
	}
	var episodio string
	err = e.QueryRow(ctx, `INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,creado_por,creado_en)
 VALUES($1,$2,$3,$4,$5) RETURNING id::text`, r.ID, r.EquipoID, fecha, usuario, ahora).Scan(&episodio)
	if err != nil {
		return err
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.hito(expediente_id,equipo_id,episodio_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 VALUES($1,$2,$3,'aceptacion',$4,'documento_externo',$5,$6)`, r.ID, r.EquipoID, episodio, fecha, usuario, ahora); err != nil {
		return err
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.estado_cambio(expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
 VALUES($1,$2,$5,'aceptada',$3,$4)`, r.ID, r.EquipoID, usuario, ahora, anterior); err != nil {
		return err
	}
	// Se inserta la colección completa y su historia mediante dos consultas fijas.
	if err := insertarMenoresAceptacion(ctx, r, p, usuario, ahora); err != nil {
		return err
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.dato_estadistico_vigencia(expediente_id,equipo_id,entidad_tipo,entidad_id,menor_id,atributo,valor,vigente_desde,registrado_en,registrado_por)
 SELECT $1::uuid,$2::uuid,'menor',id,id,'sexo',to_jsonb(sexo),$3::date,$4::timestamptz,$5::uuid FROM expediente.menor WHERE expediente_id=$1
 UNION ALL SELECT $1::uuid,$2::uuid,'expediente',$1::uuid,NULL::uuid,'subprograma',to_jsonb($6::text),$3,$4,$5`, r.ID, r.EquipoID, fecha, ahora, usuario, p.Subprograma); err != nil {
		return err
	}
	return materializarCopiaTraslado(ctx, r.ID, r.EquipoID, p.FechaActa, usuario, r.Version, ahora)
}
func (*Repositorio) Recordar(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoAceptacion, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}
func recordar(ctx context.Context, usuario, clave, ruta, huella string, r any, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.peticion_idempotente(usuario_id,clave,ruta,huella_peticion,respuesta,creada_en) VALUES($1,$2,$3,$4,$5,$6)`, usuario, clave, ruta, huella, b, ahora) //etf:tecnico
	return err
}

func cargarAceptacion(ctx context.Context, d *dominio.Derivacion) error {
	if d.Estado != dominio.Aceptada && d.Estado != dominio.EnIntervencion && d.Estado != dominio.Cerrada && d.Estado != dominio.DerivadaSPM {
		return nil
	}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var a dominio.DatosAceptados
	var fecha time.Time
	err = e.QueryRow(ctx, `SELECT centro_ss_id::text,fecha_aceptacion,subprograma,profesional_referencia_puesto,
 profesional_referencia_id::text,descripcion_situacion,coalesce(anio_numero,0),coalesce(correlativo,0),coalesce(regla_numeracion_id::text,''),fase_intervencion
 FROM expediente.expediente WHERE id=$1`, d.ID).Scan(&a.CentroID, &fecha, &a.Subprograma, &a.ProfesionalPuesto, &a.ProfesionalID, &a.Descripcion, &a.Anio, &a.Correlativo, &a.ReglaID, &a.Fase)
	if err != nil {
		return err
	}
	a.Version = d.Version
	a.MunicipioID = *d.MunicipioID
	a.EquipoID = *d.EquipoID
	a.FechaActa = fecha.Format("2006-01-02")
	a.Menores = []dominio.SexoMenor{}
	// El ensayo de actualización consulta también aceptaciones anteriores a F2.
	var historiaMenores bool
	if err := e.QueryRow(ctx, `SELECT to_regclass('expediente.consulta_menor') IS NOT NULL`).Scan(&historiaMenores); err != nil { //etf:tecnico
		return err
	} //etf:tecnico
	consulta := `SELECT orden,sexo FROM expediente.menor WHERE expediente_id=$1 ORDER BY orden` //etf:tecnico
	if historiaMenores {
		consulta = `SELECT orden,sexo FROM expediente.consulta_menor WHERE expediente_id=$1 AND vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<vigente_hasta) ORDER BY orden`
	} //etf:tecnico
	filas, err := e.Query(ctx, consulta, d.ID)
	if err != nil {
		return err
	}
	defer filas.Close()
	for filas.Next() {
		var m dominio.SexoMenor
		if err := filas.Scan(&m.Orden, &m.Sexo); err != nil {
			return err
		}
		a.Menores = append(a.Menores, m)
	}
	if err := filas.Err(); err != nil {
		return err
	}
	d.Aceptacion = &a
	return nil
}
