package http

import (
	"context"
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ModuloDesgloseMemoria struct{ Servicio entrada.DesgloseCifraMemoria }

func (m ModuloDesgloseMemoria) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(m.leer), lectura("/api/memoria/{anio}/desglose/{apartado}", "coordinacion", "profesional_etf"))
	}, pantalla("/memoria/desglose", "", "coordinacion", "profesional_etf")), "/memoria/desglose")
}
func (m ModuloDesgloseMemoria) leer(w http.ResponseWriter, r *http.Request) {
	cabecerasCifras(w)
	q, e := url.ParseQuery(r.URL.RawQuery)
	valido := e == nil
	for k, vs := range q {
		if len(vs) != 1 || (k != "etf" && k != "celda" && k != "medida" && k != "revision" && k != "clave") || vs[0] == "" {
			valido = false
		}
	}
	anio, e := strconv.Atoi(r.PathValue("anio"))
	valido = valido && e == nil
	celda, e := strconv.Atoi(q.Get("celda"))
	valido = valido && e == nil
	revision := 0
	if q.Has("revision") {
		revision, e = strconv.Atoi(q.Get("revision"))
		valido = valido && e == nil
	}
	var clave *entrada.ClaveCeldaMemoria
	if q.Has("clave") {
		clave = &entrada.ClaveCeldaMemoria{}
		dec := json.NewDecoder(strings.NewReader(q.Get("clave")))
		dec.DisallowUnknownFields()
		valido = valido && len(q.Get("clave")) <= 4096 && dec.Decode(clave) == nil && dec.Decode(new(any)) == io.EOF
	}
	if !valido {
		rechazarFiltroMemoria(w, r, m.Servicio)
		return
	}
	if m.Servicio == nil {
		errorMemoria(w, entrada.ErrDisponibleMemoria)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result, err := m.Servicio.DesglosarMemoria(ctx, actorMemoria(r), entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get("etf")}, entrada.Apartado(r.PathValue("apartado")), entrada.SeleccionDesglose{Celda: celda, Medida: q.Get("medida"), Revision: revision, Clave: clave})
	if err != nil {
		errorNavegacionMemoria(w, err)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	responderJSON(w, http.StatusOK, result)
}
