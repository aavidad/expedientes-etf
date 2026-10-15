package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

type ConfiguracionHoja struct {
	Servicio entrada.HojasHistoricas
	Textos   entrada.TextosHoja
	Idioma   string
}

func registrarHoja(mux *muxHTTP, c ConfiguracionHoja) {
	// Dos hojas a la vez como mucho: cada una puede ocupar 10 MB de entrada y
	// su lectura expandida en el proceso compartido.
	plazas := make(chan struct{}, 2)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancelar()
		select {
		case plazas <- struct{}{}:
			defer func() { <-plazas }()
		case <-ctx.Done():
			errorHoja(w, entrada.ErrNoDisponible)
			return
		}
		acceso := actorActividad(r)
		a := entrada.Actor{UsuarioID: acceso.UsuarioID, EquipoID: acceso.EquipoID, Perfiles: acceso.Perfiles}
		rechazar := func(err error) {
			if c.Servicio != nil {
				if fallo := c.Servicio.RechazarCargaHistorica(ctx, a); fallo != nil {
					err = fallo
				}
			}
			errorHoja(w, err)
		}
		if c.Servicio == nil || c.Textos == nil {
			errorHoja(w, entrada.ErrNoDisponible)
			return
		}
		if len(r.URL.Query()) != 0 || len(r.Header.Values("X-Equipo-Carga")) != 1 {
			rechazar(entrada.ErrPeticion)
			return
		}
		tipo, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		formato := ""
		switch tipo {
		case "text/csv":
			formato = "csv"
		case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
			formato = "xlsx"
		}
		if err != nil || formato == "" || len(parametros) > 1 || len(parametros) == 1 && strings.ToLower(parametros["charset"]) != "utf-8" {
			rechazar(entrada.ErrHojaFormato)
			return
		}
		confirmar := r.PathValue("lote") != ""
		if confirmar && (len(r.Header.Values("Idempotency-Key")) != 1 || len(r.Header.Values("X-Huella-Hoja")) != 1) {
			rechazar(entrada.ErrClave)
			return
		}
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10_000_000))
		defer clear(b)
		if err != nil {
			rechazar(entrada.ErrHojaLimites)
			return
		}
		idioma := c.Idioma
		if idioma != "en" {
			idioma = "es"
		}
		if preferido := strings.ToLower(r.Header.Get("Accept-Language")); strings.HasPrefix(preferido, "en") {
			idioma = "en"
		} else if strings.HasPrefix(preferido, "es") {
			idioma = "es"
		}
		textos, err := c.Textos.Textos(ctx, idioma)
		if err != nil {
			rechazar(entrada.ErrNoDisponible)
			return
		}
		if len(r.Header.Values("X-Confirmar-Duplicados")) > 1 || r.Header.Get("X-Confirmar-Duplicados") != "" && r.Header.Get("X-Confirmar-Duplicados") != "true" {
			rechazar(entrada.ErrPeticion)
			return
		}
		u := entrada.SubidaHoja{ConfirmarDuplicados: r.Header.Get("X-Confirmar-Duplicados") == "true", EquipoID: r.Header.Get("X-Equipo-Carga"), Formato: formato, Contenido: b}
		var informe entrada.InformeHoja
		var resultado entrada.ResultadoCargaHistorica
		if confirmar {
			resultado, informe, err = c.Servicio.ConfirmarHoja(ctx, a, r.PathValue("lote"), r.Header.Get("Idempotency-Key"), r.Header.Get("X-Huella-Hoja"), u)
		} else {
			informe, err = c.Servicio.ValidarHoja(ctx, a, u)
		}
		if err != nil {
			errorHoja(w, err)
			return
		}
		for i := range informe.Incidencias {
			mensaje, existe := textos[informe.Incidencias[i].Clave]
			if !existe || mensaje == "" {
				rechazar(entrada.ErrNoDisponible)
				return
			}
			informe.Incidencias[i].Mensaje = mensaje
		}
		if !informe.Valida {
			responderJSON(w, http.StatusUnprocessableEntity, informe)
			return
		}
		if confirmar {
			responderJSON(w, http.StatusOK, resultado)
		} else {
			responderJSON(w, http.StatusOK, informe)
		}
	})
	mux.declarar(h, escritura("/api/carga-2026/hoja", []string{http.MethodPost}, "coordinacion", "profesional_etf"), escritura("/api/carga-2026/hoja/{lote}/confirmar", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}

func errorHoja(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entrada.ErrHojaFormato):
		responderError(w, http.StatusUnprocessableEntity, "expediente.hoja_formato")
	case errors.Is(err, entrada.ErrHojaLimites):
		responderError(w, http.StatusRequestEntityTooLarge, "expediente.hoja_limites")
	case errors.Is(err, entrada.ErrHojaAntivirus):
		responderError(w, http.StatusServiceUnavailable, "expediente.hoja_antivirus")
	case errors.Is(err, entrada.ErrHojaCambiada):
		responderError(w, http.StatusConflict, "expediente.hoja_cambiada")
	default:
		errorCargaHistorica(w, err)
	}
}
