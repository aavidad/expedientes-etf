package http

// ModuloActividad aporta handlers, permisos y pantallas de su capacidad.
type ModuloActividad struct{ Configuracion ConfiguracionActividad }

func (m ModuloActividad) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarActividad(mux, m.Configuracion) }, pantalla("/equipo/comisiones", "actividad.titulo", "profesional_etf", "coordinacion")), "/equipo/comisiones")
}
