package http

// Regresión F5.02: un PUT de adultos con cientos de miles de elementos vacíos
// se rechaza antes de que el servicio lo copie y serialice.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
)

type familiasLimite struct{ adultos int }

func (f *familiasLimite) EditarFamilia(_ context.Context, _ entrada.Actor, _, _ string, p entrada.CambioFamilia) (entrada.ResultadoFamilia, error) {
	if p.Adultos != nil {
		f.adultos = len(*p.Adultos)
	}
	return entrada.ResultadoFamilia{}, entrada.ErrFamilia
}
func (f *familiasLimite) ConsultarFamilia(context.Context, entrada.Actor, string) (entrada.FichaFamilia, error) {
	return entrada.FichaFamilia{}, entrada.ErrFamilia
}
func (f *familiasLimite) ConsultarHistoriaFamilia(context.Context, entrada.Actor, string, string, int) (entrada.HistoriaFamilia, error) {
	return entrada.HistoriaFamilia{}, entrada.ErrFamilia
}
func (f *familiasLimite) RechazarFamilia(context.Context, entrada.Actor) error { return nil }

func TestFamiliaRechazaAdultosMasivosAntesDelServicio(t *testing.T) {
	servicio := &familiasLimite{}
	mux := http.NewServeMux()
	mux.Handle("PUT /api/expedientes/{id}/adultos", handlerFamilia(servicio))
	cuerpo := `{"version":1,"adultos":[` + strings.Repeat("{},", 174000) + `{}]}`
	var antes, despues runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&antes)
	r := httptest.NewRequest(http.MethodPut, "/api/expedientes/0b5d7d55-5c2f-4f0c-9d43-3c3f5c6a2d10/adultos", strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "clave-prueba-limites")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	runtime.ReadMemStats(&despues)
	if asignado := despues.TotalAlloc - antes.TotalAlloc; servicio.adultos != 0 || w.Code < 400 || asignado > 32<<20 {
		t.Fatalf("estado %d, adultos en servicio %d, asignados %d bytes", w.Code, servicio.adultos, asignado)
	}
}
