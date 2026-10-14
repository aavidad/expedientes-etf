package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

type LecturaEstadistica struct {
	Reglas         catalogo.ReglasEstadisticas
	CalculoInterno bool
}

//etf:tecnico
func (*LecturaEstadistica) EquipoMemoria(ctx context.Context, id string) (string, time.Time, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	var equipo string
	var modificado time.Time
	err = tx.QueryRow(ctx, `SELECT coalesce(equipo_id::text,''),coalesce(modificado_en,fecha_recepcion) FROM expediente.expediente WHERE id=$1`, id).Scan(&equipo, &modificado)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, nil
	}
	return equipo, modificado, err
}

var _ salida.LecturaMemoria = (*LecturaEstadistica)(nil)
var _ salida.LecturaSPAF = (*LecturaEstadistica)(nil)

type datosLectura struct {
	Expediente d.Expediente
	Numero     string
	Recepcion  time.Time
	Servicio   string
	Fases      d.Historia[string]
}
type fechaLectura struct{ time.Time }

func (f *fechaLectura) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	v, err := time.Parse("2006-01-02", s)
	f.Time = v
	return err
}

type episodioLectura struct {
	Creado     time.Time `json:"creado_en"`
	ID         string
	Apertura   fechaLectura  `json:"apertura_fecha"`
	Reapertura string        `json:"reapertura_de_id"`
	Cierre     *fechaLectura `json:"cierre_fecha"`
	Motivo     string        `json:"motivo_cierre"`
	Cerrado    *time.Time    `json:"cerrado_en"`
}
type hitoLectura struct {
	EpisodioID string `json:"episodio_id"`
	Tipo       string
	Fecha      fechaLectura
	Confirmado time.Time `json:"confirmado_en"`
}
type cambioLectura struct {
	ID         string
	Estado     string    `json:"estado_nuevo"`
	Registrado time.Time `json:"cambiado_en"`
}
type vigenciaLectura struct {
	ID, Atributo string
	Valor        json.RawMessage
	Desde        fechaLectura `json:"vigente_desde"`
	Registrado   time.Time    `json:"registrado_en"`
}

//etf:tecnico
func (l *LecturaEstadistica) leer(ctx context.Context, equipo string, corte, conocido time.Time) ([]datosLectura, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	// Una sola instantánea SQL reúne todas las partes, sin consultas por familia.
	consulta := `SELECT e.id::text,e.equipo_id::text,coalesce(e.numero,''),e.fecha_recepcion,e.servicio,
 coalesce((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.apertura_fecha,p.id) FROM expediente.api_estadistica_episodios p WHERE p.expediente_id=e.id AND p.creado_en<=$2),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(h) ORDER BY h.fecha,h.id) FROM expediente.api_estadistica_hitos h WHERE h.expediente_id=e.id AND h.confirmado_en<=$2),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.vigente_desde,v.registrado_en,v.id) FROM expediente.api_estadistica_vigencias v WHERE v.expediente_id=e.id AND v.registrado_en<=$2),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(f) ORDER BY f.vigente_desde) FROM (SELECT DISTINCT ON(vigente_desde) * FROM expediente.api_estadistica_familias f WHERE f.expediente_id=e.id AND f.registrado_en<=$2 ORDER BY vigente_desde,version DESC) f),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(a) ORDER BY a.vigente_desde) FROM (SELECT DISTINCT ON(vigente_desde) * FROM expediente.api_estadistica_adultos a WHERE a.expediente_id=e.id AND a.registrado_en<=$2 ORDER BY vigente_desde,version DESC) a),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(c) ORDER BY c.cambiado_en,c.id) FROM expediente.api_estadistica_estados c WHERE c.expediente_id=e.id AND c.cambiado_en<=$2),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(m) ORDER BY m.vigente_desde) FROM (SELECT DISTINCT ON(vigente_desde) * FROM expediente.api_estadistica_menores m WHERE m.expediente_id=e.id AND m.registrado_en<=$2 AND m.vigente_desde<=$3::date ORDER BY vigente_desde,version DESC) m),'[]'),
 coalesce((SELECT jsonb_agg(to_jsonb(i) ORDER BY i.vigente_desde) FROM (SELECT DISTINCT ON(vigente_desde) * FROM expediente.api_estadistica_intervencion i WHERE i.expediente_id=e.id AND i.registrado_en<=$2 AND i.vigente_desde<=$3::date ORDER BY vigente_desde,version DESC) i),'[]')
 FROM expediente.api_estadistica_expedientes e WHERE ($1='' OR e.equipo_id=nullif($1,'')::uuid) AND e.fecha_recepcion<=$2 ORDER BY e.id`
	if l.CalculoInterno {
		consulta = strings.ReplaceAll(consulta, "expediente.api_estadistica_", "estadistica.calculo_")
	}
	filas, err := tx.Query(ctx, consulta, equipo, conocido, corte)

	if err != nil {
		return nil, err
	}
	defer filas.Close()
	datos := []datosLectura{}
	for filas.Next() {
		var v datosLectura
		var episodios, hitos, vigencias, familias, adultos, estados, menores, intervenciones []byte
		if err = filas.Scan(&v.Expediente.ID, &v.Expediente.EquipoID, &v.Numero, &v.Recepcion, &v.Servicio, &episodios, &hitos, &vigencias, &familias, &adultos, &estados, &menores, &intervenciones); err != nil {
			return nil, err
		}
		var eps []episodioLectura
		var hs []hitoLectura
		var vs []vigenciaLectura
		if err = json.Unmarshal(episodios, &eps); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(hitos, &hs); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(vigencias, &vs); err != nil {
			return nil, err
		}
		var fs, as []vigenciaLectura
		if err = json.Unmarshal(familias, &fs); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(adultos, &as); err != nil {
			return nil, err
		}
		if err = incorporarFamilia(&v.Expediente, fs, as); err != nil {
			return nil, err
		}
		if err = incorporarMenoresIntervencion(&v, menores, intervenciones); err != nil {
			return nil, err
		}
		var cs []cambioLectura
		if err = json.Unmarshal(estados, &cs); err != nil {
			return nil, err
		}
		v.Expediente.Estados = historiaEstados(v.Expediente.ID, v.Recepcion, cs)

		v.Expediente.Consentimiento = historiaConsentimiento(eps, hs)

		for _, ep := range eps {
			r := d.Episodio{ID: ep.ID, Apertura: ep.Apertura.Time, ReaperturaDe: ep.Reapertura}
			if ep.Cierre != nil && ep.Cerrado != nil && !ep.Cerrado.After(conocido) {
				r.Cierre = ep.Cierre.Time
				r.Motivo = d.MotivoDesdeCodigo(ep.Motivo)
			}
			for _, h := range hs {
				if h.EpisodioID != ep.ID {
					continue
				}
				switch h.Tipo {
				case "aceptacion":
					r.Aceptacion = h.Fecha.Time
				case "cierre":
					r.Cierre = h.Fecha.Time
				}
			}
			v.Expediente.Episodios = append(v.Expediente.Episodios, r)
		}
		porID := map[string]int{}
		for i, ep := range v.Expediente.Episodios {
			porID[ep.ID] = i
		}
		for intento := 0; intento < len(v.Expediente.Episodios); intento++ {
			cambios := false
			for i := range v.Expediente.Episodios {
				ep := &v.Expediente.Episodios[i]
				if ep.ReaperturaDe == "" || !ep.Aceptacion.IsZero() {
					continue
				}
				if padre, ok := porID[ep.ReaperturaDe]; ok && !v.Expediente.Episodios[padre].Aceptacion.IsZero() {
					ep.Aceptacion = v.Expediente.Episodios[padre].Aceptacion
					cambios = true
				}
			}
			if !cambios {
				break
			}
		}

		for _, fila := range vs {
			if fila.Atributo != "subprograma" {
				continue
			}
			var codigo string
			if err = json.Unmarshal(fila.Valor, &codigo); err != nil {
				return nil, err
			}
			r := d.Revision[d.Subprograma]{ID: fila.ID, Vigencia: d.Vigencia{Desde: fila.Desde.Time}, RegistradoEn: fila.Registrado, Dato: d.Dato[d.Subprograma]{Valor: subprogramaLectura(codigo), Conocido: subprogramaLectura(codigo) != ""}}
			h := &v.Expediente.Subprograma
			if len(*h) > 0 {
				(*h)[len(*h)-1].Vigencia.Hasta = r.Vigencia.Desde
			}
			*h = append(*h, r)
		}
		v.Expediente.ServicioDerivante = d.Historia[string]{{ID: v.Expediente.ID + ":servicio", Vigencia: d.Vigencia{Desde: v.Recepcion}, RegistradoEn: v.Recepcion, Dato: d.Dato[string]{Valor: v.Servicio, Conocido: v.Servicio != ""}}}
		datos = append(datos, v)
	}
	return datos, filas.Err()
}
func subprogramaLectura(c string) d.Subprograma {
	switch c {
	case "RIESGO_SOCIAL":
		return d.Riesgo
	case "REUNIFICACION":
		return d.Reunificacion
	}
	return ""
}
func (l *LecturaEstadistica) ConsultarMemoria(ctx context.Context, equipo string, corte, conocido time.Time) ([]d.Expediente, error) {
	datos, err := l.leer(ctx, equipo, corte, conocido)
	if err != nil {
		return nil, err
	}
	r := make([]d.Expediente, 0, len(datos))
	for _, v := range datos {
		r = append(r, v.Expediente)
	}
	return r, nil
}
func (l *LecturaEstadistica) ConsultarSPAF(ctx context.Context, f d.FiltroSPAF, corte, conocido time.Time) ([]d.ExpedienteSPAF, d.Historia[d.ReglasSPAF], error) {
	if l.Reglas == nil {
		return nil, nil, d.ErrDisponibleSPAF
	}
	h, err := l.Reglas.ReglasSPAF(ctx, corte, conocido)
	if err != nil {
		return nil, nil, err
	}
	datos, err := l.leer(ctx, f.EquipoID, corte, conocido)
	if err != nil {
		return nil, nil, err
	}
	r := make([]d.ExpedienteSPAF, 0, len(datos))
	for _, v := range datos {
		exp := d.ExpedienteSPAF{ID: v.Expediente.ID, EquipoID: v.Expediente.EquipoID, Numero: v.Numero, Episodios: v.Expediente.Episodios}
		ficha := d.DatosSPAF{ServicioDerivante: d.Dato[string]{Valor: v.Servicio, Conocido: v.Servicio != ""}}
		menores, versionMenores, err := v.Expediente.Menores.ValorEn(corte, conocido)
		if err != nil {
			return nil, nil, err
		}
		ficha.Menores = d.Dato[[]d.MenorSPAF]{Conocido: menores.Conocido, Valor: []d.MenorSPAF{}}
		for _, m := range menores.Valor {
			ficha.Menores.Valor = append(ficha.Menores.Valor, d.MenorSPAF{ID: m.ID, Nacimiento: d.Dato[time.Time]{Valor: m.Nacimiento, Conocido: !m.Nacimiento.IsZero()}, CentroProteccion: m.CentroProteccion})
		}
		fase, versionFase, err := v.Fases.ValorEn(corte, conocido)
		if err != nil {
			return nil, nil, err
		}
		ficha.FaseIntervencion = fase
		exp.Ficha = d.Historia[d.DatosSPAF]{{ID: exp.ID + ":" + versionMenores + ":" + versionFase, Vigencia: d.Vigencia{Desde: corte}, RegistradoEn: conocido, Dato: d.Dato[d.DatosSPAF]{Conocido: true, Valor: ficha}}}
		r = append(r, exp)
	}
	return r, h, nil
}
