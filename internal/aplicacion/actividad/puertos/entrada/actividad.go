package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
)

type Actor = dominio.Actor
type Tipo = dominio.Tipo
type Registro = dominio.Registro
type Cambio = dominio.Cambio
type Filtro = dominio.Filtro
type Recuento = dominio.Recuento
type Organismo = dominio.Organismo
type Equipo = dominio.Equipo
type Conflicto = dominio.Conflicto

const (
	Comision = dominio.Comision
	Informe  = dominio.Informe
)

var (
	ErrDatos        = dominio.ErrDatos
	ErrOtros        = dominio.ErrOtros
	ErrPermiso      = dominio.ErrPermiso
	ErrNoEncontrado = dominio.ErrNoEncontrado
	UUIDValido      = dominio.UUIDValido
)

// ConsultaMemoria permite consumir las cifras sin depender de tablas ni HTTP.
type ConsultaMemoria interface {
	Recontar(context.Context, Actor, Filtro) (Recuento, error)
}

type Gestor interface {
	Consultar(context.Context, Actor, Tipo, Filtro) ([]Registro, error)
	Registrar(context.Context, Actor, Tipo, Cambio) (Registro, error)
	Corregir(context.Context, Actor, Tipo, string, Cambio) (Registro, error)
	Retirar(context.Context, Actor, Tipo, string, int) (Registro, error)
	Etiquetas(context.Context, Actor) (map[string]map[string]string, error)
	Equipos(context.Context, Actor) ([]Equipo, error)
	Organismos(context.Context, Actor) ([]Organismo, error)
}
