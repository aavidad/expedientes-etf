package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"io"
	"net/http"
)

func atenderEdicion(w http.ResponseWriter, r *http.Request, c ConfiguracionCatalogos, ambito string) {
	if c.Edicion == nil {
		responderError(w, http.StatusServiceUnavailable, "catalogo.no_disponible")
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		a, err := c.Edicion.Consultar(r.Context(), actorCatalogos(r), ambito)
		if err != nil {
			errorCatalogo(w, err)
			return
		}
		responderJSON(w, http.StatusOK, a)
	case http.MethodPost, http.MethodPut:
		if r.Header.Get("Content-Type") != "application/json" {
			responderError(w, http.StatusUnsupportedMediaType, "catalogo.edicion_invalida")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var cambio entrada.Cambio
		if err := d.Decode(&cambio); err != nil {
			responderError(w, http.StatusBadRequest, "catalogo.edicion_invalida")
			return
		}
		if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
			responderError(w, http.StatusBadRequest, "catalogo.edicion_invalida")
			return
		}
		a, err := c.Edicion.Guardar(r.Context(), actorCatalogos(r), ambito, cambio)
		if err != nil {
			errorCatalogo(w, err)
			return
		}
		responderJSON(w, http.StatusCreated, a)
	}
}
func conflictoCatalogo(w http.ResponseWriter, err error) bool {
	var c *entrada.Conflicto
	if errors.As(err, &c) {
		responderJSON(w, http.StatusConflict, map[string]any{"error": "catalogo.conflicto", "version_actual": c.Actual, "recurso": c.Recurso})
		return true
	}
	var p *paquetes.ConflictoPublicacion
	if errors.As(err, &p) {
		responderJSON(w, http.StatusConflict, map[string]any{"error": "catalogo.conflicto", "version_actual": p.Actual, "recurso": p.Codigo})
		return true
	}
	return false
}
