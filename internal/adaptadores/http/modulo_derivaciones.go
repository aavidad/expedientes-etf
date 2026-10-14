package http

// ModuloDerivaciones aporta handlers, permisos y pantallas de su capacidad.
type ModuloDerivaciones struct{ Configuracion ConfiguracionDerivaciones }

func (m ModuloDerivaciones) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { registrarDerivaciones(mux, m.Configuracion) })
}
