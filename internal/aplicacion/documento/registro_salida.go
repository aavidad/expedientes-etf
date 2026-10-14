package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func (s *Servicio) AnotarRegistroSalida(ctx context.Context, a d.Actor, p d.SolicitudRegistroSalida) (d.RegistroSalida, error) {
	rechazar := func(err error) (d.RegistroSalida, error) {
		if fallo := s.Rechazar(ctx, a, err.Error()); fallo != nil {
			return d.RegistroSalida{}, fallo
		}
		return d.RegistroSalida{}, err
	}
	if err := a.AutorizarEntrada(); err != nil {
		return rechazar(err)
	}
	if err := p.Validar(); err != nil {
		return rechazar(err)
	}
	if s.d.RepositorioRegistro == nil || s.d.RegistroSalida == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return rechazar(d.ErrNoDisponible)
	}
	huella, err := d.HuellaRegistro(p)
	if err != nil {
		return rechazar(d.ErrPeticion)
	}
	var resultado d.RegistroSalida
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, r, err := s.d.RepositorioRegistro.LeerRegistro(ctx, p.EmisionID)
		if err != nil {
			return err
		}
		if e.ID == "" {
			return d.ErrAcceso
		}
		if !d.EsOficio(e.Tipo) {
			return d.ErrRegistroOficio
		}
		if err := a.AutorizarEquipo(e.EquipoID); err != nil {
			return err
		}
		anterior, repetida, err := s.d.RepositorioRegistro.PeticionRegistro(ctx, a, p, huella)
		if err != nil {
			return err
		}
		if repetida {
			resultado = anterior
		} else {
			if p.Version != r.Version {
				return &d.ConflictoRegistro{VersionActual: r.Version}
			}
			resultado, err = s.d.RegistroSalida.Anotar(ctx, r, p, a, s.d.Reloj.Ahora())
			if err != nil {
				return err
			}
			if resultado.EmisionID != r.EmisionID || resultado.Version != r.Version+1 || resultado.Numero != p.Numero || resultado.Fecha != p.Fecha || resultado.AnotadoPor != a.UsuarioID || resultado.Estado != "anotado" {
				return d.ErrNoDisponible
			}
			if err := s.d.RepositorioRegistro.GuardarRegistro(ctx, resultado, r.Version); err != nil {
				return err
			}
			if err := s.d.RepositorioRegistro.RecordarRegistro(ctx, a, p, huella, resultado); err != nil {
				return err
			}
		}

		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.anotar_registro_salida", e, "ok"))
		return err
	})
	if err != nil {
		return rechazar(err)
	}
	return resultado, nil
}
