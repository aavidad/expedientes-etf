package http

// Regresión F5.02: un cuerpo de 2 MiB con cientos de miles de casos vacíos no
// debe materializarse antes de comprobar la cota de casos.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
)

type cargasLimite struct{ casos int }

func (s *cargasLimite) ValidarCargaHistorica(_ context.Context, _ entrada.Actor, p entrada.CargaHistorica) (entrada.ValidacionCargaHistorica, error) {
	s.casos = len(p.Casos)
	return entrada.ValidacionCargaHistorica{}, entrada.ErrCargaHistorica
}
func (s *cargasLimite) CargarHistoricos(_ context.Context, _ entrada.Actor, _, _ string, p entrada.CargaHistorica) (entrada.ResultadoCargaHistorica, error) {
	s.casos = len(p.Casos)
	return entrada.ResultadoCargaHistorica{}, entrada.ErrCargaHistorica
}
func (s *cargasLimite) RechazarCargaHistorica(context.Context, entrada.Actor) error { return nil }

func TestCargaHistoricaRechazaCasosMasivosSinMaterializarlos(t *testing.T) {
	for _, ruta := range []string{"/api/carga-2026", "/api/expedientes/historicos"} {
		cuerpo := []byte(`{"equipo_id":"x","casos":[` + strings.Repeat("{},", 2<<20/3-20) + `{}]}`)
		mux := nuevoMuxHTTP()
		servicio := &cargasLimite{}
		registrarCargasHistoricas(mux, servicio)
		var antes, despues runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&antes)
		r := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "clave-prueba")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		runtime.ReadMemStats(&despues)
		asignado := despues.TotalAlloc - antes.TotalAlloc
		if w.Code != http.StatusBadRequest || servicio.casos != 0 || asignado > 64<<20 {
			t.Fatalf("%s: estado %d, casos vistos %d, asignados %d bytes", ruta, w.Code, servicio.casos, asignado)
		}
	}
}

func TestCargaHistoricaAdmiteCienCasos(t *testing.T) {
	mux := nuevoMuxHTTP()
	servicio := &cargasLimite{}
	registrarCargasHistoricas(mux, servicio)
	cuerpo := `{"equipo_id":"x","casos":[` + strings.Repeat(`{"episodios":[{}],"familia":[],"menores":[]},`, 99) + `{}]}`
	r := httptest.NewRequest(http.MethodPost, "/api/carga-2026", strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if servicio.casos != 100 {
		t.Fatalf("100 casos deben llegar al servicio: %d (estado %d)", servicio.casos, w.Code)
	}
}
