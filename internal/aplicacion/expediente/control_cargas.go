package expediente

import (
	"context"
	"slices"

	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

func (s *Servicio) controlCarga(ctx context.Context, a d.Actor, accion string, f func(context.Context) error) error {
	if !d.IDValido(a.UsuarioID) {
		return d.ErrAcceso
	}
	if s.d.CargasHistoricas == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if err := f(ctx); err != nil {
			return err
		}
		_, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, "", "ok"))
		return err
	})
	if err != nil && s.RechazarCargaHistorica(ctx, a) != nil {
		return d.ErrNoDisponible
	}
	return err
}
func (s *Servicio) ConsultarPlazoCarga(ctx context.Context, a d.Actor) (r d.PlazoCarga, err error) {
	if !slices.Contains(a.Perfiles, "administracion") {
		if err = a.AutorizarConsulta(); err != nil {
			return r, err
		}
	}
	err = s.controlCarga(ctx, a, "carga.consultar_plazo", func(ctx context.Context) error {
		var e error
		r, e = s.d.CargasHistoricas.PlazoCarga(ctx, false)
		return e
	})
	if err == nil {
		r.PermiteCarga = r.Abierta(s.d.Reloj.Ahora())
	}
	return
}
func (s *Servicio) FijarPlazoCarga(ctx context.Context, a d.Actor, p d.PlazoCarga) (r d.PlazoCarga, err error) {
	if !slices.Contains(a.Perfiles, "administracion") {
		return r, d.ErrAcceso
	}
	if err = p.Validar(); err != nil {
		return r, err
	}
	err = s.controlCarga(ctx, a, "carga.fijar_plazo", func(ctx context.Context) error {
		var e error
		r, e = s.d.CargasHistoricas.GuardarPlazoCarga(ctx, p, s.d.Reloj.Ahora())
		return e
	})
	if err == nil {
		r.PermiteCarga = r.Abierta(s.d.Reloj.Ahora())
	}
	return
}
func (s *Servicio) ConsultarLotesCarga(ctx context.Context, a d.Actor, desplazamiento int) (r []d.LoteCarga, err error) {
	if desplazamiento < 0 || desplazamiento > 100000 {
		return nil, d.ErrControlCarga
	}
	if err = a.AutorizarConsulta(); err != nil {
		return nil, err
	}
	err = s.controlCarga(ctx, a, "carga.consultar_lotes", func(ctx context.Context) error {
		var e error
		r, e = s.d.CargasHistoricas.LotesCarga(ctx, desplazamiento)
		return e
	})
	return
}
func (s *Servicio) AnularLoteCarga(ctx context.Context, a d.Actor, id string) error {
	if !slices.Contains(a.Perfiles, "coordinacion") || !d.IDValido(id) {
		return d.ErrAcceso
	}
	return s.controlCarga(ctx, a, "carga.anular_lote", func(ctx context.Context) error {
		p, err := s.d.CargasHistoricas.PlazoCarga(ctx, true)
		if err != nil {
			return err
		}
		if !p.Abierta(s.d.Reloj.Ahora()) {
			return d.ErrCargaCerrada
		}
		return s.d.CargasHistoricas.AnularLoteCarga(ctx, id, s.d.Reloj.Ahora())
	})
}
func (s *Servicio) revisarCarga(ctx context.Context, p d.CargaHistorica, confirmar bool) ([]d.AvisoCarga, error) {
	plazo, err := s.d.CargasHistoricas.PlazoCarga(ctx, true)
	if err != nil {
		return nil, err
	}
	if !plazo.Abierta(s.d.Reloj.Ahora()) {
		return nil, d.ErrCargaCerrada
	}
	regla, err := s.d.Numeracion.ReglaVigente(ctx, s.d.Reloj.Ahora())
	if err != nil {
		return nil, err
	}
	if err = s.d.CargasHistoricas.ComprobarNumeracionCarga(ctx, p, s.d.Reloj.Ahora().Year(), regla.Limite()); err != nil {
		return nil, err
	}
	candidatos, err := s.d.CargasHistoricas.CandidatosCarga(ctx, p)
	if err != nil {
		return nil, err
	}
	avisos := []d.AvisoCarga{}
	for i, c := range p.Casos {
		coincide := false
		for _, previo := range candidatos {
			if c.Numero != "" && c.Numero == previo.Numero {
				avisos = append(avisos, d.AvisoCarga{Fila: i + 1, Clave: "carga.numero_ocupado"})
				coincide = false
				break
			}
			if d.CoincidenciaHistorica(c, previo, *plazo.DiasDuplicados) {
				coincide = true
			}
		}
		for j := 0; j < i; j++ {
			if d.CoincidenciaHistorica(c, p.Casos[j], *plazo.DiasDuplicados) {
				coincide = true
			}
		}
		if coincide {
			avisos = append(avisos, d.AvisoCarga{Fila: i + 1, Clave: "carga.posible_duplicado"})
		}
	}
	if confirmar && len(avisos) > 0 && !p.ConfirmarDuplicados {
		return nil, d.ErrDuplicadosSinConfirmar
	}
	return avisos, nil
}
