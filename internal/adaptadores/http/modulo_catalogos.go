package http

// ModuloCatalogos aporta handlers, permisos y pantallas de su capacidad.
type ModuloCatalogos struct{ Configuracion ConfiguracionCatalogos }

func (m ModuloCatalogos) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarCatalogos(mux, m.Configuracion) }, pantalla("/admin/catalogos", "catalogo.menu_listas", "administracion"), pantalla("/admin/catalogos/listas", "", "administracion"), pantalla("/admin/reglas", "catalogo.menu_plazos", "administracion"), pantalla("/admin/firmantes", "catalogo.menu_firmantes", "administracion")), "/admin/catalogos")
}
