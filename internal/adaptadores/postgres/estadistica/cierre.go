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

var _ salida.Cierres = (*Repositorio)(nil)

//etf:tecnico
func (*Repositorio) BloquearMemorias(ctx context.Context) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(4606,1)`)
	return err
}

//etf:tecnico
func (*Repositorio) EstadoMemoria(ctx context.Context, f d.FiltroMemoria, crear bool, usuario string, ahora time.Time) (d.EstadoMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.EstadoMemoria{}, err
	}
	if crear {
		_, err = tx.Exec(ctx, `INSERT INTO estadistica.memoria(anio,equipo_id,version,textos,modificado_por,modificado_en) VALUES($1,nullif($2,'')::uuid,1,'{}',$3,$4) ON CONFLICT(anio,equipo_id) DO NOTHING`, f.Anio, f.EquipoID, usuario, ahora)
		if err != nil {
			return d.EstadoMemoria{}, err
		}
	}
	r := d.EstadoMemoria{FiltroMemoria: f, Estado: "borrador"}
	err = tx.QueryRow(ctx, `SELECT id::text,estado,version,coalesce(revision_actual,0) FROM estadistica.memoria WHERE anio=$1 AND equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid`, f.Anio, f.EquipoID).Scan(&r.ID, &r.Estado, &r.Version, &r.RevisionActual)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, nil
	}
	return r, err
}

//etf:tecnico
func (*Repositorio) LeerRevision(ctx context.Context, f d.FiltroMemoria, revision int) (d.RevisionMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.RevisionMemoria{}, err
	}
	var r d.RevisionMemoria
	var b []byte
	err = tx.QueryRow(ctx, `SELECT r.id::text,r.memoria_id::text,r.revision,r.informe,r.huella,r.motor,r.cerrada_por::text,r.cerrada_en
 FROM estadistica.memoria_revision r JOIN estadistica.memoria m ON m.id=r.memoria_id
 WHERE m.anio=$1 AND m.equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid AND r.revision=CASE WHEN $3=0 THEN m.revision_actual ELSE $3 END`, f.Anio, f.EquipoID, revision).Scan(&r.ID, &r.MemoriaID, &r.Revision, &b, &r.Huella, &r.Motor, &r.CerradaPor, &r.CerradaEn)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, d.ErrRevisionMemoria
	}
	if err != nil {
		return r, err
	}
	defer clear(b)
	err = json.Unmarshal(b, &r.Informe)
	return r, err
}

//etf:tecnico
func (*Repositorio) RepetirActo(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (d.ResultadoActoMemoria, bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.ResultadoActoMemoria{}, false, err
	}
	var anterior, rutaAnterior string
	var b []byte
	err = tx.QueryRow(ctx, `SELECT huella_peticion,ruta,respuesta FROM estadistica.peticion_idempotente WHERE usuario_id=$1 AND clave=$2 AND creada_en>$3 ORDER BY creada_en DESC LIMIT 1`, usuario, clave, ahora.Add(-24*time.Hour)).Scan(&anterior, &rutaAnterior, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.ResultadoActoMemoria{}, false, nil
	}
	if err != nil {
		return d.ResultadoActoMemoria{}, false, err
	}
	defer clear(b)
	if anterior != huella || rutaAnterior != ruta {
		return d.ResultadoActoMemoria{}, false, d.ErrIdempotenciaMemoria
	}
	var r d.ResultadoActoMemoria
	err = json.Unmarshal(b, &r)
	r.Repetida = true
	return r, true, err
}

//etf:tecnico
func (r *Repositorio) GuardarActo(ctx context.Context, e d.EstadoMemoria, p d.ActoMemoria, revision *d.RevisionMemoria, bases []d.InformeMemoria, usuario, clave, ruta, huella string, ahora time.Time) (d.ResultadoActoMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.ResultadoActoMemoria{}, err
	}
	if e.ID == "" {
		e, err = r.EstadoMemoria(ctx, e.FiltroMemoria, true, usuario, ahora)
		if err != nil {
			return d.ResultadoActoMemoria{}, err
		}
	}
	accion := "reabrir"
	estado := "borrador"
	numero := e.RevisionActual
	if revision != nil {
		accion = "cerrar"
		estado = "cerrada"
		numero++
	}
	_, err = tx.Exec(ctx, `INSERT INTO estadistica.memoria_acto(memoria_id,version,accion,motivo,usuario_id,ocurrido_en) VALUES($1,$2,$3,nullif($4,''),$5,$6)`, e.ID, e.Version+1, accion, p.Motivo, usuario, ahora)
	if err != nil {
		return d.ResultadoActoMemoria{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE estadistica.memoria SET estado=$2,revision_actual=$3,version=version+1,modificado_por=$4,modificado_en=$5 WHERE id=$1 AND version=$6`, e.ID, estado, numero, usuario, ahora, e.Version)
	if err != nil {
		return d.ResultadoActoMemoria{}, err
	}
	if tag.RowsAffected() != 1 {
		return d.ResultadoActoMemoria{}, &d.ConflictoMemoria{Actual: e.Version}
	}
	resultado := d.ResultadoActoMemoria{EstadoMemoria: e}
	resultado.Estado = estado
	resultado.Version++
	resultado.RevisionActual = numero
	if revision != nil {
		b, err := json.Marshal(revision.Informe)
		if err != nil {
			return resultado, err
		}
		defer clear(b)
		err = tx.QueryRow(ctx, `INSERT INTO estadistica.memoria_revision(memoria_id,revision,informe,huella,motor,cerrada_por,cerrada_en) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`, e.ID, numero, b, revision.Huella, revision.Motor, usuario, ahora).Scan(&resultado.RevisionID)
		if err != nil {
			return resultado, err
		}
		resultado.Huella = revision.Huella
		for _, base := range bases {
			b, err := json.Marshal(base)
			if err != nil {
				return resultado, err
			}
			// La proyección publicada al equipo nunca incluye narrativas.
			_, err = tx.Exec(ctx, `INSERT INTO estadistica.memoria_base_equipo(memoria_id,revision,equipo_id,informe) VALUES($1,$2,$3,$4::jsonb-'Narrativas'-'Composicion'-'Fuentes')`, e.ID, numero, base.EquipoID, b)
			clear(b)
			if err != nil {
				return resultado, err
			}
		}
	}
	b, err := json.Marshal(resultado)
	if err != nil {
		return resultado, err
	}
	defer clear(b)
	_, err = tx.Exec(ctx, `INSERT INTO estadistica.peticion_idempotente(usuario_id,clave,ruta,huella_peticion,respuesta,creada_en) VALUES($1,$2,$3,$4,$5,$6)`, usuario, clave, ruta, huella, b, ahora)
	return resultado, err
}

//etf:tecnico
func (*Repositorio) BasesEquipo(ctx context.Context, equipo string) ([]d.BaseEquipoMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT memoria_id::text,revision,informe FROM estadistica.bases_equipo($1)`, equipo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := []d.BaseEquipoMemoria{}
	for rows.Next() {
		var r d.BaseEquipoMemoria
		var b []byte
		if err := rows.Scan(&r.MemoriaID, &r.Revision, &b); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, &r.Informe)
		clear(b)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, r)
	}
	return resultado, rows.Err()
}

//etf:tecnico
func (*Repositorio) AvisarDiscrepancia(ctx context.Context, dif d.DiscrepanciaMemoria) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	codigos := []string{}
	for _, a := range dif.Apartados {
		codigos = append(codigos, string(a))
	}
	var evento any
	if dif.EventoID > 0 {
		evento = int64(dif.EventoID)
	}
	_, err = tx.Exec(ctx, `SELECT estadistica.avisar_discrepancia($1,$2,$3,nullif($4,'')::uuid,$5,$6,$7)`, dif.MemoriaID, dif.Revision, dif.EquipoID, dif.ExpedienteID, evento, codigos, dif.DetectadaEn)
	return err
}

//etf:tecnico
func (*Repositorio) Discrepancias(ctx context.Context, f d.FiltroMemoria) ([]d.DiscrepanciaMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT d.id::text,d.memoria_id::text,d.revision,d.equipo_id::text,coalesce(d.expediente_id::text,''),coalesce(d.evento_id,0),d.apartados,d.detectada_en FROM estadistica.memoria_discrepancia d JOIN estadistica.memoria m ON m.id=d.memoria_id WHERE m.anio=$1 AND ($2='' OR m.equipo_id=nullif($2,'')::uuid) ORDER BY d.detectada_en,d.id`, f.Anio, f.EquipoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := []d.DiscrepanciaMemoria{}
	for rows.Next() {
		var v d.DiscrepanciaMemoria
		var codigos []string
		if err := rows.Scan(&v.ID, &v.MemoriaID, &v.Revision, &v.EquipoID, &v.ExpedienteID, &v.EventoID, &codigos, &v.DetectadaEn); err != nil {
			return nil, err
		}
		for _, c := range codigos {
			v.Apartados = append(v.Apartados, d.Apartado(c))
		}
		resultado = append(resultado, v)
	}
	return resultado, rows.Err()
}

//etf:tecnico
func (*Repositorio) RevisionesCerradas(ctx context.Context, f d.FiltroMemoria) ([]d.RevisionMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT r.id::text,r.memoria_id::text,r.revision,r.informe,r.huella,r.motor,r.cerrada_por::text,r.cerrada_en FROM estadistica.memoria_revision r JOIN estadistica.memoria m ON m.id=r.memoria_id AND m.revision_actual=r.revision WHERE m.estado='cerrada' AND m.anio=$1 AND ($2='' OR m.equipo_id=nullif($2,'')::uuid) ORDER BY m.equipo_id NULLS FIRST`, f.Anio, f.EquipoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := []d.RevisionMemoria{}
	for rows.Next() {
		var r d.RevisionMemoria
		var b []byte
		if err := rows.Scan(&r.ID, &r.MemoriaID, &r.Revision, &b, &r.Huella, &r.Motor, &r.CerradaPor, &r.CerradaEn); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, &r.Informe)
		clear(b)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, r)
	}
	return resultado, rows.Err()
}

//etf:tecnico
func (*Repositorio) RevisionConocida(ctx context.Context, f d.FiltroMemoria, conocido time.Time) (d.RevisionMemoria, bool, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.RevisionMemoria{}, false, err
	}
	var r d.RevisionMemoria
	var b []byte
	err = tx.QueryRow(ctx, `SELECT r.id::text,r.memoria_id::text,r.revision,r.informe,r.huella,r.motor,r.cerrada_por::text,r.cerrada_en
 FROM estadistica.memoria_revision r JOIN estadistica.memoria m ON m.id=r.memoria_id
 WHERE m.anio=$1 AND m.equipo_id IS NOT DISTINCT FROM nullif($2,'')::uuid AND r.cerrada_en<=$3
 AND (SELECT a.accion FROM estadistica.memoria_acto a WHERE a.memoria_id=m.id AND a.ocurrido_en<=$3 ORDER BY a.version DESC LIMIT 1)='cerrar'
 ORDER BY r.revision DESC LIMIT 1`, f.Anio, f.EquipoID, conocido).Scan(&r.ID, &r.MemoriaID, &r.Revision, &b, &r.Huella, &r.Motor, &r.CerradaPor, &r.CerradaEn)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	defer clear(b)
	err = json.Unmarshal(b, &r.Informe)
	return r, true, err
}
