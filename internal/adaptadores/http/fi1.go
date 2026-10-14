package http

import "net/http"

// ModuloFichaMenoresIntervencion publica accesos directos a las pestañas de la ficha.
// Sus datos usan las rutas versionadas del módulo Expedientes.
type ModuloFichaMenoresIntervencion struct{}

func (ModuloFichaMenoresIntervencion) Rutas() []Ruta {
	var rutas []Ruta
	for _, patron := range []string{"/equipo/menores", "/equipo/intervencion"} {
		rutas = append(rutas, Ruta{ReglaRuta: ReglaRuta{Patron: patron, Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: []string{"profesional_etf", "coordinacion"}, ExigeAmbito: true, Pantalla: true, SPA: "/equipo"}})
	}
	return rutas
}
