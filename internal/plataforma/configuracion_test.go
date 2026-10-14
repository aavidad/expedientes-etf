package plataforma

import "testing"

func TestSinConfiguracionOpcional(t *testing.T) {
	c := LeerConfiguracion(func(string) string { return "" })
	if c.HTTPDireccion != "127.0.0.1:8080" || c.Idioma != "es" || c.Tema != "expedientes-etf" || len(c.Avisos) != 0 {
		t.Fatalf("configuración: %+v", c)
	}
}

func TestConfiguracionInvalidaUsaReservaSinRegistrarValores(t *testing.T) {
	c := LeerConfiguracion(func(string) string { return "../valor-no-valido" })
	if c.HTTPDireccion != "127.0.0.1:8080" || c.Idioma != "es" || c.Tema != "expedientes-etf" {
		t.Fatalf("reserva: %+v", c)
	}
	if len(c.Avisos) != 3 {
		t.Fatalf("avisos: %v", c.Avisos)
	}
	for _, v := range c.Avisos {
		if v != "ETF_HTTP_DIRECCION" && v != "ETF_IDIOMA" && v != "ETF_TEMA" {
			t.Errorf("valor filtrado: %s", v)
		}
	}
}

func TestDireccionHTTPRechazaHostInvalido(t *testing.T) {
	c := LeerConfiguracion(func(nombre string) string {
		if nombre == "ETF_HTTP_DIRECCION" {
			return "host no valido:8080"
		}
		return ""
	})
	if c.HTTPDireccion != "127.0.0.1:8080" || len(c.Avisos) != 1 {
		t.Fatalf("configuración: %+v", c)
	}
}
