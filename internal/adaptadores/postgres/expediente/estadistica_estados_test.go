package expediente

import (
	"testing"
	"time"
)

func TestEstadosCargaHistoricaSinOrdenCausal(t *testing.T) {
	recepcion := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	carga := recepcion.AddDate(0, 2, 0)
	cambios := []cambioLectura{
		{ID: "z", Estado: "aceptada", Registrado: carga},
		{ID: "a", Estado: "cerrada", Registrado: carga},
	}
	h := historiaEstados("expediente", recepcion, cambios)
	dato, _, err := h.ValorEn(carga, carga.Add(time.Hour))
	if err != nil || dato.Conocido {
		t.Fatalf("el empate ambiguo debe quedar sin dato: %+v, %v", dato, err)
	}
	posterior := carga.Add(time.Hour)
	cambios = append(cambios, cambioLectura{ID: "b", Estado: "aceptada", Registrado: posterior})
	h = historiaEstados("expediente", recepcion, cambios)
	dato, _, err = h.ValorEn(posterior, posterior.Add(time.Hour))
	if err != nil || !dato.Conocido || dato.Valor != "aceptada" {
		t.Fatalf("el cambio posterior conocido debe contar: %+v, %v", dato, err)
	}
}
