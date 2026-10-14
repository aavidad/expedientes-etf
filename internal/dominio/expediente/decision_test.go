package expediente

import (
	"errors"
	"testing"
)

func TestNoAceptacionMotivoSubprogramaYEstado(t *testing.T) {
	d := Derivacion{Estado: DerivacionRecibida, Datos: DatosDerivacion{Extraccion: Extraccion{Tipo: "anexo3"}}}
	p := NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "La familia recibe atención de otro recurso."}
	valida, err := p.Validar(d)
	if err != nil || valida.Subprograma != "RIESGO_SOCIAL" {
		t.Fatal(valida, err)
	}
	for _, motivo := range []string{"", " \t\n"} {
		p.Motivo = motivo
		if _, err := p.Validar(d); !errors.Is(err, ErrMotivo) {
			t.Fatal(err)
		}
	}
	p = valida
	p.Subprograma = "REUNIFICACION"
	if _, err := p.Validar(d); !errors.Is(err, ErrSubprograma) {
		t.Fatal(err)
	}
	p = valida
	p.Fecha = "2027-02-29"
	if _, err := p.Validar(d); !errors.Is(err, ErrPeticion) {
		t.Fatal(err)
	}
	p = valida
	d.Estado = NoAceptada
	if _, err := p.Validar(d); !errors.Is(err, ErrEstado) {
		t.Fatal(err)
	}
}
