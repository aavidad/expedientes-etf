package expediente_test

import (
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
	"time"
)

func TestCierreFechasSegundaPersonaYRestauracion(t *testing.T) {
	ahora := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	s := d.SituacionCierre{Estado: d.Aceptada, PropuestaID: "propuesta", PropuestaPor: "proponente", Motivo: "SIN_CONSENTIMIENTO", AperturaFecha: "2026-02-03"}
	for _, fecha := range []string{"", "2026-02-30", "2026-02-02", "2026-10-14"} {
		if err := (d.PeticionCierre{Version: 2, Fecha: fecha}).Validar(s, "coordinacion", ahora); !errors.Is(err, d.ErrCierre) {
			t.Fatal(fecha, err)
		}
	}
	p := d.PeticionCierre{Version: 2, Fecha: "2026-02-03"}
	if err := p.Validar(s, "proponente", ahora); !errors.Is(err, d.ErrDobleIntervencion) {
		t.Fatal(err)
	}
	if err := p.Validar(s, "coordinacion", ahora); err != nil {
		t.Fatal(err)
	}
	s.Estado = d.Cerrada
	s.UltimoCierre = "2026-03-01"
	for _, p := range []d.PeticionReapertura{{Version: 3, Fecha: "2026-02-28", Motivo: "Retoma atención."}, {Version: 3, Fecha: "2026-03-01", Motivo: ""}, {Version: 3, Fecha: "2026-10-14", Motivo: "Retoma atención."}} {
		if err := p.Validar(s, ahora); !errors.Is(err, d.ErrCierre) {
			t.Fatal(p, err)
		}
	}
	for _, c := range []struct {
		consiente, spm bool
		estado         string
	}{{false, true, d.Aceptada}, {true, true, d.EnIntervencion}, {false, false, d.DerivadaSPM}, {true, false, d.DerivadaSPM}} {
		if got := d.EstadoReapertura(c.consiente, c.spm); got != c.estado {
			t.Fatal(got, c.estado)
		}
	}
}
