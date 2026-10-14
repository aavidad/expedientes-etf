package plataforma

import (
	"net"
	"regexp"
	"strconv"
)

// Configuracion sólo contiene las opciones consumidas por F0.04.
// Los conectores y sus secretos se incorporarán con sus respectivas entregas.
type Configuracion struct {
	HTTPDireccion string
	IdiomasRuta   string
	TemasRuta     string
	Idioma        string
	Tema          string
	Avisos        []string
}

// LeerConfiguracion recibe la lectura del entorno para poder probar sin secretos.
// No registra valores: los avisos contienen exclusivamente nombres de variables.
func LeerConfiguracion(leer func(string) string) Configuracion {
	valor := func(nombre, reserva string) string {
		if v := leer(nombre); v != "" {
			return v
		}
		return reserva
	}
	c := Configuracion{
		HTTPDireccion: valor("ETF_HTTP_DIRECCION", "127.0.0.1:8080"),
		IdiomasRuta:   valor("ETF_IDIOMAS_RUTA", "idiomas"),
		TemasRuta:     valor("ETF_TEMAS_RUTA", "temas"),
		Idioma:        valor("ETF_IDIOMA", "es"),
		Tema:          valor("ETF_TEMA", "expedientes-etf"),
	}
	host, puerto, err := net.SplitHostPort(c.HTTPDireccion)
	numero, errPuerto := strconv.Atoi(puerto)
	if err != nil || errPuerto != nil || numero < 1 || numero > 65535 || (host != "" && host != "localhost" && net.ParseIP(host) == nil) {
		c.HTTPDireccion = "127.0.0.1:8080"
		c.Avisos = append(c.Avisos, "ETF_HTTP_DIRECCION")
	}
	identificador := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	if !identificador.MatchString(c.Idioma) {
		c.Idioma = "es"
		c.Avisos = append(c.Avisos, "ETF_IDIOMA")
	}
	if !identificador.MatchString(c.Tema) {
		c.Tema = "expedientes-etf"
		c.Avisos = append(c.Avisos, "ETF_TEMA")
	}
	return c
}
