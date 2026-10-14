package expediente

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

var _ entrada.TrasladoSeguimiento = (*Servicio)(nil)

func (s *Servicio) reglaSeguimiento(ctx context.Context, fecha time.Time) (d.ReglaSeguimiento, error) {
	v, err := s.d.Catalogo.Vigente(ctx, "plazos", fecha)
	if err != nil {
		return d.ReglaSeguimiento{}, err
	}
	c, err := s.d.Catalogo.Datos(ctx, v)
	if err != nil {
		return d.ReglaSeguimiento{}, err
	}
	r := d.ReglaSeguimiento{ID: v.ID, AntelacionDias: -1}
	for _, p := range c.Plazos {
		if p.Clave == "seguimiento_tras_cese" && p.Unidad == "meses" {
			r.Meses = p.Valor
		}
		if p.Clave == "aviso_fin_seguimiento" && p.Unidad == "dias" {
			r.AntelacionDias = p.Valor
		}
	}
	if r.Meses < 1 || r.Meses > 1200 || r.AntelacionDias < 0 || r.AntelacionDias > 36600 {
		return r, d.ErrSeguimiento
	}
	return r, nil
}
func (s *Servicio) Seguimiento(ctx context.Context, a d.Actor, id, clave, accion string, p d.PeticionSeguimiento) (r d.Seguimiento, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			r, fallo = d.Seguimiento{}, d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarConsulta(); err != nil {
		return r, err
	}
	if accion == "reabrir" {
		if err := a.Autorizar(); err != nil {
			return r, err
		}
	}
	if !d.IDValido(id) {
		return r, d.ErrPeticion
	}
	if !d.ClaveValida(clave) {
		return r, d.ErrClave
	}
	if s.d.Repositorio == nil || s.d.Seguimientos == nil || s.d.Aceptaciones == nil || s.d.Cierres == nil || s.d.Catalogo == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return r, d.ErrNoDisponible
	}
	h, err := huellaActo(p)
	if err != nil {
		return r, err
	}
	ruta := "/api/expedientes/" + id + "/seguimiento/" + accion
	ahora := s.d.Reloj.Ahora().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		// RLS y autorización actuales preceden a cualquier recibo conservado.
		if _, err := s.d.Repositorio.Obtener(ctx, id); err != nil {
			return err
		}
		previa, repetida, err := s.d.Seguimientos.RepetirSeguimiento(ctx, a.UsuarioID, clave, ruta, h, ahora)
		if err != nil {
			return err
		}
		if repetida {
			r = previa
			return nil
		}
		ficha, err := s.d.Aceptaciones.Bloquear(ctx, id)
		if err != nil {
			return err
		}
		actual, err := s.d.Seguimientos.ObtenerSeguimiento(ctx, id, true)
		if err != nil {
			return err
		}
		version := ficha.Version
		if actual != nil {
			version = actual.Version
		}
		if version != p.Version {
			return d.Conflicto{VersionActual: version}
		}
		if ficha.Aceptacion == nil || ficha.Aceptacion.Subprograma != "REUNIFICACION" || (ficha.Estado != d.Cerrada && ficha.Estado != d.SeguimientoTrasCese) {
			return d.ErrEstado
		}
		if err = p.Validar(accion, actual, ahora); err != nil {
			return err
		}
		if accion == "iniciar" {
			regla, err := s.reglaSeguimiento(ctx, ahora)
			if err != nil {
				return err
			}
			cierre, err := s.d.Cierres.BloquearCierre(ctx, id)
			if err != nil {
				return err
			}
			if cierre.UltimoCierre == "" {
				return d.ErrEstado
			}
			if _, err := d.FinSeguimiento(p.Fecha, regla.Meses); err != nil {
				return err
			}
			r, err = s.d.Seguimientos.IniciarSeguimiento(ctx, id, cierre.EquipoID, cierre.EpisodioID, p, regla, a.UsuarioID, ahora)
			if err != nil {
				return err
			}
		} else {
			var regla d.ReglaSeguimiento
			if accion == "reabrir" {
				regla, err = s.reglaSeguimiento(ctx, ahora)
				if err != nil {
					return err
				}
			}
			if accion == "terminar" {
				v, err := s.d.Catalogo.Vigente(ctx, "listas", ahora)
				if err != nil {
					return err
				}
				c, err := s.d.Catalogo.Datos(ctx, v)
				if err != nil {
					return err
				}
				valido := false
				for _, o := range c.Opciones {
					if o.Lista == "fin_seguimiento" && o.Codigo == p.Motivo && o.Activa {
						valido = true
					}
				}
				if !valido {
					return d.ErrSeguimiento
				}
			}
			r, err = s.d.Seguimientos.CambiarSeguimiento(ctx, *actual, accion, p, regla, a.UsuarioID, ahora)
			if err != nil {
				return err
			}
		}
		if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.seguimiento_"+accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Seguimientos.RecordarSeguimiento(ctx, a.UsuarioID, clave, ruta, h, r, ahora)
	})
	return r, falloTrasladoSeguimiento(err)
}
func (s *Servicio) ConsultarSeguimiento(ctx context.Context, a d.Actor, id string) (r *d.Seguimiento, fallo error) {
	if err := a.AutorizarConsulta(); err != nil {
		return nil, err
	}
	if !d.IDValido(id) {
		return nil, d.ErrPeticion
	}
	if s.d.Seguimientos == nil || s.d.Repositorio == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return nil, d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if _, err := s.d.Repositorio.Obtener(ctx, id); err != nil {
			return err
		}
		var err error
		r, err = s.d.Seguimientos.ObtenerSeguimiento(ctx, id, false)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.seguimiento_consultar", id, "ok"))
		return err
	})
	if err != nil {
		return nil, falloTrasladoSeguimiento(err)
	}
	return r, nil
}
func (s *Servicio) ConsultarAvisosSeguimiento(ctx context.Context, a d.Actor) ([]d.AvisoSeguimiento, error) {
	if err := a.AutorizarConsulta(); err != nil {
		return nil, err
	}
	if s.d.Seguimientos == nil || s.d.Catalogo == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return nil, d.ErrNoDisponible
	}
	var r []d.AvisoSeguimiento
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		regla, err := s.reglaSeguimiento(ctx, s.d.Reloj.Ahora())
		if err != nil {
			return err
		}
		r, err = s.d.Seguimientos.AvisosSeguimiento(ctx, d.FechaCivil(s.d.Reloj.Ahora()), regla.AntelacionDias)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.avisos_consultar", "", "ok"))
		return err
	})
	if err != nil {
		return nil, falloTrasladoSeguimiento(err)
	}
	return r, nil
}

// VencerSeguimientos es un acto técnico sin ruta HTTP ni identidad humana.
// PostgreSQL limita la operación a vencimientos reales; nunca acepta una fecha externa.
func (s *Servicio) VencerSeguimientos(ctx context.Context) error {
	if s.d.Seguimientos == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		ids, err := s.d.Seguimientos.VencerSeguimientos(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(d.Actor{}, "expediente.seguimiento_vencer", id, "ok")); err != nil {
				return err
			}
		}
		return nil
	})
	return falloTrasladoSeguimiento(err)
}
