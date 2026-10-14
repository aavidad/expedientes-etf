package estadistica

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type DependenciasNarrativas struct {
	Repositorio   salida.Narrativas
	Transacciones plataforma.Transacciones
	Auditoria     audit.RegistroTransaccional
	Reloj         expediente.Reloj
}
type ServicioNarrativas struct{ d DependenciasNarrativas }

var _ entrada.Narrativas = (*ServicioNarrativas)(nil)

func NuevoNarrativas(d DependenciasNarrativas) *ServicioNarrativas { return &ServicioNarrativas{d} }
func (s *ServicioNarrativas) operar(ctx context.Context, a d.ActorMemoria, n d.Narrativas, guardar bool) (d.Narrativas, error) {
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return d.Narrativas{}, d.ErrDisponibleMemoria
	}
	filtro, fallo := d.AutorizarMemoria(a, n.FiltroMemoria)
	n.FiltroMemoria = filtro
	accion := "estadistica.narrativas_leer"
	if guardar {
		accion = "estadistica.narrativas_guardar"
		if fallo == nil {
			fallo = n.Validar()
		}
	}
	if fallo == nil && s.d.Repositorio == nil {
		fallo = d.ErrDisponibleMemoria
	}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	if !d.UUIDValidoSPAF(a.UsuarioID) {
		identidad = plataforma.Identidad{}
	}
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	var r d.Narrativas
	evento := func(resultado string) auditoria.Datos {
		usuario := identidad.UsuarioID
		if usuario == "" {
			usuario = "sistema"
		}
		return auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: identidad.Perfiles, Accion: accion, Entidad: "estadistica", EntidadID: r.ID, Resultado: resultado}
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		if fallo != nil {
			_, err := s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("denegado"))
			return err
		}
		var err error
		if guardar {
			if cierres, ok := s.d.Repositorio.(salida.Cierres); ok {
				if err := cierres.BloquearMemorias(ctx); err != nil {
					return err
				}
			}
			r, err = s.d.Repositorio.GuardarNarrativas(ctx, n, a.UsuarioID, ahora)
		} else {
			r, err = s.d.Repositorio.LeerNarrativas(ctx, filtro)
		}
		if err != nil {
			return err
		}
		_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("ok"))
		return err
	})
	if err != nil {
		var conflicto *d.ConflictoMemoria
		if errors.As(err, &conflicto) || errors.Is(err, d.ErrCerradaMemoria) {
			fallo = err
		} else {
			fallo = d.ErrDisponibleMemoria
		}
		ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelar()
		auditErr := s.d.Transacciones.Ejecutar(ctxAuditoria, identidad, func(ctx context.Context) error {
			_, err := s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("error"))
			return err
		})
		if auditErr != nil {
			return d.Narrativas{}, d.ErrDisponibleMemoria
		}
	}
	if fallo != nil {
		return d.Narrativas{}, fallo
	}
	return r, nil
}
func (s *ServicioNarrativas) LeerNarrativas(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (d.Narrativas, error) {
	return s.operar(ctx, a, d.Narrativas{FiltroMemoria: f}, false)
}
func (s *ServicioNarrativas) GuardarNarrativas(ctx context.Context, a d.ActorMemoria, n d.Narrativas) (d.Narrativas, error) {
	return s.operar(ctx, a, n, true)
}
