package expediente

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

// ConsultaAdultos publica el contrato de expediente a evaluación. Usa la
// transacción del consumidor, conserva RLS y confirma su auditoría en ella.
type ConsultaAdultos struct{}

func (ConsultaAdultos) AdultosVigentes(ctx context.Context, id, corte string) ([]dominio.AdultoConsulta, error) {
	if !dominio.IDValido(id) {
		return nil, dominio.ErrPeticion
	}
	if _, err := time.Parse("2006-01-02", corte); err != nil {
		return nil, dominio.ErrFamilia
	}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := e.Query(ctx, `SELECT id::text,expediente_id::text,papel,sexo,vigente_desde,vigente_hasta,version
 FROM expediente.consulta_adulto WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta) ORDER BY orden`, id, corte)
	if err != nil {
		return nil, err
	}
	resultado := []dominio.AdultoConsulta{}
	for rows.Next() {
		var a dominio.AdultoConsulta
		var desde time.Time
		var hasta *time.Time
		var version int
		if err = rows.Scan(&a.ID, &a.ExpedienteID, &a.Papel, &a.Sexo, &desde, &hasta, &version); err != nil {
			rows.Close()
			return nil, err
		}
		a.Vigencia = *vigencia(desde, hasta, version)
		resultado = append(resultado, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// La función existente encadena el evento en la misma transacción; no se
	// entregan filas si no puede registrarse la lectura.
	_, err = e.Exec(ctx, `SELECT auditoria.registrar(transaction_timestamp(),$1,$2,$3::uuid,$3::uuid,$4)`, "expediente.leer_adultos", "expediente", id, "ok") //etf:tecnico
	if err != nil {
		return nil, err
	}
	return resultado, nil
}
