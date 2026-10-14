package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type ModuloConsultaMemoria struct{ Servicio entrada.ConsultaCifrasMemoria }
type ModuloNavegacionMemoria struct{ Servicio entrada.NavegacionMemoria }

func (m ModuloConsultaMemoria) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(m.leer), lectura("/api/consulta/memoria/{anio}", "consulta"))
		mux.declarar(http.HandlerFunc(m.word), escritura("/api/consulta/memoria/{anio}/word", []string{http.MethodPost}, "consulta"))
	})
}
func (m ModuloNavegacionMemoria) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(m.leer), lectura("/api/memoria/{anio}", "coordinacion", "profesional_etf"))
		mux.declarar(http.HandlerFunc(m.leer), lectura("/api/memoria/{anio}/cifras/{apartado}", "coordinacion", "profesional_etf"))
	})
}
func filtroNavegacion(r *http.Request, claveEquipo string) (entrada.FiltroMemoria, bool) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	anio, e := strconv.Atoi(r.PathValue("anio"))
	valido := err == nil && e == nil && anio >= 1 && anio <= 9998 && len(q[claveEquipo]) <= 1 && (!q.Has(claveEquipo) || q.Get(claveEquipo) != "")
	for k := range q {
		if k != claveEquipo {
			valido = false
		}
	}
	return entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get(claveEquipo)}, valido
}
func cabecerasCifras(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func (m ModuloConsultaMemoria) leer(w http.ResponseWriter, r *http.Request) {
	cabecerasCifras(w)
	f, ok := filtroNavegacion(r, "equipo")
	if !ok {
		rechazarFiltroMemoria(w, r, m.Servicio)
		return
	}
	if m.Servicio == nil {
		errorNavegacionMemoria(w, entrada.ErrDisponibleMemoria)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cifras, err := m.Servicio.ConsultarCifras(ctx, actorMemoria(r), f)
	if err != nil {
		errorNavegacionMemoria(w, err)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	responderJSON(w, http.StatusOK, cifras)
}
func (m ModuloConsultaMemoria) word(w http.ResponseWriter, r *http.Request) {
	cabecerasCifras(w)
	f, ok := filtroNavegacion(r, "equipo")
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var p struct {
		Idioma string `json:"idioma"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if !ok || err != nil || tipo != "application/json" || dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF || len(p.Idioma) > 63 || !idiomaCifrasValido(p.Idioma) {
		rechazarFiltroMemoria(w, r, m.Servicio)
		return
	}
	if m.Servicio == nil {
		errorNavegacionMemoria(w, entrada.ErrDisponibleMemoria)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	b, err := m.Servicio.DescargarCifras(ctx, actorMemoria(r), f, p.Idioma)
	if err != nil {
		errorNavegacionMemoria(w, err)
		return
	}
	defer clear(b)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="memoria-cifras.docx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
func (m ModuloNavegacionMemoria) leer(w http.ResponseWriter, r *http.Request) {
	cabecerasCifras(w)
	f, ok := filtroNavegacion(r, "etf")
	if !ok {
		rechazarFiltroMemoria(w, r, m.Servicio)
		return
	}
	if m.Servicio == nil {
		errorNavegacionMemoria(w, entrada.ErrDisponibleMemoria)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var result any
	var err error
	if ap := r.PathValue("apartado"); ap != "" {
		result, err = m.Servicio.ComposicionMemoria(ctx, actorMemoria(r), f, entrada.Apartado(ap))
	} else {
		result, err = m.Servicio.ConsultarMemoria(ctx, actorMemoria(r), f)
	}
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

func errorNavegacionMemoria(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrEntradaSPAF) {
		responderError(w, http.StatusBadRequest, "estadistica.memoria_filtro_invalido")
		return
	}
	errorMemoria(w, err)
}

func idiomaCifrasValido(idioma string) bool {
	ok, _ := regexp.MatchString(`^[a-z]{2,3}(-[A-Z]{2})?$`, idioma)
	return ok
}
func rechazarFiltroMemoria(w http.ResponseWriter, r *http.Request, servicio interface {
	RechazarPeticionMemoria(context.Context, entrada.ActorMemoria) error
}) {
	if servicio == nil || servicio.RechazarPeticionMemoria(r.Context(), actorMemoria(r)) != nil {
		errorMemoria(w, entrada.ErrDisponibleMemoria)
		return
	}
	errorNavegacionMemoria(w, entrada.ErrEntradaSPAF)
}
