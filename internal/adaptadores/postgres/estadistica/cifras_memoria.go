package estadistica

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"github.com/jackc/pgx/v5"
)

type CifrasMemoria struct{}

var _ salida.CifrasPublicadas = (*CifrasMemoria)(nil)

//etf:tecnico
func (*CifrasMemoria) Consultar(ctx context.Context, f d.FiltroMemoria) (d.CifrasMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.CifrasMemoria{}, err
	}
	var datos []byte
	err = tx.QueryRow(ctx, `SELECT cifras FROM estadistica.api_cifras_consulta WHERE anio=$1 AND equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid`, f.Anio, f.EquipoID).Scan(&datos)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.CifrasMemoria{}, d.ErrCifrasNoPublicadasMemoria
	}
	if err != nil {
		return d.CifrasMemoria{}, err
	}
	defer clear(datos)
	var cifras d.CifrasMemoria
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cifras); err != nil {
		return d.CifrasMemoria{}, d.ErrDisponibleMemoria
	}
	if dec.Decode(new(any)) != io.EOF || cifras.Anio != f.Anio || cifras.EquipoID != f.EquipoID {
		return d.CifrasMemoria{}, d.ErrDisponibleMemoria
	}
	return cifras, nil
}

//etf:tecnico
func (*CifrasMemoria) Publicar(ctx context.Context, cifras d.CifrasMemoria) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	datos, err := json.Marshal(cifras)
	if err != nil {
		return err
	}
	defer clear(datos)
	_, err = tx.Exec(ctx, `INSERT INTO estadistica.foto_totales(anio,equipo_id,fecha,cifras,publicado_por)
 VALUES($1,nullif($2,'')::uuid,$3,$4,nullif(current_setting('app.usuario_id',true),'')::uuid)
 ON CONFLICT(anio,equipo_id) DO UPDATE SET fecha=excluded.fecha,cifras=excluded.cifras,publicado_por=excluded.publicado_por
 WHERE estadistica.foto_totales.fecha<=excluded.fecha`, cifras.Anio, cifras.EquipoID, cifras.ConocidoEn, datos)
	return err
}
