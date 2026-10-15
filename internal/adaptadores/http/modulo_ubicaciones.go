package http

import (
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"net/http"
)

// ModuloUbicaciones comparte la publicación transaccional de los catálogos.
type ModuloUbicaciones struct{ Editor entrada.Editor }

func (m ModuloUbicaciones) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atenderEdicion(w, r, ConfiguracionCatalogos{Edicion: m.Editor}, "asignacion_municipios")
		}), lectura("/api/admin/ubicaciones", "administracion"), escritura("/api/admin/ubicaciones", []string{http.MethodPost, http.MethodPut}, "administracion"))
	}, pantalla("/admin/ubicaciones", "ubicaciones.titulo", "administracion")), "/admin/ubicaciones")
}
