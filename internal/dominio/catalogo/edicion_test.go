package catalogo

import (
	"errors"
	"testing"
	"time"
)

func TestEdicionExigeTextoYReferenciaDeAmbosIdiomas(t *testing.T) {
	c := Cambio{VersionEsperada: 1, VigenteDesde: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), Opcion: &Opcion{Lista: "pronostico", Codigo: "FAVORABLE", Orden: 0, Activa: true}, Textos: map[string]string{"es": "Favorable", "en": "Favourable"}, ReferenciasEsperadas: map[string]int{"es": 0, "en": 0}}
	if err := c.Validar("listas"); err != nil {
		t.Fatal(err)
	}
	c.ReferenciasEsperadas = map[string]int{"es": 0, "fr": 0}
	if !errors.Is(c.Validar("listas"), ErrEdicion) {
		t.Fatal("referencia ausente admitida")
	}
	c.ReferenciasEsperadas = map[string]int{"es": 0, "en": 0}
	c.Textos["en"] = " "
	if !errors.Is(c.Validar("listas"), ErrEdicion) {
		t.Fatal("texto ausente admitido")
	}
	c.Textos["en"] = "Favourable"
	c.Plazo = &Plazo{Clave: "consentimiento", Valor: 30, Unidad: "dias"}
	if !errors.Is(c.Validar("listas"), ErrEdicion) {
		t.Fatal("dos reglas en una petición")
	}
}
func TestEdicionDeniegaOtrosPerfiles(t *testing.T) {
	for _, p := range []string{"equipo", "consulta", "coordinacion", "sistema", ""} {
		if !errors.Is(Autorizar(Actor{UsuarioID: "persona", Perfiles: []string{p}}), ErrPermiso) {
			t.Fatal(p)
		}
	}
}
func TestTramoLimiteYFirmanteVigencia(t *testing.T) {
	hasta := 10.0
	c := Cambio{VigenteDesde: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), Tramo: &Tramo{Tipo: "edad_menor", Desde: 10, Hasta: &hasta, Unidad: "anos"}}
	if !errors.Is(c.Validar("tramos"), ErrEdicion) {
		t.Fatal("tramo vacío")
	}
}
