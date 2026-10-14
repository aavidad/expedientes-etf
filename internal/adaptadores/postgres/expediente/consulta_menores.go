package expediente

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"time"
)

type ConsultaMenores struct{}

var _ salida.ConsultaMenores = ConsultaMenores{}

func (ConsultaMenores) MenoresVigentes(ctx context.Context, id, corte string) ([]dominio.MenorConsulta, error) {
	if !dominio.IDValido(id) {
		return nil, dominio.ErrPeticion
	}
	if fecha, err := time.Parse("2006-01-02", corte); err != nil || fecha.Year() < 1 {
		return nil, dominio.ErrMenores
	}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := e.Query(ctx, `SELECT to_jsonb(v),vigente_desde,vigente_hasta,version FROM expediente.consulta_menor v WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta) ORDER BY orden`, id, corte) //etf:tecnico
	if err != nil {
		return nil, err
	}
	resultado := []dominio.MenorConsulta{}
	for rows.Next() {
		var m dominio.MenorConsulta
		var b []byte
		var desde time.Time
		var hasta *time.Time
		var version int
		if err = rows.Scan(&b, &desde, &hasta, &version); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(b, &m); err != nil {
			rows.Close()
			return nil, err
		}
		m.Vigencia = *vigencia(desde, hasta, version)
		resultado = append(resultado, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	_, err = e.Exec(ctx, `SELECT auditoria.registrar(transaction_timestamp(),$1,$2,$3::uuid,$3::uuid,$4)`, "expediente.leer_menores", "expediente", id, "ok") //etf:tecnico
	if err != nil {
		return nil, err
	}
	return resultado, nil
}
