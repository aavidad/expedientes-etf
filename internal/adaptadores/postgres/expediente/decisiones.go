package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"time"
)

var _ salida.Decisiones = (*Repositorio)(nil)

func (*Repositorio) RepeticionDecision(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoDecision, bool, error) {
	return repeticion[dominio.ResultadoDecision](ctx, usuario, clave, ruta, huella, ahora)
}

func (*Repositorio) RecordarDecision(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoDecision, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}

func (*Repositorio) Decidir(ctx context.Context, p dominio.NoAceptacion, r dominio.ResultadoDecision, anterior, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	fecha, err := time.Parse("2006-01-02", r.Fecha)
	if err != nil {
		return dominio.ErrPeticion
	}
	var tagFilas int64
	if r.Estado == dominio.ListaEspera {
		tag, err := e.Exec(ctx, `UPDATE expediente.expediente SET estado='lista_espera',fecha_lista_espera=$2,
 version=version+1,modificado_por=$3,modificado_en=$4 WHERE id=$1 AND version=$5 AND estado='derivacion_recibida'`, r.ID, fecha, usuario, ahora, p.Version)
		if err != nil {
			return err
		}
		tagFilas = tag.RowsAffected()
	} else {
		tag, err := e.Exec(ctx, `UPDATE expediente.expediente SET estado='no_aceptada',fecha_no_aceptacion=$2,
 causa_no_aceptacion=$3,subprograma=$4,version=version+1,modificado_por=$5,modificado_en=$6
 WHERE id=$1 AND version=$7 AND estado IN ('derivacion_recibida','lista_espera')`, r.ID, fecha, p.Motivo, r.Subprograma, usuario, ahora, p.Version)
		if err != nil {
			return err
		}
		tagFilas = tag.RowsAffected()
	}
	if tagFilas != 1 {
		return dominio.ErrEstado
	}
	tipo := "no_aceptacion"
	if r.Estado == dominio.ListaEspera {
		tipo = "lista_espera"
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.hito(expediente_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 VALUES($1,$2,$3,'documento_externo',$4,$5)`, r.ID, tipo, fecha, usuario, ahora); err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.estado_cambio(expediente_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
 VALUES($1,$2,$3,$4,$5)`, r.ID, anterior, r.Estado, usuario, ahora)
	return err
}

func (*Repositorio) ListarEspera(ctx context.Context, p salida.FiltroEspera) ([]salida.FilaEspera, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	filas, err := e.Query(ctx, `SELECT id::text,estado,version,fecha_recepcion,fecha_lista_espera
 FROM expediente.expediente WHERE estado='lista_espera' ORDER BY fecha_recepcion,id LIMIT $1 OFFSET $2`, p.Limite, p.Desplazamiento)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	resultado := []salida.FilaEspera{}
	for filas.Next() {
		var f salida.FilaEspera
		var fecha time.Time
		if err := filas.Scan(&f.ID, &f.Estado, &f.Version, &f.FechaRecepcion, &fecha); err != nil {
			return nil, err
		}
		f.FechaRecepcion = f.FechaRecepcion.UTC()
		f.FechaEspera = fecha.Format("2006-01-02")
		resultado = append(resultado, f)
	}
	return resultado, filas.Err()
}
