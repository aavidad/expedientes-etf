package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"net/http"
	"net/http/httptest"
	"testing"
)

type controlLecturasPrueba struct {
	entrada.ControlCargas
	consultas int
}

func (s *controlLecturasPrueba) ConsultarPlazoCarga(context.Context, entrada.Actor) (entrada.PlazoCarga, error) {
	s.consultas++
	return entrada.PlazoCarga{Fecha: "2026-12-31", Version: 1}, nil
}
func (s *controlLecturasPrueba) ConsultarLotesCarga(context.Context, entrada.Actor, int) ([]entrada.LoteCarga, error) {
	s.consultas++
	return []entrada.LoteCarga{}, nil
}
func TestControlCargasGETYHEADSonLecturas(t *testing.T) {
	s := &controlLecturasPrueba{}
	rutas := (ModuloControlCargas{Servicio: s}).Rutas()
	for _, ruta := range rutas {
		if ruta.Patron != "/api/carga-2026/plazo" && ruta.Patron != "/api/carga-2026/lotes" {
			continue
		}
		if ruta.CSRF {
			continue
		}
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			url := ruta.Patron
			if ruta.Patron == "/api/carga-2026/lotes" {
				url += "?desplazamiento=100"
			}
			antes := s.consultas
			w := httptest.NewRecorder()
			ruta.Handler.ServeHTTP(w, httptest.NewRequest(metodo, url, nil))
			if w.Code != http.StatusOK || s.consultas != antes+1 {
				t.Fatalf("%s %s: %d, consultas=%d", metodo, url, w.Code, s.consultas)
			}
		}
	}
}
