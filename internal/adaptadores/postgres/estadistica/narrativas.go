package estadistica

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"github.com/jackc/pgx/v5"
	"time"
)

type Repositorio struct{}

var _ salida.Narrativas = (*Repositorio)(nil)

//etf:tecnico
func (*Repositorio) LeerNarrativas(ctx context.Context, f d.FiltroMemoria) (d.Narrativas, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Narrativas{}, err
	}
	var b []byte
	var id string
	var version int
	err = tx.QueryRow(ctx, `SELECT id::text,version,textos FROM estadistica.memoria WHERE anio=$1 AND equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid`, f.Anio, f.EquipoID).Scan(&id, &version, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Narrativas{FiltroMemoria: f}, nil
	}
	if err != nil {
		return d.Narrativas{}, err
	}
	defer clear(b)
	var n d.Narrativas
	if err = json.Unmarshal(b, &n); err != nil {
		return d.Narrativas{}, err
	}
	n.FiltroMemoria = f
	n.ID = id
	n.Version = version
	return n, nil
}

//etf:tecnico
func (r *Repositorio) GuardarNarrativas(ctx context.Context, n d.Narrativas, usuario string, ahora time.Time) (d.Narrativas, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Narrativas{}, err
	}
	// La petición nunca decide el ID ni la versión resultante.
	n.ID = ""
	b, err := json.Marshal(n)
	if err != nil {
		return d.Narrativas{}, err
	}
	defer clear(b)
	var id string
	var version int
	if n.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO estadistica.memoria(anio,equipo_id,version,textos,modificado_por,modificado_en) VALUES($1,nullif($2,'')::uuid,1,$3,$4,$5)
 ON CONFLICT(anio,equipo_id) DO NOTHING RETURNING id::text,version`, n.Anio, n.EquipoID, b, usuario, ahora).Scan(&id, &version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE estadistica.memoria SET textos=$3,version=version+1,modificado_por=$4,modificado_en=$5
 WHERE anio=$1 AND equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid AND version=$6 AND estado='borrador' RETURNING id::text,version`, n.Anio, n.EquipoID, b, usuario, ahora, n.Version).Scan(&id, &version)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		var actual int
		var estado string
		err = tx.QueryRow(ctx, `SELECT version,estado FROM estadistica.memoria WHERE anio=$1 AND equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid`, n.Anio, n.EquipoID).Scan(&actual, &estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return d.Narrativas{}, &d.ConflictoMemoria{}
		}
		if err != nil {
			return d.Narrativas{}, err
		}
		if estado == "cerrada" {
			return d.Narrativas{}, d.ErrCerradaMemoria
		}
		return d.Narrativas{}, &d.ConflictoMemoria{Actual: actual}
	}
	if err != nil {
		return d.Narrativas{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO estadistica.memoria_narrativa_historia(memoria_id,version,textos,registrado_por,registrado_en) VALUES($1,$2,$3,$4,$5)`, id, version, b, usuario, ahora)
	if err != nil {
		return d.Narrativas{}, err
	}
	n.ID = id
	n.Version = version
	return n, nil
}

//etf:tecnico
func (*Repositorio) NarrativasConocidas(ctx context.Context, f d.FiltroMemoria, conocido time.Time) (d.Narrativas, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Narrativas{}, err
	}
	var id string
	var version int
	var b []byte
	err = tx.QueryRow(ctx, `SELECT m.id::text,h.version,h.textos FROM estadistica.memoria m JOIN estadistica.memoria_narrativa_historia h ON h.memoria_id=m.id
 WHERE m.anio=$1 AND m.equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid AND h.registrado_en<=$3 ORDER BY h.version DESC LIMIT 1`, f.Anio, f.EquipoID, conocido).Scan(&id, &version, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Narrativas{FiltroMemoria: f}, nil
	}
	if err != nil {
		return d.Narrativas{}, err
	}
	defer clear(b)
	var n d.Narrativas
	if err = json.Unmarshal(b, &n); err != nil {
		return d.Narrativas{}, err
	}
	n.ID = id
	n.Version = version
	n.FiltroMemoria = f
	return n, nil
}
