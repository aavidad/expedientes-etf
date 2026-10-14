package http

import (
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
)

type ModuloCierreMemoria struct{ Servicio entrada.CierresMemoria }

func (m ModuloCierreMemoria) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { m.registrar(mux) })
}
func (m ModuloCierreMemoria) registrar(mux *muxHTTP) {
	for _, accion := range []string{"estado", "instantanea", "discrepancias", "cerrar", "reabrir"} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if m.Servicio == nil {
				errorMemoria(w, entrada.ErrDisponibleMemoria)
				return
			}
			a := actorMemoria(r)
			rechazar := func() {
				err := m.Servicio.RechazarCierreMemoria(r.Context(), a)
				if err == nil {
					err = entrada.ErrActoMemoria
				}
				errorMemoria(w, err)
			}
			q, err := url.ParseQuery(r.URL.RawQuery)
			anio, errAnio := strconv.Atoi(q.Get("anio"))
			valido := err == nil && errAnio == nil && len(q["anio"]) == 1 && len(q["etf"]) <= 1 && (!q.Has("etf") || q.Get("etf") != "")
			revision := 0
			for k := range q {
				if k != "anio" && k != "etf" && !(accion == "instantanea" && k == "revision") {
					valido = false
				}
			}
			if q.Has("revision") {
				var err error
				revision, err = strconv.Atoi(q.Get("revision"))
				valido = valido && err == nil && revision > 0 && len(q["revision"]) == 1
			}
			if !valido {
				rechazar()
				return
			}
			f := entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get("etf")}
			var resultado any
			switch accion {
			case "estado":
				resultado, err = m.Servicio.EstadoMemoria(r.Context(), a, f)
			case "instantanea":
				resultado, err = m.Servicio.InstantaneaMemoria(r.Context(), a, f, revision)
			case "discrepancias":
				resultado, err = m.Servicio.DiscrepanciasMemoria(r.Context(), a, f)
			default:
				tipo, _, errTipo := mime.ParseMediaType(r.Header.Get("Content-Type"))
				var cuerpo struct {
					Version *int   `json:"version"`
					Motivo  string `json:"motivo"`
				}
				decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
				decoder.DisallowUnknownFields()
				if errTipo != nil || tipo != "application/json" || decoder.Decode(&cuerpo) != nil || decoder.Decode(new(any)) != io.EOF || cuerpo.Version == nil {
					rechazar()
					return
				}
				p := entrada.ActoMemoria{Version: *cuerpo.Version, Motivo: cuerpo.Motivo}
				if accion == "cerrar" {
					resultado, err = m.Servicio.CerrarMemoria(r.Context(), a, f, r.Header.Get("Idempotency-Key"), p)
				} else {
					resultado, err = m.Servicio.ReabrirMemoria(r.Context(), a, f, r.Header.Get("Idempotency-Key"), p)
				}
			}
			if err != nil {
				errorMemoria(w, err)
				return
			}
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			responderJSON(w, http.StatusOK, resultado)
		})
		ruta := "/api/estadistica/memoria/" + accion
		switch accion {
		case "cerrar":
			mux.declarar(handler, escritura(ruta, []string{http.MethodPost}, "coordinacion", "profesional_etf"))
		case "reabrir":
			mux.declarar(handler, escritura(ruta, []string{http.MethodPost}, "coordinacion"))
		case "discrepancias":
			mux.declarar(handler, lectura(ruta, "coordinacion"))
		default:
			mux.declarar(handler, lectura(ruta, "coordinacion", "profesional_etf"))
		}
	}
}
