package http

// ModuloUsuarios aporta handlers, permisos y pantallas de su capacidad.
type ModuloUsuarios struct{ Configuracion ConfiguracionUsuarios }

func (m ModuloUsuarios) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarUsuarios(mux, m.Configuracion) }, pantalla("/admin/usuarios", "usuarios.titulo", "administracion")), "/admin/usuarios")
}
