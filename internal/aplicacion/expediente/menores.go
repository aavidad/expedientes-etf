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

func (s *Servicio) RechazarMenores(ctx context.Context, a dominio.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	if !dominio.IDValido(a.UsuarioID) {
		a = dominio.Actor{}
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.menores_rechazados", "", "error"))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
func falloMenores(err error) error {
	if err == nil {
		return nil
	}
	var conflicto dominio.Conflicto
	if errors.As(err, &conflicto) || errors.Is(err, dominio.ErrMenores) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrAcceso) || errors.Is(err, dominio.ErrEstado) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrClave) || errors.Is(err, dominio.ErrPeticion) {
		return err
	}
	return dominio.ErrNoDisponible
}
func (s *Servicio) disponibleMenores() bool {
	return s.d.Menores != nil && s.d.CatalogosMenores != nil && s.d.Transacciones != nil && s.d.RegistroTransaccional != nil && s.d.Reloj != nil
}
func (s *Servicio) EditarMenores(ctx context.Context, a dominio.Actor, id, clave string, p dominio.CambioMenores) (resultado dominio.ResultadoMenores, fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.RechazarMenores(ctx, a); err != nil {
				fallo = err
			}
			resultado = dominio.ResultadoMenores{}
		}
	}()
	if err := a.AutorizarMenores(true); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return resultado, dominio.ErrClave
	}
	if !s.disponibleMenores() {
		return resultado, dominio.ErrNoDisponible
	}
	if p.Menores != nil {
		menores := slices.Clone(*p.Menores)
		slices.SortFunc(menores, func(a, b dominio.Menor) int { return a.Orden - b.Orden })
		if menores == nil {
			menores = []dominio.Menor{}
		}
		p.Menores = &menores
	}
	b, err := json.Marshal(p)
	if err != nil {
		return resultado, dominio.ErrMenores
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id + "/menores"
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.menoresEnTransaccion(ctx, a, id, clave, p, ruta, huella, ahora, instante)
		return err
	})
	return resultado, falloMenores(err)
}
func (s *Servicio) ConsultarMenores(ctx context.Context, a dominio.Actor, id string) (resultado dominio.FichaMenores, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarMenores(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.FichaMenores{}
		}
	}()
	if err := a.AutorizarMenores(false); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !s.disponibleMenores() {
		return resultado, dominio.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.d.Menores.ObtenerMenores(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && resultado.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_menores", id, "ok"))
		return err
	})
	return resultado, falloMenores(err)
}
func (s *Servicio) ConsultarHistoriaMenores(ctx context.Context, a dominio.Actor, id, corte string) (resultado dominio.HistoriaMenores, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarMenores(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.HistoriaMenores{}
		}
	}()
	if err := a.AutorizarMenores(false); err != nil {
		return resultado, err
	}
	if !s.disponibleMenores() {
		return resultado, dominio.ErrNoDisponible
	}
	fecha, err := time.Parse("2006-01-02", corte)
	if err != nil || !dominio.IDValido(id) || fecha.Year() < 1 || corte > dominio.FechaCivil(s.d.Reloj.Ahora()) {
		return resultado, dominio.ErrMenores
	}
	if !s.disponibleMenores() {
		return resultado, dominio.ErrNoDisponible
	}
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		ficha, err := s.d.Menores.ObtenerMenores(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && ficha.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		resultado, err = s.d.Menores.HistoriaMenores(ctx, id, corte)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_historia_menores", id, "ok"))
		return err
	})
	return resultado, falloMenores(err)
}

func (s *Servicio) menoresEnTransaccion(ctx context.Context, a dominio.Actor, id, clave string, p dominio.CambioMenores, ruta, huella string, ahora, instante time.Time) (resultado dominio.ResultadoMenores, fallo error) {
	fallo = func() error {
		r, repetida, err := s.d.Menores.RepetirMenores(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			resultado = r
			return nil
		}
		ficha, err := s.d.Menores.BloquearMenores(ctx, id)
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
		catalogos, err := s.d.CatalogosMenores.Obtener(ctx)
		if err != nil {
			return err
		}
		if p.Menores != nil {
			for i := range *p.Menores {
				if (*p.Menores)[i].Subprograma == "" {
					(*p.Menores)[i].Subprograma = ficha.Subprograma
				}
			}
		}
		if err := p.Validar(catalogos, ficha.Aceptacion, instante); err != nil {
			return err
		}
		if p.Menores != nil {
			existentes := []string{}
			for i := range *p.Menores {
				menor := &(*p.Menores)[i]
				if menor.ID == "" {
					if s.d.Identificadores == nil {
						return dominio.ErrNoDisponible
					}
					menor.ID, err = s.d.Identificadores.Nuevo()
					if err != nil {
						return err
					}
				} else {
					existentes = append(existentes, menor.ID)
				}
			}
			pertenecen, err := s.d.Menores.MenoresPertenecen(ctx, id, existentes)
			if err != nil {
				return err
			}
			if !pertenecen {
				return dominio.ErrMenores
			}
		}
		if err := s.d.Menores.GuardarMenores(ctx, id, p, a.UsuarioID, ahora); err != nil {
			return err
		}
		resultado = dominio.ResultadoMenores{ID: id, Version: p.Version + 1}
		accion := "expediente.editar_menores"
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Menores.RecordarMenores(ctx, a.UsuarioID, clave, ruta, huella, resultado, ahora)
	}()
	return resultado, fallo
}
