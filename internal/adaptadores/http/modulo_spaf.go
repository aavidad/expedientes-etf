package http

// ModuloSPAF aporta la lectura mensual y sus permisos.
type ModuloSPAF struct{ Configuracion ConfiguracionSPAF }

func (m ModuloSPAF) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarSPAF(mux, m.Configuracion) }, pantalla("/equipo/tabla-spaf", "spaf.titulo", "profesional_etf", "coordinacion"), pantalla("/equipo/tabla-spaf/expedientes/{id}", "", "profesional_etf", "coordinacion")), "/equipo/tabla-spaf")
}
