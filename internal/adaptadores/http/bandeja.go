package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"net/http"
	"strconv"
)

type ModuloBandeja struct{ Servicio entrada.Bandeja }

func (m ModuloBandeja) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(m.atender), lectura("/api/coordinacion/bandeja", "coordinacion"))
		// POST transporta los términos de búsqueda sin incluir nombres en la URL.
		mux.declarar(http.HandlerFunc(m.atender), escritura("/api/coordinacion/bandeja/buscar", []string{http.MethodPost}, "coordinacion"))
	}, pantalla("/coordinacion", "bandeja.titulo", "coordinacion")), "/coordinacion")
}
func (m ModuloBandeja) atender(w http.ResponseWriter, r *http.Request) {
	if m.Servicio == nil {
		responderError(w, 503, "expediente.no_disponible")
		return
	}
	actor, err := actorDerivaciones(r)
	if err != nil {
		responderError(w, 403, "expediente.acceso_denegado")
		return
	}
	p := entrada.FiltroBandeja{Limite: 20}
	if r.Method == http.MethodPost {
		if r.Header.Get("Content-Type") != "application/json" {
			responderError(w, 415, "expediente.peticion_invalida")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&p) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) {
			responderError(w, 400, "expediente.peticion_invalida")
			return
		}
	} else {
		q := r.URL.Query()
		for k, v := range q {
			if len(v) != 1 || (k != "estado" && k != "equipo_id" && k != "municipio_id" && k != "desde" && k != "hasta" && k != "limite" && k != "desplazamiento") {
				responderError(w, 400, "expediente.peticion_invalida")
				return
			}
		}
		p.Estado = q.Get("estado")
		p.EquipoID = q.Get("equipo_id")
		p.MunicipioID = q.Get("municipio_id")
		p.Desde = q.Get("desde")
		p.Hasta = q.Get("hasta")
		for k, destino := range map[string]*int{"limite": &p.Limite, "desplazamiento": &p.Desplazamiento} {
			if q.Has(k) {
				n, err := strconv.Atoi(q.Get(k))
				if err != nil {
					responderError(w, 400, "expediente.peticion_invalida")
					return
				}
				*destino = n
			}
		}
	}
	resultado, err := m.Servicio.ConsultarBandeja(r.Context(), actor, p)
	if err != nil {
		estado := 503
		if errors.Is(err, entrada.ErrPeticion) {
			estado = 400
		}
		if errors.Is(err, entrada.ErrAcceso) {
			estado = 403
		}
		clave := "expediente.no_disponible"
		if estado == 400 {
			clave = "expediente.peticion_invalida"
		}
		if estado == 403 {
			clave = "expediente.acceso_denegado"
		}
		responderError(w, estado, clave)
		return
	}
	responderJSON(w, 200, resultado)
}
