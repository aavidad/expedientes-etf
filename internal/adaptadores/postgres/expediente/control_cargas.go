package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func errorControlCarga(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Message {
		case "carga.plazo_cerrado":
			return d.ErrCargaCerrada
		case "carga.lote_con_actividad":
			return d.ErrLoteConActividad
		case "expediente.acceso_denegado":
			return d.ErrAcceso
		case "expediente.no_encontrado":
			return d.ErrNoEncontrada
		case "carga.control_invalido":
			return d.ErrControlCarga
		}
	}
	return err
}
func (*Repositorio) PlazoCarga(ctx context.Context, bloquear bool) (r d.PlazoCarga, err error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	q := `SELECT coalesce(fecha_cierre::text,''),version,dias_duplicados FROM expediente.plazo_carga WHERE id` //etf:tecnico
	if bloquear {
		q = `SELECT coalesce(fecha_cierre::text,''),version,dias_duplicados FROM expediente.bloquear_plazo_carga()` //etf:tecnico
	} //etf:tecnico
	err = tx.QueryRow(ctx, q).Scan(&r.Fecha, &r.Version, &r.DiasDuplicados)
	return
}
func (*Repositorio) GuardarPlazoCarga(ctx context.Context, p d.PlazoCarga, t time.Time) (r d.PlazoCarga, err error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	err = tx.QueryRow(ctx, `UPDATE expediente.plazo_carga SET fecha_cierre=$1,version=version+1,dias_duplicados=coalesce($3,dias_duplicados) WHERE id AND version=$2 RETURNING fecha_cierre::text,version,dias_duplicados`, p.Fecha, p.Version, p.DiasDuplicados).Scan(&r.Fecha, &r.Version, &r.DiasDuplicados) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		actual, e := (&Repositorio{}).PlazoCarga(ctx, false)
		if e != nil {
			return r, e
		}
		return r, d.Conflicto{VersionActual: actual.Version}
	}
	return
}
func (*Repositorio) LotesCarga(ctx context.Context, desplazamiento int) ([]d.LoteCarga, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,equipo_id::text,cargada_en,(recuentos->>'casos')::integer,estado='anulado',referencia FROM expediente.carga_inicial ORDER BY cargada_en DESC,id DESC LIMIT 100 OFFSET $1`, desplazamiento) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := []d.LoteCarga{}
	for rows.Next() {
		var v d.LoteCarga
		if err = rows.Scan(&v.ID, &v.EquipoID, &v.Fecha, &v.Casos, &v.Anulado, &v.Referencia); err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, rows.Err()
}
func (*Repositorio) AnularLoteCarga(ctx context.Context, id string, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT expediente.anular_lote_carga($1,$2)`, id, t) //etf:tecnico
	return errorControlCarga(err)
}
func (*Repositorio) ComprobarNumeracionCarga(ctx context.Context, p d.CargaHistorica, anio, limite int) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	// El mismo bloqueo se toma al asignar numeración ordinaria. Dos cargas
	// simultáneas no pueden consumir ambas el último hueco disponible.
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,697052))`, p.EquipoID+":"+strconv.Itoa(anio)) //etf:tecnico
	if err != nil {
		return err
	}
	numeros := []int{}
	for _, c := range p.Casos {
		if c.Aceptacion != nil && strings.HasPrefix(c.Aceptacion.FechaActa, strconv.Itoa(anio)+"-") && c.Numero != "" {
			partes := strings.Split(c.Numero, "-")
			if len(partes) == 3 {
				n, _ := strconv.Atoi(partes[1])
				numeros = append(numeros, n)
			}
		}
	}
	if len(numeros) == 0 {
		return nil
	}
	var libres int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM generate_series(coalesce((SELECT ultimo FROM expediente.contador_expediente WHERE equipo_id=$1 AND anio=$2),0)+1,$3) n
 WHERE n<>ALL($4::integer[]) AND NOT EXISTS(SELECT 1 FROM expediente.expediente WHERE equipo_id=$1 AND anio_numero=$2 AND correlativo=n)`, p.EquipoID, anio, limite, numeros).Scan(&libres) //etf:tecnico
	if err != nil {
		return err
	}
	if libres == 0 {
		return d.ErrCargaAgotaNumeracion
	}
	return nil
}
func (*Repositorio) CandidatosCarga(ctx context.Context, p d.CargaHistorica) ([]d.CasoHistorico, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	primera, ultima := p.Casos[0].FechaDerivacion, p.Casos[0].FechaDerivacion
	numeros := []string{}
	for _, c := range p.Casos {
		if c.FechaDerivacion < primera {
			primera = c.FechaDerivacion
		}
		if c.FechaDerivacion > ultima {
			ultima = c.FechaDerivacion
		}
		if c.Numero != "" {
			numeros = append(numeros, c.Numero)
		}
	}
	rows, err := tx.Query(ctx, `SELECT coalesce(e.numero,''),(e.fecha_recepcion AT TIME ZONE 'UTC')::date::text,d.contenido,
 coalesce((SELECT jsonb_agg(a.datos) FROM expediente.adulto a WHERE a.expediente_id=e.id),'[]'),
 coalesce((SELECT jsonb_agg(m.datos) FROM expediente.menor m WHERE m.expediente_id=e.id),'[]')
 FROM expediente.expediente e JOIN expediente.derivacion_datos d ON d.expediente_id=e.id
 WHERE coalesce(e.equipo_id,e.equipo_carga_id)=$1 AND (e.numero=ANY($4::text[]) OR
 (e.fecha_recepcion AT TIME ZONE 'UTC')::date BETWEEN $2::date-(SELECT dias_duplicados FROM expediente.plazo_carga WHERE id) AND $3::date+(SELECT dias_duplicados FROM expediente.plazo_carga WHERE id))`, p.EquipoID, primera, ultima, numeros) //etf:tecnico
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := []d.CasoHistorico{}
	for rows.Next() {
		var c d.CasoHistorico
		var datos, adultos, menores []byte
		if err = rows.Scan(&c.Numero, &c.FechaDerivacion, &datos, &adultos, &menores); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(datos, &c.Datos); err != nil {
			return nil, err
		}
		var aa []d.Adulto
		var mm []d.Menor
		if err = json.Unmarshal(adultos, &aa); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(menores, &mm); err != nil {
			return nil, err
		}
		c.Familia = []d.CambioFamilia{{Adultos: &aa}}
		c.Menores = []d.CambioMenores{{Menores: &mm}}
		r = append(r, c)
	}
	return r, rows.Err()
}

func (*Repositorio) ReferenciaLoteCarga(ctx context.Context, id string) (int64, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return 0, err
	}
	var referencia int64
	err = tx.QueryRow(ctx, `SELECT referencia FROM expediente.carga_inicial WHERE id=$1`, id).Scan(&referencia) //etf:tecnico
	return referencia, err
}
