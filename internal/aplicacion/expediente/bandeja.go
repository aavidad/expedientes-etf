package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"sort"
	"strings"
)

func (s *Servicio) ConsultarBandeja(ctx context.Context, a dominio.Actor, p entrada.FiltroBandeja) (resultadoFinal entrada.ResultadoBandeja, fallo error) {
	defer func() {
		if fallo != nil && s.d.Registro != nil && s.d.Reloj != nil {
			if _, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.bandeja_rechazada", "", "denegado")); err != nil {
				resultadoFinal, fallo = entrada.ResultadoBandeja{}, dominio.ErrNoDisponible
			}
		}
	}()
	if err := a.Autorizar(); err != nil {
		return entrada.ResultadoBandeja{}, err
	}
	p.Busqueda = strings.TrimSpace(p.Busqueda)
	if err := p.Validar(); err != nil {
		return entrada.ResultadoBandeja{}, err
	}
	if s.d.Bandeja == nil || s.d.Catalogo == nil || s.d.NombresMunicipios == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return entrada.ResultadoBandeja{}, dominio.ErrNoDisponible
	}
	var resultado entrada.ResultadoBandeja
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		nombres, err := s.d.NombresMunicipios.Nombres(ctx)
		if err != nil {
			return dominio.ErrNoDisponible
		}
		version, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", s.d.Reloj.Ahora())
		if err != nil {
			return err
		}
		datos, err := s.d.Catalogo.Datos(ctx, version)
		if err != nil {
			return err
		}
		codigoMunicipio := ""
		if p.MunicipioID != "" {
			for _, m := range datos.Municipios {
				if m.ID == p.MunicipioID {
					codigoMunicipio = m.INE
				}
			}
			if codigoMunicipio == "" {
				return dominio.ErrPeticion
			}
		}
		resultado, err = s.d.Bandeja.ConsultarBandeja(ctx, p, codigoMunicipio)
		if err != nil {
			return err
		}
		resultado.Equipos = []dominio.OpcionBandeja{}
		resultado.Municipios = []dominio.OpcionBandeja{}
		for _, e := range datos.Equipos {
			resultado.Equipos = append(resultado.Equipos, dominio.OpcionBandeja{ID: e.ID, Etiqueta: nombres[e.ClaveTexto], Codigo: e.Codigo})
		}
		for _, m := range datos.Municipios {
			resultado.Municipios = append(resultado.Municipios, dominio.OpcionBandeja{ID: m.ID, Etiqueta: nombres[m.ClaveTexto], Codigo: m.INE})
		}
		sort.Slice(resultado.Municipios, func(i, j int) bool { return resultado.Municipios[i].Etiqueta < resultado.Municipios[j].Etiqueta })
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.consultar_bandeja", "", "ok"))
		return err
	})
	if err != nil {
		if errors.Is(err, dominio.ErrPeticion) {
			return entrada.ResultadoBandeja{}, err
		}
		return entrada.ResultadoBandeja{}, dominio.ErrNoDisponible
	}
	return resultado, nil
}
