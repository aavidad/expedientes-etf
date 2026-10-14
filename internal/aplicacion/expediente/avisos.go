package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"slices"
	"time"
)

var _ entrada.Avisos = (*Servicio)(nil)

func (s *Servicio) RechazarAvisos(ctx context.Context, a dominio.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	if !dominio.IDValido(a.UsuarioID) {
		a = dominio.Actor{}
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.avisos_rechazados", "", "error"))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
func (s *Servicio) ejecutarAvisos(ctx context.Context, a dominio.Actor, accion, id string, op func(context.Context) error) (fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.RechazarAvisos(ctx, a); err != nil {
				fallo = err
			}
		}
	}()
	if err := a.AutorizarFamilia(false); err != nil {
		return err
	}
	if s.d.Avisos == nil || s.d.Catalogo == nil || s.d.Reloj == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil {
		return dominio.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if err := op(ctx); err != nil {
			return err
		}
		_, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok"))
		return err
	})
	if errors.Is(err, dominio.ErrPeticion) || errors.Is(err, dominio.ErrAcceso) || errors.Is(err, dominio.ErrNoEncontrada) {
		return err
	}
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
func (s *Servicio) reglasAvisos(ctx context.Context, ahora time.Time) (dominio.ReglasAvisos, int, error) {
	v, err := s.d.Catalogo.Vigente(ctx, "plazos", ahora)
	if err != nil {
		return dominio.ReglasAvisos{}, 0, err
	}
	d, err := s.d.Catalogo.Datos(ctx, v)
	if err != nil {
		return dominio.ReglasAvisos{}, 0, err
	}
	r := dominio.ReglasAvisos{Duraciones: map[string]int{}}
	encontrada := false
	for _, p := range d.Plazos {
		if p.Clave == "plazo_consentimiento" {
			if p.Valor < 0 || p.Unidad != "dias_naturales" {
				return r, 0, dominio.ErrNoDisponible
			}
			r.ConsentimientoDias = p.Valor
			encontrada = true
		}
	}
	if !encontrada {
		return r, 0, dominio.ErrNoDisponible
	}
	for _, f := range d.DuracionesFase {
		r.Duraciones[f.Subprograma+":"+f.Fase] = f.DiasOrientativos
	}
	return r, v.Numero, nil
}
func (s *Servicio) ConsultarAvisos(ctx context.Context, a dominio.Actor, id string) (entrada.AvisosExpediente, error) {
	r := entrada.AvisosExpediente{}
	err := s.ejecutarAvisos(ctx, a, "expediente.consultar_avisos", id, func(ctx context.Context) error {
		if !dominio.IDValido(id) {
			return dominio.ErrPeticion
		}
		ahora := s.d.Reloj.Ahora()
		hoy := dominio.FechaCivil(ahora)
		reglas, version, err := s.reglasAvisos(ctx, ahora)
		if err != nil {
			return err
		}
		datos, err := s.d.Avisos.DatosAvisos(ctx, id, "", hoy)
		if err != nil {
			return err
		}
		if len(datos) != 1 {
			return dominio.ErrNoDisponible
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && datos[0].EquipoID != a.EquipoID {
			return dominio.ErrAcceso
		}
		avisos, err := dominio.CalcularAvisos(datos[0], hoy, reglas, true)
		if err != nil {
			return err
		}
		r = entrada.AvisosExpediente{ID: id, Corte: hoy, VersionPlazos: version, Avisos: avisos}
		return nil
	})
	if err != nil {
		return entrada.AvisosExpediente{}, err
	}
	return r, nil
}
func (s *Servicio) ContarAvisos(ctx context.Context, a dominio.Actor, equipo string) (entrada.RecuentoAvisos, error) {
	r := entrada.RecuentoAvisos{EquipoID: equipo, PorCodigo: map[string]int{}}
	err := s.ejecutarAvisos(ctx, a, "expediente.contar_avisos", "", func(ctx context.Context) error {
		if !dominio.IDValido(equipo) {
			return dominio.ErrPeticion
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && equipo != a.EquipoID {
			return dominio.ErrAcceso
		}
		ahora := s.d.Reloj.Ahora()
		hoy := dominio.FechaCivil(ahora)
		reglas, _, err := s.reglasAvisos(ctx, ahora)
		if err != nil {
			return err
		}
		datos, err := s.d.Avisos.DatosAvisos(ctx, "", equipo, hoy)
		if err != nil {
			return err
		}
		expedientes := map[string]bool{}
		contar := func(d dominio.DatosAvisos, avisos []dominio.Aviso) error {
			if d.EquipoID != equipo {
				return dominio.ErrAcceso
			}
			if len(avisos) > 0 {
				expedientes[d.ID] = true
			}
			for _, aviso := range avisos {
				r.Total++
				r.PorCodigo[aviso.Codigo]++
			}
			return nil
		}
		for _, d := range datos {
			avisos, err := dominio.CalcularAvisos(d, hoy, reglas, false)
			if err != nil {
				return err
			}
			if err := contar(d, avisos); err != nil {
				return err
			}
		}
		if ahora.Month() == time.January {
			corte := time.Date(ahora.Year()-1, time.December, 31, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			datos, err = s.d.Avisos.DatosAvisos(ctx, "", equipo, corte)
			if err != nil {
				return err
			}
			for _, d := range datos {
				if !d.AtendidoAnio {
					continue
				}
				// En enero sólo se añaden los datos de Memoria al corte del año anterior.
				d.Estado = "cerrado"
				avisos, err := dominio.CalcularAvisos(d, corte, reglas, true)
				if err != nil {
					return err
				}
				if err := contar(d, avisos); err != nil {
					return err
				}
			}
		}
		r.Expedientes = len(expedientes)
		return nil
	})
	if err != nil {
		return entrada.RecuentoAvisos{}, err
	}
	return r, nil
}
