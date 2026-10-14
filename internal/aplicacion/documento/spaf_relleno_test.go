package documento

import (
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"strings"
	"testing"
	"time"
)

func TestRellenoSPAF14ColumnasTotalesYDesconocidos(t *testing.T) {
	fecha := time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)
	tabla := est.TablaSPAF{FiltroSPAF: est.FiltroSPAF{Anio: 2026, Mes: 1}, Resumen: est.ResumenSPAF{Altas: 2, Bajas: 1, Casos: 3, Menores: 6}, Filas: []est.FilaSPAF{{Numero: "06-01-26", NumeroMenores: est.Dato[int]{Conocido: true, Valor: 1}, Aceptacion: est.Dato[time.Time]{Conocido: true, Valor: fecha}, DatosSPAF: est.DatosSPAF{Menores: est.Dato[[]est.MenorSPAF]{Conocido: true, Valor: []est.MenorSPAF{{Nombre: "Lucía", Apellidos: "Serrano Ruiz", Nacimiento: est.Dato[time.Time]{Conocido: true, Valor: fecha}}}}, Municipio: est.Dato[string]{Conocido: true, Valor: "18013"}, ZTS: est.Dato[string]{Valor: "dato residual"}, Instrumentos: est.Dato[[]string]{Conocido: true, Valor: []string{"ENTREVISTA"}}}}}}
	textos := map[string]string{"documento.spaf.sin_dato": "Sin dato", "documento.spaf.anexo": "Anexo %d", "documento.spaf.acta": "Acta", "documento.spaf.fecha_formato": "02/01/2006"}
	nombres := map[string]string{"municipio.18013": "Alhama de Granada", "instrumento.ENTREVISTA": "Entrevista"}
	r, err := rellenarSPAF(tabla, "06", textos, nombres)
	if err != nil {
		t.Fatal(err)
	}
	fila := r.Filas["casos_spaf"][0]
	if len(fila) != 14 || fila["menores"] != "Serrano Ruiz, Lucía / 07/01/2026" || fila["municipio_zts"] != "Alhama de Granada / Sin dato" || fila["instrumentos"] != "Entrevista" || r.Campos["menores"] != "6" || r.Campos["casos"] != "3" {
		t.Fatal(r)
	}
	for _, v := range fila {
		if strings.Contains(v, "residual") {
			t.Fatal("valor desconocido expuesto")
		}
	}
	tabla.Filas[0].Menores.Valor = append(tabla.Filas[0].Menores.Valor, est.MenorSPAF{Nombre: "Mateo", Apellidos: "Serrano Ruiz", CentroProteccion: est.Dato[string]{Conocido: true, Valor: "Centro Sierra Nevada"}})
	tabla.Filas[0].Guardadores = est.Dato[[]est.GuardadorSPAF]{Conocido: true, Valor: []est.GuardadorSPAF{{Nombre: "Clara Ruiz Torres", Parentesco: "MADRE"}}}
	nombres["papel_parental.MADRE"] = "Madre"
	mixto, err := rellenarSPAF(tabla, "06", textos, nombres)
	if err != nil || mixto.Filas["casos_spaf"][0]["guardadores"] != "Clara Ruiz Torres / Madre; Centro Sierra Nevada" {
		t.Fatal("guardadores del menor en hogar omitidos", mixto, err)
	}
	for i := range tabla.Filas[0].Menores.Valor {
		tabla.Filas[0].Menores.Valor[i].CentroProteccion = est.Dato[string]{Conocido: true, Valor: ""}
	}
	sinCentro, err := rellenarSPAF(tabla, "06", textos, nombres)
	if err != nil || sinCentro.Filas["casos_spaf"][0]["guardadores"] != "Clara Ruiz Torres / Madre" {
		t.Fatal("centro vacío oculta guardadores", sinCentro, err)
	}
	delete(nombres, "instrumento.ENTREVISTA")
	if _, err := rellenarSPAF(tabla, "06", textos, nombres); err != d.ErrNoDisponible {
		t.Fatal("código sin traducción servido", err)
	}
	tabla.Filas = nil
	r, err = rellenarSPAF(tabla, "06", textos, nombres)
	if err != nil || len(r.Filas["casos_spaf"]) != 0 {
		t.Fatal(err, r)
	}
}
