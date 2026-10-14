package expediente

import "testing"

func TestDatosSinDNIYEscalaFueraDeExpediente(t *testing.T) {
	e := DatosPermitidos(Extraccion{Campos: []Campo{
		{Codigo: "familia.dni", Valor: "12345678Z"},
		{Codigo: "valorame.global", Valor: "nivel_1"},
		{Codigo: "valoracion.historia", Valor: "DNI 12345678Z; la familia reside en Cúllar Vega."},
	}})
	if len(e.Campos) != 1 || e.Campos[0].Valor != "DNI ; la familia reside en Cúllar Vega." || !e.Campos[0].Dudoso {
		t.Fatal(e)
	}
}

func TestLimiteDeMenoresTambienEnPDF(t *testing.T) {
	var e Extraccion
	for i := 0; i < 15; i++ {
		e.Campos = append(e.Campos, Campo{Codigo: "menores.persona.nombre"})
	}
	if err := ValidarExtraccion(e); err != nil {
		t.Fatal(err)
	}
	e.Campos = append(e.Campos, Campo{Codigo: "menores.persona.nombre"})
	if err := ValidarExtraccion(e); err == nil {
		t.Fatal("se admiten 16 menores")
	}
}

func TestPDFReconocidoAvisaEvaluacionAunqueNoLaExtraiga(t *testing.T) {
	e := DatosPermitidos(Extraccion{Tipo: "anexo3", Version: "pdf-1"})
	if len(e.Campos) != 0 || len(e.Avisos) != 1 || e.Avisos[0] != "expediente.valorame_pendiente" {
		t.Fatal(e)
	}
}
