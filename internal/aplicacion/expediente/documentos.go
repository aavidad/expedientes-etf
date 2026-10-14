package expediente

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

type ConsultaDocumentos struct{ Repositorio salida.LecturaDocumentos }

var _ entrada.DatosDocumentos = ConsultaDocumentos{}

func (c ConsultaDocumentos) ParaDocumentos(ctx context.Context, a entrada.Actor, id string) (entrada.DatosParaDocumento, error) {
	if err := a.AutorizarConsulta(); err != nil {
		return d.DatosParaDocumento{}, err
	}
	if !d.IDValido(id) {
		return d.DatosParaDocumento{}, d.ErrPeticion
	}
	if c.Repositorio == nil {
		return d.DatosParaDocumento{}, d.ErrNoDisponible
	}
	datos, err := c.Repositorio.ParaDocumentos(ctx, id)
	if err != nil {
		return d.DatosParaDocumento{}, err
	}
	if a.Autorizar() != nil && (datos.EquipoID == nil || *datos.EquipoID != a.EquipoID) {
		return d.DatosParaDocumento{}, d.ErrAcceso
	}
	return datos, nil
}
