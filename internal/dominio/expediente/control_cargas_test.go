package expediente

import (
	"testing"
	"time"
)

func TestPlazoCargaFechaCivil(t *testing.T) {
	p := PlazoCarga{Fecha: "2026-10-14", Version: 1}
	madrid, _ := time.LoadLocation("Europe/Madrid")
	if !p.Abierta(time.Date(2026, 10, 14, 23, 59, 59, 0, madrid)) || p.Abierta(time.Date(2026, 10, 15, 0, 0, 0, 0, madrid)) || (PlazoCarga{}).Abierta(time.Now()) {
		t.Fatal("frontera del plazo")
	}
}
func TestCoincidenciaHistoricaIdentidadesYVentana(t *testing.T) {
	menores := []Menor{{Nombre: "Lucía", Apellidos: "Serrano Ruiz"}}
	a := CasoHistorico{FechaDerivacion: "2026-01-01", Menores: []CambioMenores{{Menores: &menores}}}
	b := a
	b.FechaDerivacion = "2026-01-31"
	if !CoincidenciaHistorica(a, b, 30) {
		t.Fatal("coincidencia en el límite")
	}
	b.FechaDerivacion = "2026-02-01"
	if CoincidenciaHistorica(a, b, 30) {
		t.Fatal("fuera de ventana")
	}
	b = a
	otros := []Menor{{Nombre: "Lucía", Apellidos: "Molina Pérez"}}
	b.Menores = []CambioMenores{{Menores: &otros}}
	if CoincidenciaHistorica(a, b, 30) {
		t.Fatal("nombre común no identifica")
	}
	b = CasoHistorico{FechaDerivacion: a.FechaDerivacion}
	if CoincidenciaHistorica(b, b, 30) {
		t.Fatal("vacíos no coinciden")
	}
}

func TestCoincidenciaNoAceptadosFamiliaLeida(t *testing.T) {
	c := CasoHistorico{FechaDerivacion: "2026-03-01", Datos: DatosDerivacion{Extraccion: Extraccion{Campos: []Campo{{Codigo: "familia.nombre", Valor: "Familia Salazar Molina"}}}}}
	if !CoincidenciaHistorica(c, c, 30) {
		t.Fatal("familia sin menores no detectada")
	}
}
