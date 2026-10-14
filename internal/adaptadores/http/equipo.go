package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"net/http"
	"strconv"
	"time"
)

type ModuloEquipo struct{ Servicio entrada.BandejaEquipo }

func (m ModuloEquipo) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancelar()
			if m.Servicio == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			acceso := identidad.AccesoActual(ctx)
			a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
			for _, p := range acceso.Usuario.Perfiles {
				a.Perfiles = append(a.Perfiles, string(p))
			}
			if idioma := r.PathValue("idioma"); idioma != "" {
				etiquetas, err := m.Servicio.EtiquetasEquipo(ctx, a, idioma)
				if err != nil {
					errorAceptacion(w, err)
					return
				}
				responderJSON(w, http.StatusOK, etiquetas)
				return
			}
			if id := r.PathValue("id"); id != "" {
				f, err := m.Servicio.AbrirFichaEquipo(ctx, a, id)
				if err != nil {
					errorAceptacion(w, err)
					return
				}
				responderJSON(w, http.StatusOK, f)
				return
			}
			q := r.URL.Query()
			f := entrada.FiltroEquipo{Busqueda: q.Get("q"), Estado: q.Get("estado"), Subprograma: q.Get("subprograma"), Fase: q.Get("fase"), Limite: 50}
			for clave, destino := range map[string]*int{"limite": &f.Limite, "desplazamiento": &f.Desplazamiento} {
				if valor := q.Get(clave); valor != "" {
					n, err := strconv.Atoi(valor)
					if err != nil {
						*destino = -1
					} else {
						*destino = n
					}
				}
			}
			p, err := m.Servicio.ListarEquipo(ctx, a, f)
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, p)
		})
		for _, patron := range []string{"/api/equipo/expedientes", "/api/equipo/expedientes/{id}", "/api/equipo/etiquetas/{idioma}"} {
			regla := lectura(patron, "profesional_etf", "coordinacion")
			regla.ExigeAmbito = true
			mux.declarar(h, regla)
		}
	}, pantalla("/equipo", "equipo.titulo", "profesional_etf", "coordinacion"), pantalla("/equipo/expediente", "", "profesional_etf", "coordinacion")), "/equipo")
}
