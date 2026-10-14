package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

var _ entrada.Hitos = (*Servicio)(nil)

func (s *Servicio) ConfirmarHito(ctx context.Context, a dominio.Actor, id, clave string, p dominio.ConfirmacionHito) (recibo dominio.ResultadoHito, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarHito(ctx, a, fallo.Error()) != nil {
			recibo, fallo = dominio.ResultadoHito{}, dominio.ErrNoDisponible
		}
	}()
	if err := a.AutorizarHito(); err != nil {
		return recibo, err
	}
	if !dominio.IDValido(id) {
		return recibo, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return recibo, dominio.ErrClave
	}
	if s.d.Hitos == nil || s.d.Transacciones == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil {
		return recibo, dominio.ErrNoDisponible
	}
	b, err := json.Marshal(p)
	if err != nil {
		return recibo, dominio.ErrPeticion
	}
	huellaBytes := sha256.Sum256(b)
	huella := hex.EncodeToString(huellaBytes[:])
	ruta := "/api/expedientes/" + id + "/hitos"
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.Hitos.RepeticionHito(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			d, err := s.d.Hitos.BloquearHito(ctx, id)
			if err != nil {
				return err
			}
			if a.Autorizar() != nil && d.EquipoID != a.EquipoID {
				return dominio.ErrAcceso
			}
			if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.consultar_recibo_hito", id, "ok")); err != nil {
				return err
			}
			recibo = r
			return nil
		}
		d, err := s.d.Hitos.BloquearHito(ctx, id)
		if err != nil {
			return err
		}
		// La aplicación comprueba el ámbito además de la seguridad por filas.
		if a.Autorizar() != nil && d.EquipoID != a.EquipoID {
			return dominio.ErrAcceso
		}
		if d.Version != p.Version {
			return dominio.Conflicto{VersionActual: d.Version}
		}
		if err := p.Validar(d, instante); err != nil {
			return err
		}
		estado := d.Estado
		if p.Tipo == "consentimiento" {
			estado = dominio.EnIntervencion
		}
		recibo = dominio.ResultadoHito{ID: id, Tipo: p.Tipo, Fecha: p.Fecha, Estado: estado, Version: d.Version + 1}
		if err := s.d.Hitos.ConfirmarHito(ctx, d, p, a.UsuarioID, ahora); err != nil {
			return err
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.confirmar_hito", id, "ok")); err != nil {
			return err
		}
		return s.d.Hitos.RecordarHito(ctx, a.UsuarioID, clave, ruta, huella, recibo, ahora)
	})
	if err != nil {
		return dominio.ResultadoHito{}, falloHito(err)
	}
	return recibo, nil
}

func falloHito(err error) error {
	var conflicto dominio.Conflicto
	if errors.As(err, &conflicto) {
		return err
	}
	for _, conocido := range []error{dominio.ErrAcceso, dominio.ErrNoEncontrada, dominio.ErrPeticion, dominio.ErrEstado, dominio.ErrIdempotencia, dominio.ErrHitoTipo, dominio.ErrHitoFecha, dominio.ErrSubprograma} {
		if errors.Is(err, conocido) {
			return err
		}
	}
	return dominio.ErrNoDisponible
}

func (s *Servicio) ConsultarHitos(ctx context.Context, a dominio.Actor, id string) (hitos []dominio.Hito, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarHito(ctx, a, fallo.Error()) != nil {
			hitos, fallo = nil, dominio.ErrNoDisponible
		}
	}()
	if err := a.AutorizarHito(); err != nil {
		return nil, err
	}
	if !dominio.IDValido(id) {
		return nil, dominio.ErrPeticion
	}
	if s.d.Hitos == nil || s.d.Transacciones == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil {
		return nil, dominio.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		d, err := s.d.Hitos.BloquearHito(ctx, id)
		if err != nil {
			return err
		}
		if a.Autorizar() != nil && d.EquipoID != a.EquipoID {
			return dominio.ErrAcceso
		}
		hitos, err = s.d.Hitos.ConsultarHitos(ctx, id)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.consultar_hitos", id, "ok"))
		return err
	})
	if err != nil {
		return nil, falloHito(err)
	}
	return hitos, nil
}

func (s *Servicio) RechazarHito(ctx context.Context, a dominio.Actor, clave string) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	if a.UsuarioID != "" && !dominio.IDValido(a.UsuarioID) {
		a = dominio.Actor{}
	}
	resultado := "error"
	if clave == dominio.ErrAcceso.Error() {
		resultado = "denegado"
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.hito_rechazado", "", resultado))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
