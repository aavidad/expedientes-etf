package http

// ModuloSalidas publica las pantallas; expediente y documento conservan sus API.
type ModuloSalidas struct{}

func (ModuloSalidas) Rutas() []Ruta {
	return []Ruta{pantalla("/coordinacion/cierres", "salidas.cierres", "coordinacion"), pantalla("/coordinacion/expedientes/{id}/salidas", "", "coordinacion")}
}
