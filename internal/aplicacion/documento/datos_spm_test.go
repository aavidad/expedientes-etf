package documento

import (
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"testing"
)

func TestAnexo5NoConfundeContactoDelServicioDerivanteConETF(t *testing.T) {
	a := &d.Aceptacion{Profesional: "TS", Equipo: "ETF Alhama"}
	campos := map[string][]string{"servicio.profesional": {"Marta Serrano Ruiz"}, "servicio.telefono": {"958123456"}, "servicio.correo": {"marta.serrano@servicios.invalid"}, "familia.contacto.nombre": {"Lucía"}, "familia.contacto.apellidos": {"Ortega Salas"}, "familia.contacto.telefono": {"600123456"}}
	rellenarDatosSPM(a, campos)
	for _, k := range []string{"profesional", "profesional_telefono", "profesional_email"} {
		if a.CamposSPM[k] != "" {
			t.Fatal("contacto del servicio atribuido al ETF", k)
		}
	}
	if a.FilasSPM["spm_contactos"][0]["telefono"] != "600123456" {
		t.Fatal("contacto familiar perdido")
	}
	if len(a.FilasSPM["spm_significativas"]) != 1 || a.FilasSPM["spm_significativas"][0]["nombre"] != "" {
		t.Fatal("fila opcional vacía no disponible")
	}
}
