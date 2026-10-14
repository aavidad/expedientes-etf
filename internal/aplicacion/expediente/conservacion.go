package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type ServicioConservacion struct {
	Repositorio   salida.Conservaciones
	Catalogo      catalogo.Repositorio
	Defecto       salida.PlazoConservacion
	Reloj         salida.Reloj
	Transacciones plataforma.Transacciones
	Registro      registro.Registro
}

var _ entrada.Conservaciones = (*ServicioConservacion)(nil)

// Marcar se ejecuta dentro de la transacción que cierra o consulta los plazos.
func (s *ServicioConservacion) Marcar(ctx context.Context, id string, ahora time.Time) error {
	v, err := s.Catalogo.Vigente(ctx, "plazos", ahora)
	if err != nil {
		return err
	}
	datos, err := s.Catalogo.Datos(ctx, v)
	if err != nil {
		return err
	}

	anios := 0
	for _, p := range datos.Plazos {
		if p.Clave == "conservacion" {
			if p.Unidad != "anos" || p.Valor < 1 || p.Valor > 200 {
				return d.ErrConservacion
			}
			anios = p.Valor
		}
	}
	if anios == 0 {
		anios, err = s.Defecto.Anios(ctx)
		if err != nil {
			return err
		}
	}
	return s.Repositorio.MarcarConservacion(ctx, id, anios, v.ID, ahora)
}
func (s *ServicioConservacion) disponible() bool {
	return s.Repositorio != nil && s.Catalogo != nil && s.Defecto != nil && s.Reloj != nil && s.Transacciones != nil && s.Registro != nil
}
func (s *ServicioConservacion) fallo(ctx context.Context, a d.Actor, err error) error {
	if err == nil {
		return nil
	}
	if !s.disponible() {
		return d.ErrNoDisponible
	}
	_, fallo := s.Registro.Confirmar(ctx, auditoria.Datos{OcurridoEn: s.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "conservacion.rechazada", Entidad: "expediente", Resultado: "denegado"})
	if fallo != nil {
		return d.ErrNoDisponible
	}
	var c d.Conflicto
	if errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrConservacion) || errors.As(err, &c) {
		return err
	}
	return d.ErrNoDisponible
}
func (s *ServicioConservacion) PendientesConservacion(ctx context.Context, a d.Actor) (r []d.Conservacion, err error) {
	if err = a.AutorizarConservacion(); err != nil {
		return nil, s.fallo(ctx, a, err)
	}
	if !s.disponible() {
		return nil, d.ErrNoDisponible
	}
	ahora := s.Reloj.Ahora()
	err = s.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if err := s.Marcar(ctx, "", ahora); err != nil {
			return err
		}
		r, err = s.Repositorio.ListarConservacion(ctx, ahora)
		return err
	})
	if err != nil {
		return nil, s.fallo(ctx, a, err)
	}
	return r, nil
}
func (s *ServicioConservacion) ConfirmarConservacion(ctx context.Context, a d.Actor, id string, version int, anonimizar bool) (r d.Conservacion, err error) {
	if err = a.AutorizarConservacion(); err != nil {
		return r, s.fallo(ctx, a, err)
	}
	if !d.IDValido(id) || version < 1 {
		return r, s.fallo(ctx, a, d.ErrConservacion)
	}
	if !s.disponible() {
		return r, d.ErrNoDisponible
	}
	ahora := s.Reloj.Ahora()
	err = s.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if err := s.Marcar(ctx, id, ahora); err != nil {
			return err
		}
		r, err = s.Repositorio.ConfirmarConservacion(ctx, id, version, anonimizar, ahora)
		return err
	})
	if err != nil {
		return d.Conservacion{}, s.fallo(ctx, a, err)
	}
	return r, nil
}
