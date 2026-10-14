package catalogo

import (
	"testing"
	"time"
)

func fecha(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestVigenciaIntervaloCivil(t *testing.T) {
	hasta := fecha(t, "2026-07-01")
	v := Version{Ambito: "listas", Estado: "publicada", VigenteDesde: fecha(t, "2026-01-01"), VigenteHasta: &hasta}
	for _, caso := range []struct {
		dia  string
		rige bool
	}{{"2025-12-31", false}, {"2026-01-01", true}, {"2026-06-30", true}, {"2026-07-01", false}, {"2026-12-31", false}} {
		if got := v.Rige(fecha(t, caso.dia)); got != caso.rige {
			t.Fatalf("%s: %v", caso.dia, got)
		}
	}
	v.VigenteHasta = nil
	if !v.Rige(fecha(t, "2040-12-31")) {
		t.Fatal("vigencia abierta")
	}
	for _, estado := range []string{"borrador", "retirada"} {
		v.Estado = estado
		if v.Rige(fecha(t, "2026-01-01")) {
			t.Fatal(estado)
		}
	}
}
func TestMemoriaUsaDiciembreNoFechaActual(t *testing.T) {
	f, err := FechaMemoria(2026)
	if err != nil || !f.Equal(fecha(t, "2026-12-31")) {
		t.Fatal(f, err)
	}
	hasta := fecha(t, "2026-07-01")
	anterior := Version{Ambito: "listas", Estado: "publicada", VigenteDesde: fecha(t, "2026-01-01"), VigenteHasta: &hasta}
	nueva := Version{Ambito: "listas", Estado: "publicada", VigenteDesde: hasta}
	if anterior.Rige(f) || !nueva.Rige(f) {
		t.Fatal("Memoria eligió versión anterior")
	}
	for _, anio := range []int{0, -1, 10000} {
		if _, err := FechaMemoria(anio); err != ErrVigencia {
			t.Fatal(anio, err)
		}
	}
}
func TestVigenciasInvalidasYZona(t *testing.T) {
	desde := fecha(t, "2026-01-01")
	igual := desde
	antes := fecha(t, "2025-12-31")
	for _, hasta := range []*time.Time{&igual, &antes} {
		v := Version{Ambito: "listas", VigenteDesde: desde, VigenteHasta: hasta}
		if v.Validar() != ErrVigencia {
			t.Fatal(v)
		}
	}
	v := Version{Ambito: "listas", Estado: "publicada", VigenteDesde: desde}
	if !v.Rige(time.Date(2026, 1, 1, 23, 59, 0, 0, time.FixedZone("UTC-8", -8*3600))) {
		t.Fatal("día civil desplazado")
	}
	if v.Rige(time.Time{}) {
		t.Fatal("fecha vacía")
	}
	v.Ambito = "arbitrario"
	if v.Validar() != ErrAmbito {
		t.Fatal("ámbito inválido")
	}
}
