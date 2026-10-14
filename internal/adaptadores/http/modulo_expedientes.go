package http

// ModuloExpedientes aporta handlers, permisos y pantallas de su capacidad.
type ModuloExpedientes struct{ Configuracion ConfiguracionAceptaciones }

func (m ModuloExpedientes) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { registrarAceptaciones(mux, m.Configuracion) })
}
