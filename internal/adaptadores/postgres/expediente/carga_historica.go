package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

func (*Repositorio) RepetirCargaHistorica(ctx context.Context, u, c, r, h string, t time.Time) (d.ResultadoCargaHistorica, bool, error) {
	return repeticion[d.ResultadoCargaHistorica](ctx, u, c, r, h, t)
}
func (*Repositorio) RecordarCargaHistorica(ctx context.Context, u, c, r, h string, p d.ResultadoCargaHistorica, t time.Time) error {
	return recordar(ctx, u, c, r, h, p, t)
}
func (*Repositorio) CrearLoteHistorico(ctx context.Context, id, equipo, u string, t time.Time, n int) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.carga_inicial(id,equipo_id,cargada_por,cargada_en,recuentos) VALUES($1,$2,$3,$4,jsonb_build_object('casos',$5::integer))`, id, equipo, u, t, n) //etf:tecnico
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return d.ErrIdempotencia
	}
	return err
}
func (*Repositorio) CrearCasoHistorico(ctx context.Context, id, lote, equipo string, c d.CasoHistorico, u string, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	fecha, _ := time.Parse("2006-01-02", c.FechaDerivacion)
	_, err = tx.Exec(ctx, `INSERT INTO expediente.expediente(id,fecha_recepcion,origen,carga_id,equipo_carga_id,cargado_por) VALUES($1,$2,'carga_inicial',$3,$4,$5)`, id, fecha, lote, equipo, u) //etf:tecnico
	if err != nil {
		return err
	}
	b, err := json.Marshal(c.Datos)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido) VALUES($1,'1',$2)`, id, b) //etf:tecnico
	return err
}
func (*Repositorio) EstadoHistorico(ctx context.Context, id, estado, u string, t time.Time) error {
	s, err := (&Repositorio{}).BloquearCierre(ctx, id)
	if err != nil {
		return err
	}
	return cambiarEstadoCierre(ctx, id, s, estado, u, t)
}

func (*Repositorio) CrearLoteHojaHistorico(ctx context.Context, id, equipo, u string, t time.Time, n int, f d.FuenteHoja) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.carga_inicial(id,equipo_id,cargada_por,cargada_en,recuentos,origen,huella_fuente,formato_fuente,bytes_fuente,antivirus)
 VALUES($1,$2,$3,$4,jsonb_build_object('casos',$5::integer),'hoja',decode($6,'hex'),$7,$8,'limpio')`, id, equipo, u, t, n, f.Huella, f.Formato, f.Bytes) //etf:tecnico
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return d.ErrIdempotencia
	}
	return err
}
