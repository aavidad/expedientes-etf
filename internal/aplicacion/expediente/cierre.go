package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

var _ entrada.Cierres = (*Servicio)(nil)

func (s *Servicio) Cerrar(ctx context.Context, a d.Actor, id, clave string, p d.PeticionCierre) (d.ResultadoCierre, error) {
	return s.actoCierre(ctx, a, id, clave, "cerrar", p, p.Version)
}
func (s *Servicio) Reabrir(ctx context.Context, a d.Actor, id, clave string, p d.PeticionReapertura) (d.ResultadoCierre, error) {
	return s.actoCierre(ctx, a, id, clave, "reabrir", p, p.Version)
}
func (s *Servicio) actoCierre(ctx context.Context, a d.Actor, id, clave, accion string, p any, version int) (r d.ResultadoCierre, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			r, fallo = d.ResultadoCierre{}, d.ErrNoDisponible
		}
	}()
	if err := a.Autorizar(); err != nil {
		return r, err
	}
	if !d.IDValido(id) || version < 1 {
		return r, d.ErrCierre
	}
	if !d.ClaveValida(clave) {
		return r, d.ErrClave
	}
	if s.d.EmisionesFinalizacion == nil || s.d.Repositorio == nil || s.d.Cierres == nil || s.d.ConsultaCierreSPM == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return r, d.ErrNoDisponible
	}
	b, err := json.Marshal(p)
	if err != nil {
		return r, d.ErrCierre
	}
	defer clear(b)
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id + "/" + accion
	ahora := s.d.Reloj.Ahora().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		// La autorización actual precede al recibo; se conserva el orden de bloqueos de F3.02a.
		if _, err := s.d.Repositorio.Obtener(ctx, id); err != nil {
			return err
		}
		previa, repetida, err := s.d.Cierres.RepetirActoCierre(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			r = previa
			return nil
		}
		permite, err := s.d.ConsultaCierreSPM.PermiteCierreOrdinarioSPM(ctx, a, id)
		if err != nil {
			return err
		}
		situacion, err := s.d.Cierres.BloquearCierre(ctx, id)
		if err != nil {
			return err
		}
		if situacion.Version != version {
			return d.Conflicto{VersionActual: situacion.Version}
		}
		if accion == "cerrar" {
			peticion := p.(d.PeticionCierre)
			r, err = s.confirmarCierre(ctx, a, id, situacion, peticion, permite, ahora)
		} else {
			peticion := p.(d.PeticionReapertura)
			if err := peticion.Validar(situacion, ahora); err != nil {
				return err
			}
			r, err = s.d.Cierres.Reabrir(ctx, id, situacion, peticion, d.EstadoReapertura(situacion.Consentimiento, permite), a.UsuarioID, ahora)
		}
		if err != nil {
			return err
		}
		if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente."+accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Cierres.RecordarActoCierre(ctx, a.UsuarioID, clave, ruta, huella, r, ahora)
	})
	if err != nil {
		var c d.Conflicto
		if errors.As(err, &c) || errors.Is(err, d.ErrCierre) || errors.Is(err, d.ErrDobleIntervencion) || errors.Is(err, d.ErrCierreSPM) || errors.Is(err, d.ErrEstado) || errors.Is(err, d.ErrFinalizacion) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrIdempotencia) {
			return d.ResultadoCierre{}, err
		}
		return d.ResultadoCierre{}, d.ErrNoDisponible
	}
	return r, nil
}

// confirmarCierre se ejecuta dentro de la transacción del cierre o del traslado.
func (s *Servicio) confirmarCierre(ctx context.Context, a d.Actor, id string, situacion d.SituacionCierre, peticion d.PeticionCierre, permite bool, ahora time.Time) (d.ResultadoCierre, error) {
	if err := peticion.Validar(situacion, a.UsuarioID, ahora); err != nil {
		return d.ResultadoCierre{}, err
	}
	if !permite {
		spm, err := s.d.ConsultaCierreSPM.ConsultarSalidaSPM(ctx, a, id)
		if err != nil {
			return d.ResultadoCierre{}, err
		}
		if spm == nil || spm.ForzadoID == "" || spm.MotivoForzado == "" || peticion.ForzadoID != spm.ForzadoID {
			return d.ResultadoCierre{}, d.ErrCierreSPM
		}
	} else if peticion.ForzadoID != "" {
		return d.ResultadoCierre{}, d.ErrCierre
	}
	emitida, err := s.d.EmisionesFinalizacion.FinalizacionEmitida(ctx, id, situacion.Version-1)
	if err != nil {
		return d.ResultadoCierre{}, err
	}
	r, err := s.d.Cierres.ConfirmarCierre(ctx, id, situacion, peticion, a.UsuarioID, ahora)
	if err == nil && s.d.Conservacion != nil {
		err = s.d.Conservacion.Marcar(ctx, id, ahora)
	}
	if !emitida {
		r.Aviso = "expediente.cierre_sin_anexo"
	}
	return r, err
}
