package http

// ModuloDocumentos aporta handlers, permisos y pantallas de su capacidad.
type ModuloDocumentos struct{ Configuracion ConfiguracionDocumentos }

func (m ModuloDocumentos) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { registrarDocumentos(mux, m.Configuracion) })
}
