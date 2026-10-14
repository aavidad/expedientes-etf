package expediente

import "testing"

func TestSPMCierreOrdinario(t *testing.T) {
	for _, c := range []struct {
		nombre   string
		salida   *SalidaSPM
		esperado bool
	}{
		{"sin_derivacion", nil, true}, {"a_la_espera", &SalidaSPM{Respuesta: SPMAlaEspera}, false},
		{"inicio_conocido", &SalidaSPM{Respuesta: SPMInicioConocido, FechaRespuesta: "2026-10-13"}, true},
		{"medida_no_necesaria", &SalidaSPM{Respuesta: SPMMedidaNoNecesaria, FechaRespuesta: "2026-10-13"}, true},
		{"inicio_sin_fecha", &SalidaSPM{Respuesta: SPMInicioConocido}, false}, {"forzado_no_es_respuesta", &SalidaSPM{Respuesta: SPMAlaEspera, ForzadoID: "77777777-7777-4777-8777-777777777777"}, false},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			if PermiteCierreOrdinarioSPM(c.salida) != c.esperado {
				t.Fatal(c)
			}
		})
	}
}
func TestSPMFechaYRespuesta(t *testing.T) {
	s := &SalidaSPM{FechaDerivacion: "2026-10-12", Respuesta: SPMAlaEspera}
	for _, p := range []PeticionSPM{{Version: 1, Respuesta: SPMInicioConocido}, {Version: 1, Respuesta: SPMInicioConocido, Fecha: "2026-10-11"}, {Version: 1, Respuesta: SPMAlaEspera, Fecha: "2026-10-13"}, {Version: 1, Respuesta: "otra", Fecha: "2026-10-13"}} {
		if p.ValidarSPM("responder", s) == nil {
			t.Fatal(p)
		}
	}
	if (PeticionSPM{Version: 1, Respuesta: SPMAlaEspera}).ValidarSPM("responder", s) != nil {
		t.Fatal("espera sin fecha")
	}
	if (PeticionSPM{Version: 1, Motivo: "   "}).ValidarSPM("forzar", s) == nil {
		t.Fatal("forzado sin motivo")
	}
}

func TestForzadoSPMPendienteSinFechaConfirmada(t *testing.T) {
	s := &SalidaSPM{Respuesta: SPMAlaEspera}
	p := PeticionSPM{Version: 1, Motivo: "La coordinación confirma la necesidad del cierre excepcional."}
	if err := p.ValidarSPM("forzar", s); err != nil {
		t.Fatal(err)
	}
	if PermiteCierreOrdinarioSPM(s) {
		t.Fatal("forzado no habilita cierre ordinario")
	}
}
