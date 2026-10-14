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

func (s *Servicio) RechazarFamilia(ctx context.Context, a dominio.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	if !dominio.IDValido(a.UsuarioID) {
		a = dominio.Actor{}
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.familia_rechazada", "", "error"))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}
func falloFamilia(err error) error {
	if err == nil {
		return nil
	}
	var conflicto dominio.Conflicto
	if errors.As(err, &conflicto) || errors.Is(err, dominio.ErrFamilia) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrAcceso) || errors.Is(err, dominio.ErrEstado) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrClave) || errors.Is(err, dominio.ErrPeticion) {
		return err
	}
	return dominio.ErrNoDisponible
}
func (s *Servicio) disponibleFamilia() bool {
	return s.d.Familias != nil && s.d.CatalogosFamilia != nil && s.d.Transacciones != nil && s.d.RegistroTransaccional != nil && s.d.Reloj != nil
}
func (s *Servicio) EditarFamilia(ctx context.Context, a dominio.Actor, id, clave string, p dominio.CambioFamilia) (resultado dominio.ResultadoFamilia, fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.RechazarFamilia(ctx, a); err != nil {
				fallo = err
			}
			resultado = dominio.ResultadoFamilia{}
		}
	}()
	if err := a.AutorizarFamilia(true); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return resultado, dominio.ErrClave
	}
	if !s.disponibleFamilia() {
		return resultado, dominio.ErrNoDisponible
	}
	if p.Adultos != nil {
		adultos := slices.Clone(*p.Adultos)
		slices.SortFunc(adultos, func(a, b dominio.Adulto) int { return a.Orden - b.Orden })
		if adultos == nil {
			adultos = []dominio.Adulto{}
		}
		p.Adultos = &adultos
	}
	b, err := json.Marshal(p)
	if err != nil {
		return resultado, dominio.ErrFamilia
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/expedientes/" + id
	if p.Adultos != nil {
		ruta += "/adultos"
	}
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.familiaEnTransaccion(ctx, a, id, clave, p, ruta, huella, ahora, instante)
		return err
	})
	return resultado, falloFamilia(err)
}
func (s *Servicio) ConsultarFamilia(ctx context.Context, a dominio.Actor, id string) (resultado dominio.FichaFamilia, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarFamilia(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.FichaFamilia{}
		}
	}()
	if err := a.AutorizarFamilia(false); err != nil {
		return resultado, err
	}
	if !dominio.IDValido(id) {
		return resultado, dominio.ErrPeticion
	}
	if !s.disponibleFamilia() {
		return resultado, dominio.ErrNoDisponible
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.d.Familias.ObtenerFamilia(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && resultado.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_familia", id, "ok"))
		return err
	})
	return resultado, falloFamilia(err)
}
func (s *Servicio) ConsultarHistoriaFamilia(ctx context.Context, a dominio.Actor, id, corte string, anio int) (resultado dominio.HistoriaFamilia, fallo error) {
	defer func() {
		if fallo != nil {
			if e := s.RechazarFamilia(ctx, a); e != nil {
				fallo = e
			}
			resultado = dominio.HistoriaFamilia{}
		}
	}()
	if err := a.AutorizarFamilia(false); err != nil {
		return resultado, err
	}
	if !s.disponibleFamilia() {
		return resultado, dominio.ErrNoDisponible
	}
	fecha, err := time.Parse("2006-01-02", corte)
	if err != nil || !dominio.IDValido(id) || anio < 1 || anio > 9999 || fecha.Year() != anio || corte > dominio.FechaCivil(s.d.Reloj.Ahora()) {
		return resultado, dominio.ErrFamilia
	}
	if !s.disponibleFamilia() {
		return resultado, dominio.ErrNoDisponible
	}
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		ficha, err := s.d.Familias.ObtenerFamilia(ctx, id)
		if err != nil {
			return err
		}
		if !slices.Contains(a.Perfiles, "coordinacion") && ficha.EquipoID != a.EquipoID {
			return dominio.ErrNoEncontrada
		}
		resultado, err = s.d.Familias.HistoriaFamilia(ctx, id, corte)
		if err != nil {
			return err
		}
		resultado.AnioSMI = anio
		if resultado.Familia.Ingresos != nil && s.d.SMI != nil {
			filas, err := s.d.SMI.SMI(ctx)
			if err != nil {
				return err
			}
			catalogos, err := s.d.CatalogosFamilia.Obtener(ctx)
			if err != nil {
				return err
			}
			for _, smi := range filas {
				if smi.Anio == anio && smi.ImporteAnualCentimos != nil {
					resultado.TramoCalculado = catalogos.CalcularTramo(*resultado.Familia.Ingresos, *smi.ImporteAnualCentimos)
					version := smi.Version
					resultado.VersionSMI = &version
				}
			}
		} else {
			resultado.TramoCalculado = resultado.Familia.Tramo
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_historia_familia", id, "ok"))
		return err
	})
	return resultado, falloFamilia(err)
}

func (s *Servicio) familiaEnTransaccion(ctx context.Context, a dominio.Actor, id, clave string, p dominio.CambioFamilia, ruta, huella string, ahora, instante time.Time) (resultado dominio.ResultadoFamilia, fallo error) {
	fallo = func() error {
		r, repetida, err := s.d.Familias.RepetirFamilia(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			resultado = r
			return nil
		}
		ficha, err := s.d.Familias.BloquearFamilia(ctx, id)
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
		catalogos, err := s.d.CatalogosFamilia.Obtener(ctx)
		if err != nil {
			return err
		}
		if err := p.Validar(catalogos, ficha.Aceptacion, instante); err != nil {
			return err
		}
		if p.Adultos != nil {
			existentes := []string{}
			for i := range *p.Adultos {
				adulto := &(*p.Adultos)[i]
				if adulto.ID == "" {
					if s.d.Identificadores == nil {
						return dominio.ErrNoDisponible
					}
					adulto.ID, err = s.d.Identificadores.Nuevo()
					if err != nil {
						return err
					}
				} else {
					existentes = append(existentes, adulto.ID)
				}
			}
			pertenecen, err := s.d.Familias.AdultosPertenecen(ctx, id, existentes)
			if err != nil {
				return err
			}
			if !pertenecen {
				return dominio.ErrFamilia
			}
		}
		if err := s.d.Familias.GuardarFamilia(ctx, id, p, a.UsuarioID, ahora); err != nil {
			return err
		}
		resultado = dominio.ResultadoFamilia{ID: id, Version: p.Version + 1}
		accion := "expediente.editar_familia"
		if p.Adultos != nil {
			accion = "expediente.editar_adultos"
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok")); err != nil {
			return err
		}
		return s.d.Familias.RecordarFamilia(ctx, a.UsuarioID, clave, ruta, huella, resultado, ahora)
	}()
	return resultado, fallo
}
