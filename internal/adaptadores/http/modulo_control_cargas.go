package http

import (
	"context"
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

type ModuloControlCargas struct{ Servicio entrada.ControlCargas }

func (m ModuloControlCargas) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if m.Servicio == nil {
				errorCargaHistorica(w, entrada.ErrNoDisponible)
				return
			}
			lectura := r.Method == http.MethodGet || r.Method == http.MethodHead
			q := r.URL.Query()
			desplazamiento := 0
			if len(q) != 0 && lectura && r.URL.Path == "/api/carga-2026/lotes" && len(q) == 1 && len(q["desplazamiento"]) == 1 {
				var e error
				desplazamiento, e = strconv.Atoi(q.Get("desplazamiento"))
				if e != nil || desplazamiento < 0 || desplazamiento > 100000 {
					errorCargaHistorica(w, entrada.ErrPeticion)
					return
				}
			} else if len(q) != 0 {
				errorCargaHistorica(w, entrada.ErrPeticion)
				return
			}
			acceso := actorActividad(r)
			a := entrada.Actor{UsuarioID: acceso.UsuarioID, EquipoID: acceso.EquipoID, Perfiles: acceso.Perfiles}
			if lectura {
				if r.URL.Path == "/api/carga-2026/lotes" {
					v, err := m.Servicio.ConsultarLotesCarga(ctx, a, desplazamiento)
					if err != nil {
						errorCargaHistorica(w, err)
						return
					}
					responderJSON(w, 200, v)
					return
				}
				v, err := m.Servicio.ConsultarPlazoCarga(ctx, a)
				if err != nil {
					errorCargaHistorica(w, err)
					return
				}
				responderJSON(w, 200, v)
				return
			}
			tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || tipo != "application/json" {
				errorCargaHistorica(w, entrada.ErrPeticion)
				return
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
			dec.DisallowUnknownFields()
			if r.PathValue("lote") != "" {
				var p struct {
					Confirmar bool `json:"confirmar"`
				}
				if dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF || !p.Confirmar {
					errorCargaHistorica(w, entrada.ErrControlCarga)
					return
				}
				if err = m.Servicio.AnularLoteCarga(ctx, a, r.PathValue("lote")); err != nil {
					errorCargaHistorica(w, err)
					return
				}
				responderJSON(w, 200, struct {
					Anulado bool `json:"anulado"`
				}{true})
				return
			}
			var p entrada.PlazoCarga
			if dec.Decode(&p) != nil || dec.Decode(new(any)) != io.EOF {
				errorCargaHistorica(w, entrada.ErrControlCarga)
				return
			}
			v, err := m.Servicio.FijarPlazoCarga(ctx, a, p)
			if err != nil {
				errorCargaHistorica(w, err)
				return
			}
			responderJSON(w, 200, v)
		})
		mux.declarar(h, lectura("/api/carga-2026/plazo", "administracion", "coordinacion", "profesional_etf"),
			escritura("/api/carga-2026/plazo", []string{http.MethodPut}, "administracion"),
			lectura("/api/carga-2026/lotes", "coordinacion", "profesional_etf"),
			escritura("/api/carga-2026/lotes/{lote}/anular", []string{http.MethodPost}, "coordinacion"))
	}, pantalla("/admin/carga-historica", "carga.control_titulo", "administracion"), pantalla("/coordinacion/carga-lotes", "carga.lotes_titulo", "coordinacion"))
}
