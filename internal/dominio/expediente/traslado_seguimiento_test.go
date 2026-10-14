package expediente_test

import (
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
	"time"
)

func TestSeguimientoCalendarioCivil(t *testing.T) {
	for _, c := range []struct {
		fecha string
		meses int
		fin   string
	}{
		{"2024-02-29", 12, "2025-02-28"}, {"2026-01-31", 1, "2026-02-28"}, {"2026-10-14", 12, "2027-10-14"}, {"2023-01-31", 13, "2024-02-29"},
	} {
		fin, err := d.FinSeguimiento(c.fecha, c.meses)
		if err != nil || fin != c.fin {
			t.Fatal(c, fin, err)
		}
	}
	for _, c := range []struct {
		fecha string
		meses int
	}{{"2026-02-30", 12}, {"2026-10-14", 0}, {"9999-12-31", 1}} {
		if _, err := d.FinSeguimiento(c.fecha, c.meses); !errors.Is(err, d.ErrSeguimiento) {
			t.Fatal(c, err)
		}
	}
}
func TestSeguimientoTerminacionYReapertura(t *testing.T) {
	ahora := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	s := d.Seguimiento{Version: 1, Estado: "activo", FechaCese: "2026-10-01", FechaFin: "2027-10-01"}
	p := d.PeticionSeguimiento{Version: 1, Fecha: "2026-10-14", Motivo: "CAMBIO_SITUACION"}
	if err := p.Validar("terminar", &s, ahora); err != nil {
		t.Fatal(err)
	}
	s.TramoDesde = "2026-10-14"
	p.Fecha = "2026-10-13"
	if err := p.Validar("terminar", &s, ahora); !errors.Is(err, d.ErrSeguimiento) {
		t.Fatal("fin anterior a reapertura", err)
	}
	p.Fecha = "2026-09-30"
	if err := p.Validar("terminar", &s, ahora); !errors.Is(err, d.ErrSeguimiento) {
		t.Fatal(err)
	}
	s.Estado = "terminado"
	p = d.PeticionSeguimiento{Version: 2, Motivo: "La familia vuelve a necesitar seguimiento."}
	if err := p.Validar("reabrir", &s, ahora); err != nil {
		t.Fatal(err)
	}
	if err := p.Validar("reabrir", &s, ahora.AddDate(1, 0, 0)); err != nil {
		t.Fatal(err)
	}
}
func TestTrasladoExigeOtroEquipoYMotivo(t *testing.T) {
	s := d.SituacionCierre{EquipoID: "11111111-1111-4111-8111-111111111111", Motivo: "TRASLADO"}
	p := d.PeticionTraslado{EquipoDestinoID: "22222222-2222-4222-8222-222222222222"}
	if err := p.ValidarDestino(s); err != nil {
		t.Fatal(err)
	}
	p.EquipoDestinoID = s.EquipoID
	if err := p.ValidarDestino(s); !errors.Is(err, d.ErrTraslado) {
		t.Fatal(err)
	}
	p.EquipoDestinoID = "22222222-2222-4222-8222-222222222222"
	s.Motivo = "OBJ_TOTAL"
	if err := p.ValidarDestino(s); !errors.Is(err, d.ErrTraslado) {
		t.Fatal(err)
	}
}
