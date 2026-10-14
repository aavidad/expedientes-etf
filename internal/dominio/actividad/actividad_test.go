package actividad

import (
	"errors"
	"testing"
)

func TestOtrosExigeTextoYComisionNoAdmiteOrganismo(t *testing.T) {
	for _, texto := range []string{"", " \t\n", "\u00a0"} {
		c := Cambio{Fecha: "2026-02-14", Organismo: "OTROS", OrganismoOtros: texto}
		if !errors.Is(c.Validar(Informe, true), ErrOtros) {
			t.Fatalf("Otros sin contenido: %q", texto)
		}
	}
	if err := (Cambio{Fecha: "2026-02-14", Organismo: "OTROS", OrganismoOtros: "Centro de atención familiar de Guadix"}).Validar(Informe, true); err != nil {
		t.Fatal(err)
	}
	for _, c := range []Cambio{{Fecha: "2026-02-30"}, {Fecha: "2026-02-14", Organismo: "SPAF"}, {Fecha: "2026-02-14", VersionEsperada: 1}} {
		if !errors.Is(c.Validar(Comision, true), ErrDatos) {
			t.Fatalf("comisión inválida: %+v", c)
		}
	}
}

func TestPermisosPorPerfil(t *testing.T) {
	for _, perfil := range []string{"coordinacion", "profesional_etf", "consulta", "administracion", "sistema"} {
		a := Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", EquipoID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{perfil}}
		if permitido := a.Autorizar(false) == nil; permitido != (perfil == "coordinacion" || perfil == "profesional_etf") {
			t.Fatal(perfil, "lectura")
		}
		if permitido := a.Autorizar(true) == nil; permitido != (perfil == "profesional_etf") {
			t.Fatal(perfil, "escritura")
		}
	}
	if (Actor{}).Autorizar(false) == nil {
		t.Fatal("actor vacío")
	}
}
