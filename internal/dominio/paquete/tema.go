package paquete

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

const TemaPredeterminado = "expedientes-etf"
const VersionApp = "0.1.0"

type Tema struct {
	LicenciasFuentes map[string]string `json:"licencias_fuentes,omitempty"`
	Identificador    string            `json:"identificador"`
	Nombre           map[string]string `json:"nombre"`
	Version          string            `json:"version"`
	Autor            string            `json:"autor"`
	Licencia         string            `json:"licencia"`
	VersionMinimaApp string            `json:"version_minima_app"`
	Tipo             string            `json:"tipo"`
	Fuentes          map[string]string `json:"fuentes,omitempty"`
}

type Recurso struct {
	Contenido []byte
	Tipo      string
}

var rutaFuente = regexp.MustCompile(`^fuentes/[a-zA-Z0-9_-]+\.woff2$`)
var nombreTema = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var claveToken = regexp.MustCompile(`^[a-z]+\.[a-z0-9-]+$`)
var color = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var medida = regexp.MustCompile(`^(0|[0-9]{1,3}(\.[0-9]{1,2})?(px|rem|em))( (0|[0-9]{1,3}(\.[0-9]{1,2})?(px|rem|em)))?$`)
var fuente = regexp.MustCompile(`^[a-zA-Z0-9 ',_-]{1,160}$`)
var peso = regexp.MustCompile(`^[1-9]00$`)

func NombreTemaValido(nombre string) bool { return nombreTema.MatchString(nombre) }

func versionCompatible(minima string) bool {
	if !VersionValida(minima) {
		return false
	}
	a, b := strings.Split(minima, "."), strings.Split(VersionApp, ".")
	for i := range a {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x != y {
			return x < y
		}
	}
	return true
}

func (t Tema) Valido() bool {
	if !NombreTemaValido(t.Identificador) || !VersionValida(t.Version) || !versionCompatible(t.VersionMinimaApp) ||
		len(t.Autor) == 0 || len(t.Autor) > 128 || len(t.Licencia) == 0 || len(t.Licencia) > 128 ||
		len(t.Nombre) == 0 || len(t.Nombre) > 32 || (t.Tipo != "claro" && t.Tipo != "oscuro" && t.Tipo != "alto-contraste") {
		return false
	}
	for idioma, nombre := range t.Nombre {
		if !CodigoIdiomaValido(idioma) || len(nombre) == 0 || len(nombre) > 128 {
			return false
		}
	}
	for clave, ruta := range t.Fuentes {
		if len(t.LicenciasFuentes[clave]) == 0 || len(t.LicenciasFuentes[clave]) > 32768 {
			return false
		}
		if (clave != "texto" && clave != "titulo") || !rutaFuente.MatchString(ruta) {
			return false
		}
	}
	return true
}

var tokensNecesarios = []string{
	"color.fondo",
	"color.superficie",
	"color.superficie-suave",
	"color.superficie-neutra",
	"color.borde",
	"color.borde-suave",
	"color.borde-fila",
	"color.borde-control",
	"color.texto",
	"color.texto-secundario",
	"color.texto-apagado",
	"color.icono",
	"color.acento",
	"color.acento-fuerte",
	"color.acento-suave",
	"color.sobre-acento",
	"color.calculado-fondo",
	"color.calculado-texto",
	"color.info-borde",
	"color.aviso-fondo",
	"color.aviso-borde",
	"color.aviso-borde-campo",
	"color.aviso-texto",
	"color.aviso-texto-cuerpo",
	"color.aviso-pildora",
	"color.destacado-fondo",
	"color.destacado-texto",
	"color.neutro-texto",
	"color.peligro",
	"color.foco",
	"fuente.texto",
	"fuente.titulo",
	"texto.titulo-pagina",
	"texto.cifra",
	"texto.titulo-seccion",
	"texto.marca",
	"texto.titulo-lateral",
	"texto.normal",
	"texto.etiqueta",
	"texto.meta",
	"peso.normal",
	"peso.medio",
	"peso.seminegrita",
	"peso.negrita",
	"espacio.1",
	"espacio.2",
	"espacio.3",
	"espacio.4",
	"espacio.5",
	"espacio.6",
	"espacio.7",
	"espacio.8",
	"espacio.9",
	"espacio.10",
	"espacio.etiqueta",
	"espacio.tarjeta",
	"espacio.entre-tarjetas",
	"espacio.pagina",
	"alto.control",
	"alto.boton-decision",
	"alto.cabecera",
	"ancho.menu",
	"ancho.lateral",
	"ancho.buscador",
	"tamano.avatar",
	"tamano.paso",
	"tamano.marca",
	"tamano.icono",
	"tamano.casilla",
	"radio.control",
	"radio.boton",
	"radio.aviso",
	"radio.tarjeta",
	"radio.pildora",
	"borde.normal",
	"borde.marcado",
	"borde.actual",
	"sombra.tarjeta",
	"movimiento.duracion",
	"color.peligro-fondo",
}

func TokenValido(clave, valor string) bool {
	if !claveToken.MatchString(clave) {
		return false
	}
	conocido := false
	for _, nombre := range tokensNecesarios {
		if clave == nombre {
			conocido = true
			break
		}
	}
	if !conocido {
		return false
	}
	switch strings.SplitN(clave, ".", 2)[0] {
	case "color":
		return color.MatchString(valor)
	case "fuente":
		return fuente.MatchString(valor) && strings.Count(valor, "'")%2 == 0
	case "peso":
		return peso.MatchString(valor)
	case "texto", "espacio", "alto", "ancho", "tamano", "radio", "borde":
		return medida.MatchString(valor)
	case "sombra":
		return valor == "none"
	case "movimiento":
		return valor == "0ms"
	}
	return false
}

func luminancia(valor string) float64 {
	n, _ := strconv.ParseUint(strings.TrimPrefix(valor, "#"), 16, 24)
	canales := []float64{float64(n>>16) / 255, float64((n>>8)&255) / 255, float64(n&255) / 255}
	for i, c := range canales {
		if c <= 0.04045 {
			canales[i] = c / 12.92
		} else {
			canales[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return canales[0]*0.2126 + canales[1]*0.7152 + canales[2]*0.0722
}
func Contraste(a, b string) float64 {
	x, y := luminancia(a), luminancia(b)
	if x < y {
		x, y = y, x
	}
	return (x + 0.05) / (y + 0.05)
}

// TokensValidos comprueba las parejas semánticas; nunca redondea antes de decidir.
func TokensValidos(tokens map[string]string, tipo string) bool {
	for _, clave := range tokensNecesarios {
		if !TokenValido(clave, tokens[clave]) {
			return false
		}
	}
	for clave, valor := range tokens {
		if !TokenValido(clave, valor) {
			return false
		}
	}
	minimo := 4.5
	controles := 3.0
	if tipo == "alto-contraste" {
		minimo = 7
		controles = 4.5
	}
	pares := [][2]string{
		{"color.texto", "color.fondo"}, {"color.texto", "color.superficie"}, {"color.texto", "color.superficie-suave"},
		{"color.texto-secundario", "color.fondo"}, {"color.texto-secundario", "color.superficie"}, {"color.texto-secundario", "color.superficie-suave"},
		{"color.texto-apagado", "color.superficie"}, {"color.sobre-acento", "color.acento"}, {"color.sobre-acento", "color.acento-fuerte"},
		{"color.acento", "color.fondo"}, {"color.acento", "color.superficie"},
		{"color.acento-fuerte", "color.acento-suave"}, {"color.calculado-texto", "color.calculado-fondo"},
		{"color.aviso-texto", "color.aviso-fondo"}, {"color.aviso-texto", "color.aviso-pildora"},
		{"color.aviso-texto-cuerpo", "color.aviso-fondo"}, {"color.destacado-texto", "color.destacado-fondo"},
		{"color.neutro-texto", "color.superficie-neutra"}, {"color.peligro", "color.fondo"}, {"color.peligro", "color.superficie"}, {"color.peligro", "color.peligro-fondo"},
	}
	for _, par := range pares {
		if Contraste(tokens[par[0]], tokens[par[1]]) < minimo {
			return false
		}
	}
	for _, par := range [][2]string{
		{"color.borde-control", "color.superficie"}, {"color.borde-control", "color.fondo"},
		{"color.aviso-borde-campo", "color.aviso-fondo"}, {"color.aviso-borde-campo", "color.superficie"},
		{"color.icono", "color.superficie"},
	} {
		if Contraste(tokens[par[0]], tokens[par[1]]) < controles {
			return false
		}
	}
	for _, fondo := range []string{"color.fondo", "color.superficie", "color.superficie-suave"} {
		if Contraste(tokens["color.foco"], tokens[fondo]) < 3 {
			return false
		}
	}
	if tokens["alto.control"] != "44px" && tokens["alto.control"] != "48px" {
		return false
	}
	if tipo == "alto-contraste" && tokens["borde.actual"] != "3px" {
		return false
	}
	return true
}

func TokensRequeridos() []string { return append([]string(nil), tokensNecesarios...) }
