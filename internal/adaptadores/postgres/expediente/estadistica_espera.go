package expediente

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

//etf:tecnico
func (l *LecturaEstadistica) incorporarEsperasMemoria(ctx context.Context, expedientes []d.Expediente, equipo string, corte, conocido time.Time) ([]d.Expediente, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	consulta := `SELECT expediente_id::text,coalesce(equipo_id::text,''),recepcion,desde,registrado_en
 FROM expediente.espera_memoria WHERE ($1='' OR equipo_id=nullif($1,'')::uuid)
 AND desde<=$2::date AND registrado_en<=$3
 AND (hasta IS NULL OR hasta>$2::date OR terminado_en>$3) ORDER BY expediente_id`
	if l.CalculoInterno {
		consulta = strings.ReplaceAll(consulta, "expediente.espera_memoria", "estadistica.calculo_espera_memoria")
	}
	filas, err := tx.Query(ctx, consulta, equipo, corte, conocido)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	porID := map[string]int{}
	for i, exp := range expedientes {
		porID[exp.ID] = i
	}
	for filas.Next() {
		var exp d.Expediente
		var recepcion, desde, registrado time.Time
		if err = filas.Scan(&exp.ID, &exp.EquipoID, &recepcion, &desde, &registrado); err != nil {
			return nil, err
		}
		vigencia := d.Vigencia{Desde: desde}
		// La espera siempre procede del Anexo 3 (Riesgo). No convierte en datos
		// conocidos el sexo ni la relación de menores aún sin confirmar al aceptar.
		exp.Estados = d.Historia[string]{{ID: exp.ID + ":espera", Vigencia: vigencia, RegistradoEn: registrado, Dato: d.Dato[string]{Conocido: true, Valor: "lista_espera"}}}
		exp.Subprograma = d.Historia[d.Subprograma]{{ID: exp.ID + ":espera-subprograma", Vigencia: vigencia, RegistradoEn: registrado, Dato: d.Dato[d.Subprograma]{Conocido: true, Valor: d.Riesgo}}}
		exp.ServicioDerivante = d.Historia[string]{{ID: exp.ID + ":servicio", Vigencia: d.Vigencia{Desde: recepcion}, RegistradoEn: recepcion, Dato: d.Dato[string]{Conocido: true, Valor: "SSSSCC"}}}
		if i, ok := porID[exp.ID]; ok {
			expedientes[i] = exp
		} else {
			porID[exp.ID] = len(expedientes)
			expedientes = append(expedientes, exp)
		}
	}
	if err = filas.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(expedientes, func(a, b d.Expediente) int { return strings.Compare(a.ID, b.ID) })
	return expedientes, nil
}

//etf:tecnico
func (*LecturaEstadistica) EquiposAfectadosMemoria(ctx context.Context, id string) ([]string, time.Time, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	var equipos []string
	var modificado time.Time
	err = tx.QueryRow(ctx, `SELECT array_remove(ARRAY[e.equipo_id,m.equipo_id],NULL)::text[],
 coalesce(e.modificado_en,e.fecha_recepcion)
 FROM expediente.expediente e LEFT JOIN expediente.espera_memoria m ON m.expediente_id=e.id AND m.desde IS NOT NULL WHERE e.id=$1`, id).Scan(&equipos, &modificado)
	return equipos, modificado, err
}
