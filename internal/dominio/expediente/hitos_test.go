package expediente

import (
	"errors"
	"testing"
	"time"
)

func TestHitosFechasEstadosYSubprogramas(t *testing.T) {
	hoy := time.Date(2026, 10, 13, 10, 0, 0, 0, time.FixedZone("Madrid", 7200))
	d := SituacionHitos{Estado: Aceptada, Subprograma: "RIESGO_SOCIAL", FechaAceptacion: "2026-02-03", Version: 2}
	for _, c := range []struct {
		nombre, tipo, fecha, estado, subprograma string
		esperado                                 error
	}{
		{"firma día aceptación", "consentimiento", "2026-02-03", Aceptada, "RIESGO_SOCIAL", nil},
		{"firma hoy", "consentimiento", "2026-10-13", Aceptada, "REUNIFICACION", nil},
		{"antes aceptación", "consentimiento", "2026-02-02", Aceptada, "RIESGO_SOCIAL", ErrHitoFecha},
		{"futuro", "consentimiento", "2026-10-14", Aceptada, "RIESGO_SOCIAL", ErrHitoFecha},
		{"fecha imposible", "consentimiento", "2026-02-30", Aceptada, "RIESGO_SOCIAL", ErrHitoFecha},
		{"firma ya confirmada", "consentimiento", "2026-10-13", EnIntervencion, "RIESGO_SOCIAL", ErrEstado},
		{"anexo riesgo", "anexo7", "2026-04-03", EnIntervencion, "RIESGO_SOCIAL", nil},
		{"anexo reunificación", "anexo8", "2026-04-03", Aceptada, "REUNIFICACION", nil},
		{"anexo otro subprograma", "anexo8", "2026-04-03", Aceptada, "RIESGO_SOCIAL", ErrSubprograma},
		{"sin aceptar", "anexo7", "2026-04-03", DerivacionRecibida, "RIESGO_SOCIAL", ErrEstado},
		{"tipo ajeno", "aceptacion", "2026-04-03", Aceptada, "RIESGO_SOCIAL", ErrHitoTipo},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			d.Estado, d.Subprograma = c.estado, c.subprograma
			err := (ConfirmacionHito{Tipo: c.tipo, Fecha: c.fecha, Version: 2}).Validar(d, hoy)
			if !errors.Is(err, c.esperado) {
				t.Fatalf("%v; esperado %v", err, c.esperado)
			}
		})
	}
	// El límite usa el día civil del reloj; UTC todavía está en el día anterior.
	medianoche := time.Date(2026, 10, 14, 0, 30, 0, 0, time.FixedZone("Madrid", 7200))
	d.Estado = Aceptada
	if err := (ConfirmacionHito{Tipo: "consentimiento", Fecha: "2026-10-14", Version: 2}).Validar(d, medianoche); err != nil {
		t.Fatal(err)
	}
}

func TestHitosPermisos(t *testing.T) {
	usuario, equipo := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	for _, c := range []struct {
		perfil, equipo string
		permitido      bool
	}{
		{"coordinacion", "", true}, {"profesional_etf", equipo, true}, {"profesional_etf", "", false}, {"consulta", equipo, false}, {"administracion", equipo, false}, {"sistema", equipo, false},
	} {
		a := Actor{UsuarioID: usuario, EquipoID: c.equipo, Perfiles: []string{c.perfil}}
		if (a.AutorizarHito() == nil) != c.permitido {
			t.Fatal(c)
		}
	}
}
