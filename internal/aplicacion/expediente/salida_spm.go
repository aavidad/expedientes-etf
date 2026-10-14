package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"time"
)

var _ entrada.SalidasSPM = (*Servicio)(nil)

func (s *Servicio) RechazarSPM(ctx context.Context, a d.Actor, clave string) error {
	return s.RechazarDecision(ctx, a, clave)
}
func (s *Servicio) OperarSPM(ctx context.Context, a d.Actor, id, clave, accion string, p d.PeticionSPM) (r d.SalidaSPM, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarSPM(ctx, a, fallo.Error()) != nil {
			r, fallo = d.SalidaSPM{}, d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarSPM(); err != nil {
		return r, err
	}
	if accion == "forzar" && a.Autorizar() != nil {
		return r, d.ErrAcceso
	}
	if !d.IDValido(id) {
		return r, d.ErrSPM
	}
	if !d.ClaveValida(clave) {
		return r, d.ErrClave
	}
	if s.d.Repositorio == nil || s.d.SalidasSPM == nil || s.d.Transacciones == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil || s.d.Identificadores == nil {
		return r, d.ErrNoDisponible
	}
	b, _ := json.Marshal(p)
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id + "/spm/" + accion
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		// Se comprueba el acceso actual antes de devolver un recibo antiguo.
		e, err := s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if e.EquipoID == nil {
			return d.ErrEstado
		}
		if err := a.AutorizarEquipoSPM(*e.EquipoID); err != nil {
			return err
		}
		previa, repetida, err := s.d.SalidasSPM.RepetirSPM(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			r = previa
			return nil
		}
		actual, err := s.d.SalidasSPM.LeerSPM(ctx, id, true)
		if err != nil {
			return err
		}
		e, err = s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if e.Estado == "cerrada" {
			return d.ErrEstado
		}
		version := e.Version
		if actual != nil {
			version = actual.Version
		}
		if p.Version != version {
			return d.Conflicto{VersionActual: version}
		}
		if accion == "crear" && actual != nil {
			return d.ErrEstado
		}
		if err := p.ValidarSPM(accion, actual); err != nil {
			return err
		}
		if actual != nil {
			r = *actual
		} else {
			nuevo, err := s.d.Identificadores.Nuevo()
			if err != nil {
				return err
			}
			r = d.SalidaSPM{ID: nuevo, ExpedienteID: id, EquipoID: *e.EquipoID, Version: 0, Motivo: p.Motivo, PropuestaMedida: p.PropuestaMedida, Respuesta: d.SPMAlaEspera}
		}
		r.Version++
		if accion != "forzar" {
			r.ForzadoID = ""
			r.MotivoForzado = ""
		}
		switch accion {
		case "confirmar":
			r.FechaDerivacion = p.Fecha
		case "responder":
			r.Respuesta = p.Respuesta
			r.FechaRespuesta = p.Fecha
			r.ForzadoID = ""
			r.MotivoForzado = ""
		case "forzar":
			r.MotivoForzado = p.Motivo
			r.ForzadoID, err = s.d.Identificadores.Nuevo()
			if err != nil {
				return err
			}
		}
		if err := s.d.SalidasSPM.GuardarSPM(ctx, accion, r, p, a.UsuarioID, ahora); err != nil {
			return err
		}
		evento := s.evento(a, "expediente.spm_"+accion, id, "ok")
		if accion == "forzar" {
			evento.EntidadID = r.ForzadoID
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, evento); err != nil {
			return err
		}
		return s.d.SalidasSPM.RecordarSPM(ctx, a.UsuarioID, clave, ruta, huella, r, ahora)
	})
	if err != nil {
		var c d.Conflicto
		if errors.As(err, &c) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrEstado) || errors.Is(err, d.ErrSPM) || errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrIdempotencia) {
			return d.SalidaSPM{}, err
		}
		return d.SalidaSPM{}, d.ErrNoDisponible
	}
	return r, nil
}
func (s *Servicio) ConsultarSalidaSPM(ctx context.Context, a d.Actor, id string) (r *d.SalidaSPM, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarSPM(ctx, a, fallo.Error()) != nil {
			r, fallo = nil, d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarSPM(); err != nil {
		return nil, err
	}
	if !d.IDValido(id) {
		return nil, d.ErrSPM
	}
	if s.d.Repositorio == nil || s.d.SalidasSPM == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return nil, d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		e, err := s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if e.EquipoID == nil {
			return d.ErrEstado
		}
		if err := a.AutorizarEquipoSPM(*e.EquipoID); err != nil {
			return err
		}
		r, err = s.d.SalidasSPM.LeerSPM(ctx, id, false)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.spm_consultar", id, "ok"))
		return err
	})
	return r, err
}

type ConsultaCierreSPM struct {
	Repositorio salida.Repositorio
	Salidas     salida.SalidasSPM
}

var _ entrada.ConsultaCierreSPM = ConsultaCierreSPM{}

func (c ConsultaCierreSPM) PermiteCierreOrdinarioSPM(ctx context.Context, a entrada.Actor, id string) (bool, error) {
	r, err := c.ConsultarSalidaSPM(ctx, a, id)
	if err != nil {
		return false, err
	}
	return d.PermiteCierreOrdinarioSPM(r), nil
}
func (c ConsultaCierreSPM) ConsultarSalidaSPM(ctx context.Context, a entrada.Actor, id string) (*d.SalidaSPM, error) {
	if err := a.AutorizarSPM(); err != nil {
		return nil, err
	}
	if !d.IDValido(id) {
		return nil, d.ErrSPM
	}
	if c.Repositorio == nil || c.Salidas == nil {
		return nil, d.ErrNoDisponible
	}
	e, err := c.Repositorio.Obtener(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.EquipoID == nil {
		return nil, d.ErrEstado
	}
	if err := a.AutorizarEquipoSPM(*e.EquipoID); err != nil {
		return nil, err
	}
	r, err := c.Salidas.LeerSPM(ctx, id, true)
	if err != nil {
		return nil, err
	}
	return r, nil
}
