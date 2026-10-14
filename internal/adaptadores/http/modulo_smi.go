package http

// ModuloSMI aporta handlers, permisos y pantallas de su capacidad.
type ModuloSMI struct{ Configuracion ConfiguracionSMI }

func (m ModuloSMI) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarSMI(mux, m.Configuracion) }, pantalla("/coordinacion/smi", "smi.titulo", "coordinacion")), "/coordinacion/smi")
}
