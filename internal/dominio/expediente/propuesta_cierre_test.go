package expediente_test

import (
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
)

func TestFinalizacionSinIntervencion(t *testing.T) {
	for _, c := range []struct {
		estado       string
		sin, permite bool
		err          error
	}{
		{d.Aceptada, true, true, nil}, {d.Aceptada, false, true, d.ErrConsentimientoCierre}, {d.Aceptada, true, false, d.ErrConsentimientoCierre},
		{"en_intervencion", false, false, nil}, {"en_intervencion", true, true, d.ErrConsentimientoCierre}, {"cerrada", true, true, d.ErrEstado}, {"derivada_spm", false, false, d.ErrEstado},
	} {
		p := d.Finalizacion{Version: 2, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: c.sin}
		if err := p.Validar(d.Derivacion{Estado: c.estado}, false, c.permite); !errors.Is(err, c.err) {
			t.Fatal(c, err)
		}
	}
	p := d.Finalizacion{Version: 1, Motivo: "OTROS"}
	if err := p.Validar(d.Derivacion{Estado: "en_intervencion"}, true, false); !errors.Is(err, d.ErrFinalizacion) {
		t.Fatal(err)
	}
	p.MotivoTexto = "La familia continúa con apoyo comunitario."
	if err := p.Validar(d.Derivacion{Estado: "en_intervencion"}, true, false); err != nil {
		t.Fatal(err)
	}
}
