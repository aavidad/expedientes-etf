package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"strings"
	"time"
)

var _ entrada.PropuestasCierre = (*Servicio)(nil)

func (s *Servicio) GuardarFinalizacion(ctx context.Context, a d.Actor, id, clave string, p d.Finalizacion) (d.PropuestaCierre, error) {
	return s.cambiarCierre(ctx, a, id, clave, "finalizacion", p, p.Version)
}
func (s *Servicio) ProponerCierre(ctx context.Context, a d.Actor, id, clave string, p d.VersionCierre) (d.PropuestaCierre, error) {
	return s.cambiarCierre(ctx, a, id, clave, "proponer-cierre", p, p.Version)
}
func (s *Servicio) RetirarPropuestaCierre(ctx context.Context, a d.Actor, id, clave string, p d.VersionCierre) (d.PropuestaCierre, error) {
	return s.cambiarCierre(ctx, a, id, clave, "retirar-propuesta-cierre", p, p.Version)
}
func (s *Servicio) cambiarCierre(ctx context.Context, a d.Actor, id, clave, operacion string, p any, version int) (recibo d.PropuestaCierre, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			recibo = d.PropuestaCierre{}
			fallo = d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarPropuesta(); err != nil {
		return recibo, err
	}
	if !d.IDValido(id) || version < 1 {
		return recibo, d.ErrPeticion
	}
	if !d.ClaveValida(clave) {
		return recibo, d.ErrClave
	}
	if s.d.Propuestas == nil || s.d.MotivosCierre == nil || s.d.Aceptaciones == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return recibo, d.ErrNoDisponible
	}
	b, err := json.Marshal(p)
	if err != nil {
		return recibo, d.ErrPeticion
	}
	defer clear(b)
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id + "/" + operacion
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.Propuestas.RepeticionCierre(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			if r.EquipoID != a.EquipoID {
				return d.ErrAcceso
			}
			recibo = r
			return nil
		}
		ficha, err := s.d.Aceptaciones.Bloquear(ctx, id)
		if err != nil {
			return err
		}
		// Incluso una persona con ambos perfiles sólo propone para su equipo.
		if ficha.EquipoID == nil || *ficha.EquipoID != a.EquipoID {
			return d.ErrAcceso
		}
		if ficha.Version != version {
			return d.Conflicto{VersionActual: ficha.Version}
		}
		if ficha.Estado != d.Aceptada && ficha.Estado != "en_intervencion" && ficha.Estado != d.DerivadaSPM {
			return d.ErrEstado
		}
		fichaValidacion := ficha
		if ficha.Estado == d.DerivadaSPM {
			if s.d.Cierres == nil {
				return d.ErrNoDisponible
			}
			situacion, err := s.d.Cierres.BloquearCierre(ctx, id)
			if err != nil {
				return err
			}
			fichaValidacion.Estado = d.Aceptada
			if situacion.Consentimiento {
				fichaValidacion.Estado = d.EnIntervencion
			}
		}
		pendiente, err := s.d.Propuestas.Pendiente(ctx, id)
		if err != nil {
			return err
		}
		recibo = d.PropuestaCierre{ExpedienteID: id, EquipoID: a.EquipoID, PropuestaPor: a.UsuarioID, PropuestaEn: ahora, Version: version + 1, Estado: "borrador"}
		if ficha.Numero != nil {
			recibo.Numero = *ficha.Numero
		}
		if operacion == "retirar-propuesta-cierre" {
			if !pendiente {
				return d.ErrEstado
			}
			recibo, err = s.d.Propuestas.Retirar(ctx, id, a.UsuarioID, ahora)
			if err != nil {
				return err
			}
			recibo.Version = version + 1
		} else {
			if pendiente {
				return d.ErrEstado
			}
			var final *d.FinalizacionGuardada
			if operacion == "finalizacion" {
				peticion := p.(d.Finalizacion)
				catalogo, err := s.d.MotivosCierre.CierreVigente(ctx, ahora)
				if err != nil {
					return err
				}
				for _, m := range catalogo.Motivos {
					if m.Codigo == peticion.Motivo {
						if err := peticion.Validar(fichaValidacion, m.AdmiteTexto, m.SinConsentimiento); err != nil {
							return err
						}
						final = &d.FinalizacionGuardada{Finalizacion: peticion, CatalogoVersionID: catalogo.VersionID, MotivoActa: m.Acta, MotivoSPAF: m.SPAF}
						break
					}
				}
				if final == nil {
					return d.ErrFinalizacion
				}
				if err := s.d.Propuestas.GuardarFinalizacion(ctx, id, a.EquipoID, *final, a.UsuarioID, ahora); err != nil {
					return err
				}
			} else {
				final, err = s.d.Propuestas.LeerFinalizacion(ctx, id)
				if err != nil {
					return err
				}
				if final == nil {
					return d.ErrFinalizacion
				}
				// La edición sólo ocurre mediante GuardarFinalizacion, que validó el catálogo.
				if fichaValidacion.Estado == d.Aceptada && !final.SinConsentimiento {
					return d.ErrConsentimientoCierre
				}
				recibo.Estado = "pendiente"
				recibo.Motivo = final.Motivo
				recibo, err = s.d.Propuestas.Proponer(ctx, recibo)
				if err != nil {
					return err
				}
			}
			recibo.Motivo = final.Motivo
		}
		// Proponer y retirar cambian versión, nunca estado ni fecha de un hito.
		if operacion != "finalizacion" {
			final, err := s.d.Propuestas.LeerFinalizacion(ctx, id)
			if err != nil {
				return err
			}
			if final == nil {
				return d.ErrFinalizacion
			}
			if err := s.d.Propuestas.GuardarFinalizacion(ctx, id, a.EquipoID, *final, a.UsuarioID, ahora); err != nil {
				return err
			}
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente."+strings.ReplaceAll(operacion, "-", "_"), id, "ok")); err != nil {
			return err
		}
		return s.d.Propuestas.RecordarCierre(ctx, a.UsuarioID, clave, ruta, huella, recibo, ahora)
	})
	if err != nil {
		var conflicto d.Conflicto
		if errors.As(err, &conflicto) || errors.Is(err, d.ErrFinalizacion) || errors.Is(err, d.ErrConsentimientoCierre) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrEstado) || errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrIdempotencia) {
			return d.PropuestaCierre{}, err
		}
		return d.PropuestaCierre{}, d.ErrNoDisponible
	}
	return recibo, nil
}
func (s *Servicio) RechazarPropuesta(ctx context.Context, a d.Actor, clave string) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	if a.UsuarioID != "" && !d.IDValido(a.UsuarioID) {
		a = d.Actor{}
	}
	resultado := "error"
	if clave == d.ErrAcceso.Error() {
		resultado = "denegado"
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.propuesta_cierre_rechazada", "", resultado))
	return err
}
func (s *Servicio) ListarPropuestasCierre(ctx context.Context, a d.Actor, limite, desplazamiento int) (filas []d.PropuestaCierre, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			filas = nil
			fallo = d.ErrNoDisponible
		}
	}()
	if err := a.Autorizar(); err != nil {
		return nil, err
	}
	if limite < 1 || limite > 200 || desplazamiento < 0 {
		return nil, d.ErrPeticion
	}
	if s.d.Propuestas == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return nil, d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		filas, err = s.d.Propuestas.ListarPropuestas(ctx, limite, desplazamiento)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.consultar_propuestas_cierre", "", "ok"))
		return err
	})
	if err != nil {
		return nil, d.ErrNoDisponible
	}
	return filas, nil
}

func (s *Servicio) ConsultarFinalizacion(ctx context.Context, a d.Actor, id string) (final d.FinalizacionGuardada, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			final = d.FinalizacionGuardada{}
			fallo = d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarConsulta(); err != nil {
		return final, err
	}
	if !d.IDValido(id) {
		return final, d.ErrPeticion
	}
	if s.d.Repositorio == nil || s.d.Propuestas == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return final, d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		ficha, err := s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		f, err := s.d.Propuestas.LeerFinalizacion(ctx, id)
		if err != nil {
			return err
		}
		if f != nil {
			final = *f
		}
		final.Version = ficha.Version
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.consultar_finalizacion", id, "ok"))
		return err
	})
	if err != nil {
		if errors.Is(err, d.ErrNoEncontrada) {
			return d.FinalizacionGuardada{}, err
		}
		return d.FinalizacionGuardada{}, d.ErrNoDisponible
	}
	return final, nil
}
