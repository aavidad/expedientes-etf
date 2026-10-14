package http

import (
	"errors"
	"net/http"
	"slices"
	"strings"
)

// ReglaRuta declara el acceso completo a un patrón HTTP.
type ReglaRuta struct {
	ClaveTitulo                                  string
	Patron                                       string
	Metodos                                      []string
	Perfiles                                     []string
	Publica, ExigeAmbito, CSRF, Pantalla, Sesion bool
	SPA                                          string
}
type reglaRuta = ReglaRuta

// Ruta reúne el handler y su regla; las pantallas reciben el handler SPA al componer.
type Ruta struct {
	ReglaRuta
	Handler http.Handler
}

// Modulo aporta sus rutas mediante composición explícita, sin init ni estado global.
type Modulo interface{ Rutas() []Ruta }

func (m *muxHTTP) declarar(handler http.Handler, reglas ...reglaRuta) {
	patrones := map[string]bool{}
	for _, regla := range reglas {
		m.rutas = append(m.rutas, Ruta{ReglaRuta: regla, Handler: handler})
		if !m.soloRutas && !patrones[regla.Patron] {
			m.Handle(regla.Patron, handler)
			patrones[regla.Patron] = true
		}
	}
}

func lectura(patron string, perfiles ...string) reglaRuta {
	return reglaRuta{Patron: patron, Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles}
}
func escritura(patron string, metodos []string, perfiles ...string) reglaRuta {
	return reglaRuta{Patron: patron, Metodos: metodos, Perfiles: perfiles, CSRF: true}
}
func pantalla(patron, clave string, perfiles ...string) Ruta {
	regla := lectura(patron, perfiles...)
	regla.Pantalla, regla.ClaveTitulo, regla.Publica, regla.SPA = true, clave, len(perfiles) == 0, patron
	return Ruta{ReglaRuta: regla}
}
func rutasDeModulo(registrar func(*muxHTTP), pantallas ...Ruta) []Ruta {
	mux := &muxHTTP{soloRutas: true}
	registrar(mux)
	return append(pantallas, mux.rutas...)
}
func reglasDeRutas(modulos []Modulo) []reglaRuta {
	var reglas []reglaRuta
	for _, modulo := range modulos {
		for _, ruta := range modulo.Rutas() {
			reglas = append(reglas, ruta.ReglaRuta)
		}
	}
	return reglas
}

// ValidarRutas rechaza registros incompletos, duplicados y patrones incompatibles.
func ValidarRutas(rutas []Ruta) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("http.registro_invalido")
		}
	}()
	if len(rutas) == 0 {
		return errors.New("http.registro_invalido")
	}
	pares := map[string]bool{}
	patrones := map[string]bool{}
	pantallas := map[string]bool{}
	mux := http.NewServeMux()
	for _, ruta := range rutas {
		if ruta.Patron == "" || len(ruta.Metodos) == 0 || ruta.Publica == (len(ruta.Perfiles) > 0) || (ruta.Handler == nil && !ruta.Pantalla) {
			return errors.New("http.registro_invalido")
		}
		for _, perfil := range ruta.Perfiles {
			if !slices.Contains([]string{"administracion", "coordinacion", "profesional_etf", "consulta"}, perfil) {
				return errors.New("http.registro_invalido")
			}
		}
		for _, metodo := range ruta.Metodos {
			if !slices.Contains([]string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, http.MethodTrace, http.MethodConnect}, metodo) || (metodo != http.MethodGet && metodo != http.MethodHead && !ruta.CSRF) {
				return errors.New("http.registro_invalido")
			}
			par := metodo + " " + ruta.Patron
			if pares[par] {
				return errors.New("http.registro_invalido")
			}
			pares[par] = true
		}
		if ruta.Pantalla {
			if pantallas[ruta.Patron] || !slices.Contains(ruta.Metodos, http.MethodGet) {
				return errors.New("http.registro_invalido")
			}
			pantallas[ruta.Patron] = true
		}
		if !patrones[ruta.Patron] {
			mux.HandleFunc(ruta.Patron, func(http.ResponseWriter, *http.Request) {})
			patrones[ruta.Patron] = true
		}
	}
	for _, ruta := range rutas {
		if ruta.SPA != "" && !pantallas[ruta.SPA] {
			return errors.New("http.registro_invalido")
		}
	}
	return nil
}

func (s *sesionHTTP) reglasRutas() []reglaRuta {
	if s.rutas != nil {
		return s.rutas
	}
	return tablaRutas()
}

func asociarSPA(rutas []Ruta, spa string) []Ruta {
	for i := range rutas {
		if strings.HasPrefix(rutas[i].Patron, "/api/") {
			rutas[i].SPA = spa
		}
	}
	return rutas
}
