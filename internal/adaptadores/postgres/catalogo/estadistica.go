package catalogo

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"time"
)

var _ salida.ReglasEstadisticas = (*Repositorio)(nil)

//etf:tecnico
func (*Repositorio) ReglasSPAF(ctx context.Context, corte, conocido time.Time) (d.Historia[d.ReglasSPAF], error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	// El límite superior se reconstruye por publicaciones conocidas, evitando
	// que una publicación posterior recorte la vigencia de una foto anterior.
	filas, err := tx.Query(ctx, `SELECT v.id::text,v.vigente_desde,v.publicada_en,m.codigo,m.spaf
 FROM catalogo.version v JOIN catalogo.motivo_cierre m ON m.version_id=v.id
 WHERE v.ambito='equivalencias' AND v.estado='publicada' AND v.vigente_desde<=$1 AND v.publicada_en<=$2
 ORDER BY v.vigente_desde,v.id,m.orden`, corte, conocido)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	h := d.Historia[d.ReglasSPAF]{}
	for filas.Next() {
		var id, codigo, spaf string
		var desde, registrado time.Time
		if err := filas.Scan(&id, &desde, &registrado, &codigo, &spaf); err != nil {
			return nil, err
		}
		if len(h) == 0 || h[len(h)-1].ID != id {
			if len(h) > 0 {
				h[len(h)-1].Vigencia.Hasta = desde
			}
			h = append(h, d.Revision[d.ReglasSPAF]{ID: id, Vigencia: d.Vigencia{Desde: desde}, RegistradoEn: registrado, Dato: d.Dato[d.ReglasSPAF]{Conocido: true, Valor: d.ReglasSPAF{MotivosCierre: map[d.Motivo]string{}}}})
		}
		motivo := d.MotivoDesdeCodigo(codigo)
		if motivo != "" {
			h[len(h)-1].Dato.Valor.MotivosCierre[motivo] = spaf
		}
	}
	return h, filas.Err()
}

//etf:tecnico
func (*Repositorio) CodigoEquipo(ctx context.Context, id string) (string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return "", err
	}
	var codigo string
	err = tx.QueryRow(ctx, `SELECT codigo FROM catalogo.equipo WHERE id=$1`, id).Scan(&codigo)
	return codigo, err
}

//etf:tecnico
func (*Repositorio) EquiposMemoria(ctx context.Context) ([]string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM catalogo.equipo ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
