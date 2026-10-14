package expediente

import (
	"errors"
	"slices"
)

var ErrConservacion = errors.New("conservacion.no_permitida")

type Conservacion struct {
	ExpedienteID   string `json:"expediente_id"`
	Numero         string `json:"numero"`
	Cierre         string `json:"cierre"`
	Fin            string `json:"fin_conservacion"`
	Anios          int    `json:"anios"`
	Version        int    `json:"version"`
	Estado         string `json:"estado"`
	PuedeConfirmar bool   `json:"puede_confirmar"`
}

func (a Actor) AutorizarConservacion() error {
	if !IDValido(a.UsuarioID) || !slices.Contains(a.Perfiles, "administracion") {
		return ErrAcceso
	}
	return nil
}
