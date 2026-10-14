package expediente

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

func serializarVigenciasEstadistica(t *testing.T, vigencias []vigenciaLectura) []byte {
	t.Helper()
	filas := []map[string]any{}
	for _, v := range vigencias {
		filas = append(filas, map[string]any{"id": v.ID, "vigente_desde": v.Desde.Time.Format("2006-01-02"), "registrado_en": v.Registrado, "valor": v.Valor})
	}
	b, err := json.Marshal(filas)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMenoresEstadisticaVigenciaVacioYSinDato(t *testing.T) {
	dia := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	ms := []vigenciaLectura{
		{ID: "febrero", Desde: fechaLectura{dia}, Registrado: dia, Valor: json.RawMessage(`[{"id":"menor","sexo":"M","subprograma":"REUNIFICACION","fecha_nacimiento":"2016-05-14","propuesto_atencion_temprana":false}]`)},
		{ID: "marzo", Desde: fechaLectura{dia.AddDate(0, 1, 0)}, Registrado: dia.AddDate(0, 1, 0), Valor: json.RawMessage(`[]`)},
	}
	var datos datosLectura
	if err := incorporarMenoresIntervencion(&datos, serializarVigenciasEstadistica(t, ms), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	exp := datos.Expediente
	conocido := dia.AddDate(0, 2, 0)
	menores, _, err := exp.Menores.ValorEn(dia, conocido)
	if err != nil || !menores.Conocido || len(menores.Valor) != 1 {
		t.Fatal(menores, err)
	}
	m := menores.Valor[0]
	if m.Sexo != d.Nina || m.Subprograma != d.Reunificacion || m.Nacimiento.Format("2006-01-02") != "2016-05-14" || !m.PropuestaAtencionTemprana.Conocido || m.PropuestaAtencionTemprana.Valor {
		t.Fatal(m)
	}
	vacio, _, err := exp.Menores.ValorEn(dia.AddDate(0, 1, 0), conocido)
	if err != nil || !vacio.Conocido || len(vacio.Valor) != 0 {
		t.Fatal(vacio, err)
	}
	ausente, _, err := exp.Menores.ValorEn(dia.AddDate(0, 0, -1), conocido)
	if err != nil || ausente.Conocido {
		t.Fatal(ausente, err)
	}
}

func TestCoordinacionEstadisticaDesconocidaConservaLimites(t *testing.T) {
	dia := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	h := []vigenciaLectura{
		{ID: "desconocida", Desde: fechaLectura{dia}, Registrado: dia, Valor: json.RawMessage(`{"listas_conocidas":[],"marcas":[]}`)},
		{ID: "vacia", Desde: fechaLectura{dia.AddDate(0, 1, 0)}, Registrado: dia.AddDate(0, 1, 0), Valor: json.RawMessage(`{"listas_conocidas":["coordinacion"],"marcas":[]}`)},
		{ID: "marcada", Desde: fechaLectura{dia.AddDate(0, 2, 0)}, Registrado: dia.AddDate(0, 2, 0), Valor: json.RawMessage(`{"listas_conocidas":["coordinacion"],"marcas":[{"lista":"coordinacion","codigo":"OTROS","menores_excluidos":[]}]}`)},
	}
	var datos datosLectura
	if err := incorporarMenoresIntervencion(&datos, []byte(`[]`), serializarVigenciasEstadistica(t, h)); err != nil {
		t.Fatal(err)
	}
	exp := datos.Expediente
	conocido := dia.AddDate(0, 3, 0)
	for n, revision := range h {
		dato, _, err := exp.RecursosCoordinacion.ValorEn(revision.Desde.Time, conocido)
		if err != nil || dato.Conocido != (n > 0) || len(dato.Valor) != max(0, n-1) {
			t.Fatal(n, dato, err)
		}
	}
	if len(exp.Propuestas) != 0 || len(exp.Responsables) != 0 || len(exp.Menores) != 0 {
		t.Fatal("se dedujeron datos sin fuente")
	}
	factores, _, err := exp.Factores.ValorEn(h[2].Desde.Time, conocido)
	if err != nil || !factores.Conocido || len(factores.Valor) != 1 || factores.Valor[0].Lista != "coordinacion" || factores.Valor[0].Codigo != "OTROS" {
		t.Fatal(factores, err)
	}
	serializado, err := json.Marshal(exp)
	if err != nil || strings.Contains(string(serializado), "texto_otros") {
		t.Fatal(string(serializado), err)
	}
}
