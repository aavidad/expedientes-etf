package documento_test

import (
	"context"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"os"
	"strings"
	"testing"
)

func TestAnexo6PlantillasMarcadoresSustituidos(t *testing.T) {
	relleno := d.Relleno{Campos: map[string]string{"numero": "06-501-26", "fecha_documento": "13/10/2026", "fecha_entrada_sscc": "", "entidad_derivante": "", "servicio_destino": "Atención sanitaria", "equipo": "ETF Valle Lecrín", "profesional": "", "telefono_etf": "", "correo_etf": "", "direccion": "Calle del Laurel 18, Alhama de Granada", "estructura": "", "vivienda": "", "ingresos": "", "situacion_laboral": "", "situacion_economica": "", "tramo_ingresos": ""}, Filas: map[string][]map[string]string{"contactos": {{"nombre": "Lucía", "apellidos": "Ortega Salas", "telefono": "", "correo": ""}}, "menores": {{"nombre": "Nerea Ortega Salas", "nacimiento": "07/05/2016", "escolarizacion": "", "centro": ""}}, "adultos": {{"papel": "Madre", "nombre": "Lucía Ortega Salas", "parentesco": "Madre", "edad": "", "estudios": "", "otros_datos": ""}}}}
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			b, err := os.ReadFile("../../../../idiomas/" + idioma + "/plantillas/salida/anexo6/v1.0.0.docx")
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := (word.Motor{}).Generar(context.Background(), b, relleno)
			if err != nil {
				t.Fatal(err)
			}
			xml := xmlDocumento(t, resultado)
			if !strings.Contains(xml, "Nerea Ortega Salas") || !strings.Contains(xml, "06-501-26") || strings.Contains(xml, "{{") || strings.Contains(xml, "Haga clic") || strings.Contains(xml, "Elija un elemento") {
				t.Fatal("relleno incompleto")
			}
			t.Log("DOCX-OK: plantilla abierta, datos sustituidos y apartados narrativos vacíos")
		})
	}
}
