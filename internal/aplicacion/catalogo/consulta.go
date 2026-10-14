package catalogo

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type Consulta struct {
	repositorio   salida.Repositorio
	transacciones plataforma.Transacciones
}

var _ entrada.Catalogos = (*Consulta)(nil)

func NuevaConsulta(repo salida.Repositorio, tx plataforma.Transacciones) *Consulta {
	return &Consulta{repo, tx}
}

func (c *Consulta) ejecutar(ctx context.Context, actor entrada.Actor, operacion func(context.Context) error) error {
	return c.transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, operacion)
}

func validar(ambito string, fecha time.Time) error {
	if !dominio.AmbitoValido(ambito) {
		return dominio.ErrAmbito
	}
	if fecha.IsZero() || fecha.Year() < 1 || fecha.Year() > 9999 {
		return dominio.ErrVigencia
	}
	return nil
}

func (c *Consulta) Vigente(ctx context.Context, actor entrada.Actor, ambito string, fecha time.Time) (dominio.Version, error) {
	if err := validar(ambito, fecha); err != nil {
		return dominio.Version{}, err
	}
	var version dominio.Version
	err := c.ejecutar(ctx, actor, func(ctx context.Context) error {
		var err error
		version, err = c.repositorio.Vigente(ctx, ambito, dominio.FechaCivil(fecha))
		return err
	})
	if err != nil {
		return dominio.Version{}, err
	}
	return version, nil
}
func (c *Consulta) Memoria(ctx context.Context, actor entrada.Actor, ambito string, anio int) (dominio.Version, error) {
	fecha, err := dominio.FechaMemoria(anio)
	if err != nil {
		return dominio.Version{}, err
	}
	return c.Vigente(ctx, actor, ambito, fecha)
}
func (c *Consulta) Versiones(ctx context.Context, actor entrada.Actor, ambito string) ([]dominio.Version, error) {
	if !dominio.AmbitoValido(ambito) {
		return nil, dominio.ErrAmbito
	}
	var versiones []dominio.Version
	err := c.ejecutar(ctx, actor, func(ctx context.Context) error {
		var err error
		versiones, err = c.repositorio.Versiones(ctx, ambito)
		return err
	})
	if err != nil {
		return nil, err
	}
	return versiones, nil
}
func (c *Consulta) Datos(ctx context.Context, actor entrada.Actor, ambito string, fecha time.Time) (dominio.Datos, error) {
	if err := validar(ambito, fecha); err != nil {
		return dominio.Datos{}, err
	}
	var datos dominio.Datos
	err := c.ejecutar(ctx, actor, func(ctx context.Context) error {
		version, err := c.repositorio.Vigente(ctx, ambito, dominio.FechaCivil(fecha))
		if err != nil {
			return err
		}
		datos, err = c.repositorio.Datos(ctx, version)
		return err
	})
	if err != nil {
		return dominio.Datos{}, err
	}
	return datos, nil
}
