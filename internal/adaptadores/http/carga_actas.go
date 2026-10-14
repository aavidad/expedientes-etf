package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"net/http"
	"time"
)

func registrarActasCarga(mux *muxHTTP, s entrada.Actas) {
	plazas := make(chan struct{}, 1)
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if s == nil {
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		acceso := actorActividad(r)
		a := entrada.Actor{UsuarioID: acceso.UsuarioID, EquipoID: acceso.EquipoID, Perfiles: acceso.Perfiles}
		rechazar := func(err error) {
			if e := s.RechazarActa(ctx, a); e != nil {
				err = e
			}
			errorActa(w, err)
		}
		if len(r.Header.Values("X-Equipo-Carga")) != 1 || len(r.URL.Query()) != 0 {
			rechazar(entrada.ErrPeticion)
			return
		}
		select {
		case plazas <- struct{}{}:
			defer func() { <-plazas }()
		case <-ctx.Done():
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		nombre, b, err := leerAnexo(w, r)
		if err != nil {
			rechazar(err)
			return
		}
		defer clear(b)
		datos, err := s.LeerActa(ctx, a, r.Header.Get("X-Equipo-Carga"), r.PathValue("tipo"), entrada.Subida{Nombre: nombre, Contenido: b})
		if err != nil {
			errorActa(w, err)
			return
		}
		responderJSON(w, 200, datos)
	}), escritura("/api/carga-2026/actas/leer/{tipo}", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}
func errorActa(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrActa) {
		responderError(w, 422, "carga.actas.tipo_invalido")
		return
	}
	errorDerivacion(w, err)
}
