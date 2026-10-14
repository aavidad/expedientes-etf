package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	paquete "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type Cambio = dominio.Cambio
type Conflicto = dominio.Conflicto

var ErrEdicion = dominio.ErrEdicion
var ErrPermiso = dominio.ErrPermiso

type FirmanteVista struct {
	dominio.Firmante
	CargoTextos map[string]string `json:"cargo_textos"`
}
type Administracion struct {
	Datos     dominio.Datos   `json:"datos"`
	Firmantes []FirmanteVista `json:"firmantes"`
	paquete.Catalogos
}
type Editor interface {
	Consultar(context.Context, Actor, string) (Administracion, error)
	Guardar(context.Context, Actor, string, Cambio) (Administracion, error)
}
