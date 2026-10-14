package entrada

import (
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

var ErrActa = errors.New("carga.actas.tipo_invalido")

type ResultadoActa struct {
	Contenido d.Extraccion `json:"contenido"`
	Antivirus string       `json:"antivirus"`
}
type Actas interface {
	LeerActa(context.Context, Actor, string, string, Subida) (ResultadoActa, error)
	RechazarActa(context.Context, Actor) error
}
