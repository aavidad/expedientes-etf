package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"slices"
	"time"
)

func (s *Servicio) RechazarIntervencion(ctx context.Context, a dominio.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	if !dominio.IDValido(a.UsuarioID) {
		a = dominio.Actor{}
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.intervencion_rechazados", "", "error"))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
func falloIntervencion(err error) error {
	if err == nil {
		return nil
	}
	var conflicto dominio.Conflicto
	if errors.As(err, &conflicto) || errors.Is(err, dominio.ErrIntervencion) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrAcceso) || errors.Is(err, dominio.ErrEstado) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrClave) || errors.Is(err, dominio.ErrPeticion) {
		return err
	}
	return dominio.ErrNoDisponible
}
func (s *Servicio) disponibleIntervencion() bool {
	return s.d.Intervencion != nil && s.d.Menores != nil && s.d.CatalogosIntervencion != nil && s.d.Transacciones != nil && s.d.RegistroTransaccional != nil && s.d.Reloj != nil
}
func (s *Servicio) EditarIntervencion(ctx context.Context, a dominio.Actor, id, clave string, p dominio.CambioIntervencion) (resultado dominio.ResultadoIntervencion, fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.RechazarIntervencion(ctx, a); err != nil {
				fallo = err
			}
			resultado = dominio.ResultadoIntervencion{}
		}
	}()
	if err := a.AutorizarIntervencion(true); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return resultado, dominio.ErrClave
	}
	if !s.disponibleIntervencion() {
		return resultado, dominio.ErrNoDisponible
	}
	if p.Intervencion != nil {
		normalizada := p.Intervencion.Normalizada()
		p.Intervencion = &normalizada
	}
	b, err := json.Marshal(p)
	if err != nil {
		return resultado, dominio.ErrIntervencion
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id + "/intervencion"
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.Intervencion.RepetirIntervencion(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			resultado = r
			return nil
		}
		ficha, err := s.d.Intervencion.BloquearIntervencion(ctx, id)
		if err != nil {
			return err
		}
		if ficha.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		if ficha.Version != p.Version {
			return dominio.Conflicto{VersionActual: ficha.Version}
		}
		if ficha.Estado != "aceptada" && ficha.Estado != "en_intervencion" {
			return dominio.ErrEstado
		}
		fecha, err := time.Parse("2006-01-02", p.Desde)
		if err != nil || fecha.Year() < 1 || fecha.Before(ficha.Aceptacion) || p.Desde > dominio.FechaCivil(instante) || p.Version < 1 || p.Intervencion == nil {
			return dominio.ErrIntervencion
		}
		catalogos, err := s.d.CatalogosIntervencion.ObtenerIntervencion(ctx, p.Desde)
		if err != nil {
			return err
		}
		if err := p.Validar(catalogos, ficha.Aceptacion, instante); err != nil {
			return err
		}
		ids := p.Intervencion.Exclusiones()
		if len(ids) > 0 {
			historia, err := s.d.Menores.HistoriaMenores(ctx, id, p.Desde)
			if err != nil {
				return err
			}
			vigentes := map[string]bool{}
			for _, m := range historia.Menores {
				vigentes[m.ID] = true
			}
			for _, id := range ids {
				if !vigentes[id] {
					return dominio.ErrIntervencion
				}
			}
		}
		if err := s.d.Intervencion.GuardarIntervencion(ctx, id, p, a.UsuarioID, ahora); err != nil {
			return err
		}
		resultado = dominio.ResultadoIntervencion{ID: id, Version: p.Version + 1}
		accion := "expediente.editar_intervencion"
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Intervencion.RecordarIntervencion(ctx, a.UsuarioID, clave, ruta, huella, resultado, ahora)
	})
	return resultado, falloIntervencion(err)
}
func (s *Servicio) ConsultarIntervencion(ctx context.Context, a dominio.Actor, id string) (resultado dominio.FichaIntervencion, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarIntervencion(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.FichaIntervencion{}
		}
	}()
	if err := a.AutorizarIntervencion(false); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !s.disponibleIntervencion() {
		return resultado, dominio.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.d.Intervencion.ObtenerIntervencion(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && resultado.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_intervencion", id, "ok"))
		return err
	})
	return resultado, falloIntervencion(err)
}
func (s *Servicio) ConsultarHistoriaIntervencion(ctx context.Context, a dominio.Actor, id, corte string) (resultado dominio.HistoriaIntervencion, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarIntervencion(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.HistoriaIntervencion{}
		}
	}()
	if err := a.AutorizarIntervencion(false); err != nil {
		return resultado, err
	}
	if !s.disponibleIntervencion() {
		return resultado, dominio.ErrNoDisponible
	}
	fecha, err := time.Parse("2006-01-02", corte)
	if err != nil || !dominio.IDValido(id) || fecha.Year() < 1 || corte > dominio.FechaCivil(s.d.Reloj.Ahora()) {
		return resultado, dominio.ErrIntervencion
	}
	if !s.disponibleIntervencion() {
		return resultado, dominio.ErrNoDisponible
	}
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		ficha, err := s.d.Intervencion.ObtenerIntervencion(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && ficha.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		resultado, err = s.d.Intervencion.HistoriaIntervencion(ctx, id, corte)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_historia_intervencion", id, "ok"))
		return err
	})
	return resultado, falloIntervencion(err)
}
