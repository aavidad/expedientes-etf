package http

// ModuloInstalacion aporta handlers, permisos y pantallas de su capacidad.
type ModuloInstalacion struct{ Configuracion ConfiguracionInstalacion }

func (m ModuloInstalacion) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarInstalacion(mux, m.Configuracion) }, pantalla("/admin/paquetes", "paquetes.titulo", "administracion"), pantalla("/admin/idiomas", "paquetes.idiomas", "administracion"), pantalla("/admin/temas", "paquetes.temas", "administracion")), "/admin/paquetes")
}
