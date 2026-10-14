package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
	"time"
)

var _ salida.Intervenciones = (*Repositorio)(nil)

func (r *Repositorio) BloquearIntervencion(ctx context.Context, id string) (dominio.FichaIntervencion, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.FichaIntervencion{}, err
	}
	var version int
	if err = e.QueryRow(ctx, `SELECT version FROM expediente.expediente WHERE id=$1 FOR UPDATE`, id).Scan(&version); err != nil { //etf:tecnico
		if errors.Is(err, pgx.ErrNoRows) {
			err = dominio.ErrNoEncontrada
		}
		return dominio.FichaIntervencion{}, err
	}
	return r.ObtenerIntervencion(ctx, id)
}
func (*Repositorio) ObtenerIntervencion(ctx context.Context, id string) (dominio.FichaIntervencion, error) {
	var r dominio.FichaIntervencion
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var equipo *string
	var fecha *time.Time
	var datos []byte
	var revision *string
	err = e.QueryRow(ctx, `SELECT e.id::text,e.equipo_id::text,e.estado,e.version,e.fecha_aceptacion,v.datos,v.id::text
 FROM expediente.expediente e LEFT JOIN expediente.consulta_intervencion v ON v.expediente_id=e.id
 AND v.vigente_desde<=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date AND (v.vigente_hasta IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date<v.vigente_hasta)
 WHERE e.id=$1`, id).Scan(&r.ID, &equipo, &r.Estado, &r.Version, &fecha, &datos, &revision) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, dominio.ErrNoEncontrada
	}
	if err != nil {
		return r, err
	}
	if equipo == nil || fecha == nil {
		return r, dominio.ErrEstado
	}
	r.EquipoID, r.Aceptacion = *equipo, *fecha
	if revision == nil {
		return r, nil
	}
	if err = json.Unmarshal(datos, &r.Intervencion); err != nil {
		return r, err
	}
	rows, err := e.Query(ctx, `SELECT m.lista,m.codigo,rm.texto_otros FROM expediente.intervencion_marca rm
 JOIN expediente.marca m ON m.id=rm.marca_id WHERE rm.revision_id=$1`, *revision) //etf:tecnico
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var lista, codigo, texto string
		if err = rows.Scan(&lista, &codigo, &texto); err != nil {
			return r, err
		}
		if marcas := r.Intervencion.Listas[lista]; marcas != nil {
			for n := range *marcas {
				if (*marcas)[n].Codigo == codigo {
					(*marcas)[n].TextoOtros = texto
				}
			}
		}
	}
	return r, rows.Err()
}
func (*Repositorio) RepetirIntervencion(ctx context.Context, usuario, clave, ruta, huella string, ahora time.Time) (dominio.ResultadoIntervencion, bool, error) {
	return repeticion[dominio.ResultadoIntervencion](ctx, usuario, clave, ruta, huella, ahora)
}
func (*Repositorio) RecordarIntervencion(ctx context.Context, usuario, clave, ruta, huella string, r dominio.ResultadoIntervencion, ahora time.Time) error {
	return recordar(ctx, usuario, clave, ruta, huella, r, ahora)
}
func (*Repositorio) GuardarIntervencion(ctx context.Context, id string, p dominio.CambioIntervencion, usuario string, ahora time.Time) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var equipo string
	err = e.QueryRow(ctx, `UPDATE expediente.expediente SET version=version+1,modificado_por=$2,modificado_en=$3 WHERE id=$1 AND version=$4 RETURNING equipo_id::text`, id, usuario, ahora, p.Version).Scan(&equipo) //etf:tecnico
	if err != nil {
		return err
	}
	conocidas := []string{}
	type marca struct {
		Lista     string   `json:"lista"`
		Codigo    string   `json:"codigo"`
		Texto     string   `json:"texto"`
		Excluidos []string `json:"excluidos"`
	}
	marcas := []marca{}
	for _, lista := range dominio.ListasIntervencion() {
		if ms := p.Intervencion.Listas[lista]; ms != nil {
			conocidas = append(conocidas, lista)
			for _, m := range *ms {
				marcas = append(marcas, marca{lista, m.Codigo, m.TextoOtros, m.MenoresExcluidos})
			}
		}
	}
	datos, err := json.Marshal(marcas)
	if err != nil {
		return err
	}
	var revision string
	err = e.QueryRow(ctx, `INSERT INTO expediente.intervencion_revision(expediente_id,equipo_id,version,vigente_desde,fase,listas_conocidas,registrado_por,registrado_en) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id::text`, id, equipo, p.Version+1, p.Desde, p.Intervencion.Fase, conocidas, usuario, ahora).Scan(&revision) //etf:tecnico
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.marca(expediente_id,equipo_id,lista,codigo)
 SELECT $1::uuid,$2::uuid,j->>'lista',j->>'codigo' FROM jsonb_array_elements($3::jsonb) j
 ON CONFLICT(expediente_id,lista,codigo) DO NOTHING`, id, equipo, datos)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.intervencion_marca(revision_id,marca_id,expediente_id,equipo_id,texto_otros)
 SELECT $1::uuid,m.id,m.expediente_id,m.equipo_id,j->>'texto' FROM expediente.marca m JOIN jsonb_array_elements($3::jsonb) j ON m.lista=j->>'lista' AND m.codigo=j->>'codigo' WHERE m.expediente_id=$2`, revision, id, datos)
	if err != nil {
		return err
	}
	_, err = e.Exec(ctx, `INSERT INTO expediente.intervencion_exclusion(revision_id,marca_id,menor_id,expediente_id,equipo_id)
 SELECT $1::uuid,m.id,x::uuid,m.expediente_id,m.equipo_id FROM expediente.marca m JOIN jsonb_array_elements($3::jsonb) j ON m.lista=j->>'lista' AND m.codigo=j->>'codigo' CROSS JOIN LATERAL jsonb_array_elements_text(j->'excluidos') x WHERE m.expediente_id=$2`, revision, id, datos)
	return err
}
func (*Repositorio) HistoriaIntervencion(ctx context.Context, id, corte string) (dominio.HistoriaIntervencion, error) {
	r := dominio.HistoriaIntervencion{Menores: []dominio.IntervencionMenor{}}
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return r, err
	}
	var datos []byte
	var desde, hasta, faseDesde *time.Time
	var version *int
	var menores []string
	// Intervención y pertenencia de menores se leen con la misma instantánea.
	err = e.QueryRow(ctx, `SELECT v.datos,v.vigente_desde,v.vigente_hasta,v.version,v.fase_desde,
 ARRAY(SELECT m.id::text FROM expediente.consulta_menor m WHERE m.expediente_id=e.id AND m.vigente_desde<=$2::date AND (m.vigente_hasta IS NULL OR $2::date<m.vigente_hasta) ORDER BY m.orden)
 FROM expediente.expediente e LEFT JOIN expediente.consulta_intervencion v ON v.expediente_id=e.id AND v.vigente_desde<=$2::date AND (v.vigente_hasta IS NULL OR $2::date<v.vigente_hasta)
 WHERE e.id=$1`, id, corte).Scan(&datos, &desde, &hasta, &version, &faseDesde, &menores) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return r, dominio.ErrNoEncontrada
	}
	if err != nil {
		return r, err
	}
	if desde != nil {
		if err = json.Unmarshal(datos, &r.Intervencion); err != nil {
			return r, err
		}
		r.Vigencia = vigencia(*desde, hasta, *version)
	}
	if faseDesde != nil {
		fecha := faseDesde.Format("2006-01-02")
		r.FaseDesde = &fecha
	}
	desconocida := dominio.Intervencion{Listas: map[string]*[]dominio.MarcaIntervencion{"coordinacion": nil, "factor_familia": nil, "factor_menor": nil, "tipologia": nil}}
	for _, id := range menores {
		if r.Intervencion == nil {
			r.Menores = append(r.Menores, desconocida.Aplicar(id))
		} else {
			r.Menores = append(r.Menores, r.Intervencion.Aplicar(id))
		}
	}
	return r, nil
}
