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
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

var _ entrada.Decisiones = (*Servicio)(nil)

func (s *Servicio) NoAceptar(ctx context.Context, a dominio.Actor, id, clave string, p dominio.NoAceptacion) (dominio.ResultadoDecision, error) {
	return s.decidir(ctx, a, id, clave, dominio.NoAceptada, p)
}

func (s *Servicio) DejarEnEspera(ctx context.Context, a dominio.Actor, id, clave string, p dominio.Espera) (dominio.ResultadoDecision, error) {
	return s.decidir(ctx, a, id, clave, dominio.ListaEspera, dominio.NoAceptacion{Version: p.Version})
}

func (s *Servicio) decidir(ctx context.Context, a dominio.Actor, id, clave, estado string, p dominio.NoAceptacion) (recibo dominio.ResultadoDecision, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarDecision(ctx, a, fallo.Error()) != nil {
			recibo, fallo = dominio.ResultadoDecision{}, dominio.ErrNoDisponible
		}
	}()
	if err := a.Autorizar(); err != nil {
		return recibo, err
	}
	if !dominio.IDValido(id) {
		return recibo, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return recibo, dominio.ErrClave
	}
	if s.d.Decisiones == nil || s.d.Aceptaciones == nil || s.d.Transacciones == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil {
		return recibo, dominio.ErrNoDisponible
	}
	// La huella usa la entrada, sin fecha automática: el recibo sobrevive al cambio de día.
	b, err := json.Marshal(p)
	if err != nil {
		return recibo, dominio.ErrPeticion
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	operacion, accion := "no-aceptar", "expediente.no_aceptar_derivacion"
	if estado == dominio.ListaEspera {
		operacion, accion = "lista-espera", "expediente.dejar_lista_espera"
	}
	ruta := "/api/coordinacion/derivaciones/" + id + "/" + operacion
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.Decisiones.RepeticionDecision(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			recibo = r
			return nil
		}
		d, err := s.d.Aceptaciones.Bloquear(ctx, id)
		if err != nil {
			return err
		}
		if d.Version != p.Version {
			return dominio.Conflicto{VersionActual: d.Version}
		}
		if estado == dominio.ListaEspera {
			if err := (dominio.Espera{Version: p.Version}).Validar(d); err != nil {
				return err
			}
			p.Fecha = dominio.FechaCivil(instante)
		} else {
			p, err = p.Validar(d)
			if err != nil {
				return err
			}
		}
		recibo = dominio.ResultadoDecision{ID: id, Estado: estado, Version: d.Version + 1, Fecha: p.Fecha, Subprograma: p.Subprograma}
		if err := s.d.Decisiones.Decidir(ctx, p, recibo, d.Estado, a.UsuarioID, ahora); err != nil {
			return err
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Decisiones.RecordarDecision(ctx, a.UsuarioID, clave, ruta, huella, recibo, ahora)
	})
	if err != nil {
		var conflicto dominio.Conflicto
		if errors.As(err, &conflicto) || errors.Is(err, dominio.ErrEstado) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrPeticion) || errors.Is(err, dominio.ErrMotivo) || errors.Is(err, dominio.ErrSubprograma) || errors.Is(err, dominio.ErrOrigenEspera) {
			return dominio.ResultadoDecision{}, err
		}
		return dominio.ResultadoDecision{}, dominio.ErrNoDisponible
	}
	return recibo, nil
}

func (s *Servicio) RechazarDecision(ctx context.Context, a dominio.Actor, clave string) error {
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
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.decision_rechazada", "", resultado))
	return err
}

func (s *Servicio) ListarEspera(ctx context.Context, a dominio.Actor, p entrada.FiltroEspera) (filas []entrada.FilaEspera, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarDecision(ctx, a, fallo.Error()) != nil {
			filas, fallo = nil, dominio.ErrNoDisponible
		}
	}()
	if err := a.Autorizar(); err != nil {
		return nil, err
	}
	if p.Limite < 1 || p.Limite > 200 || p.Desplazamiento < 0 {
		return nil, dominio.ErrPeticion
	}
	if s.d.Decisiones == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return nil, dominio.ErrNoDisponible
	}
	filas = []entrada.FilaEspera{}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		datos, err := s.d.Decisiones.ListarEspera(ctx, salida.FiltroEspera{Limite: p.Limite, Desplazamiento: p.Desplazamiento})
		if err != nil {
			return err
		}
		for _, d := range datos {
			filas = append(filas, entrada.FilaEspera{ID: d.ID, Estado: d.Estado, Version: d.Version, FechaRecepcion: d.FechaRecepcion, FechaEspera: d.FechaEspera})
		}
		evento := s.evento(a, "expediente.consultar_lista_espera", "", "ok")
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, evento)
		return err
	})
	if err != nil {
		return nil, dominio.ErrNoDisponible
	}
	return filas, nil
}
