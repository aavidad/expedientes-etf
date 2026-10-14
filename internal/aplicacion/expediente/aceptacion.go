package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	catalogo "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidadDominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

var _ entrada.Aceptaciones = (*Servicio)(nil)

func (s *Servicio) Aceptar(ctx context.Context, a dominio.Actor, id, clave string, p dominio.Aceptacion) (recibo dominio.ResultadoAceptacion, fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.RechazarAceptacion(ctx, a, fallo.Error()); err != nil {
				recibo = dominio.ResultadoAceptacion{}
				fallo = dominio.ErrNoDisponible
			}
		}
	}()
	if err := a.Autorizar(); err != nil {
		return dominio.ResultadoAceptacion{}, err
	}
	if !dominio.IDValido(id) {
		return dominio.ResultadoAceptacion{}, dominio.ErrPeticion
	}
	if !dominio.ClaveValida(clave) {
		return dominio.ResultadoAceptacion{}, dominio.ErrClave
	}
	if s.d.Aceptaciones == nil || s.d.Numeracion == nil || s.d.Transacciones == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil {
		return dominio.ResultadoAceptacion{}, dominio.ErrNoDisponible
	}
	// Orden canónico: cambiar el orden JSON de menores no cambia la petición.
	p.Menores = slices.Clone(p.Menores)
	slices.SortFunc(p.Menores, func(a, b dominio.SexoMenor) int { return a.Orden - b.Orden })
	b, err := json.Marshal(p)
	if err != nil {
		return dominio.ResultadoAceptacion{}, dominio.ErrPeticion
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/coordinacion/derivaciones/" + id + "/aceptar"
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	var resultado dominio.ResultadoAceptacion
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		resultado, err = s.aceptacionEnTransaccion(ctx, a, id, clave, p, ruta, huella, ahora, instante, nil)
		return err
	})
	if err != nil {
		var conflicto dominio.Conflicto
		if errors.As(err, &conflicto) || errors.Is(err, dominio.ErrTraslado) || errors.Is(err, dominio.ErrFechaEspera) || errors.Is(err, dominio.ErrEstado) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrObligatorios) || errors.Is(err, dominio.ErrProfesional) || errors.Is(err, dominio.ErrSexo) || errors.Is(err, dominio.ErrNumeracion) {
			return dominio.ResultadoAceptacion{}, err
		}
		return dominio.ResultadoAceptacion{}, dominio.ErrNoDisponible
	}
	return resultado, nil
}

// RechazarAceptacion confirma el intento fallido después de revertir la escritura.
func (s *Servicio) RechazarAceptacion(ctx context.Context, a dominio.Actor, clave string) error {
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
	if _, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.aceptacion_rechazada", "", resultado)); err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}

func (s *Servicio) aceptacionEnTransaccion(ctx context.Context, a dominio.Actor, id, clave string, p dominio.Aceptacion, ruta, huella string, ahora, instante time.Time, numero *dominio.NumeroHistorico) (resultado dominio.ResultadoAceptacion, fallo error) {
	fallo = func() error {
		r, repetida, err := s.d.Aceptaciones.Repeticion(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			resultado = r
			return nil
		}
		d, err := s.d.Aceptaciones.Bloquear(ctx, id)
		if err != nil {
			return err
		}
		if d.Version != p.Version {
			return dominio.Conflicto{VersionActual: d.Version}
		}
		if d.Estado != dominio.DerivacionRecibida && d.Estado != dominio.ListaEspera {
			return dominio.ErrEstado
		}
		if d.Estado == dominio.ListaEspera && p.FechaActa != dominio.FechaCivil(instante) {
			return dominio.ErrFechaEspera
		}
		if d.Traslado != nil && d.Traslado.EquipoDestinoID != p.EquipoID {
			return dominio.ErrTraslado
		}
		if err := p.Validar(d); err != nil {
			return err
		}
		fecha, _ := p.Fecha()
		fechaCatalogo := fecha
		if numero != nil {
			fechaCatalogo = instante
		}
		v, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", fechaCatalogo)
		if err != nil {
			return err
		}
		datos, err := s.d.Catalogo.Datos(ctx, v)
		if err != nil {
			return err
		}
		activo := func(e catalogo.Equipo) bool {
			return numero != nil || !fecha.Before(catalogo.FechaCivil(e.ActivoDesde)) && (e.ActivoHasta == nil || fecha.Before(catalogo.FechaCivil(*e.ActivoHasta)))
		}
		municipioOK, centroOK, equipoCodigo := false, false, ""
		for _, m := range datos.Municipios {
			if m.ID == p.MunicipioID {
				municipioOK = true
			}
		}
		// La propuesta geográfica no impone la asignación: Coordinación la confirma.
		for _, e := range datos.Equipos {
			if e.ID == p.EquipoID && activo(e) {
				equipoCodigo = e.Codigo
			}
		}
		for _, c := range datos.Centros {
			if c.ID == p.CentroID && activo(c) {
				centroOK = true
			}
		}
		if !municipioOK || !centroOK || equipoCodigo == "" {
			return dominio.ErrObligatorios
		}
		if p.ProfesionalID != nil {
			if s.d.Usuarios == nil {
				return dominio.ErrProfesional
			}
			u, err := s.d.Usuarios.Obtener(ctx, *p.ProfesionalID)
			if err != nil || u.Estado != identidadDominio.Activo || u.EquipoID != p.EquipoID || !slices.Contains(u.Perfiles, identidadDominio.ProfesionalETF) {
				return dominio.ErrProfesional
			}
		}
		siguiente := 0
		textoNumero, reglaID := "", ""
		if numero == nil {
			regla, err := s.d.Numeracion.ReglaVigente(ctx, fecha)
			if err != nil {
				return err
			}
			if _, err := regla.Formatear(equipoCodigo, 1, fecha.Year()); err != nil {
				return err
			}
			siguiente, err = s.d.Aceptaciones.Siguiente(ctx, p.EquipoID, fecha.Year(), regla.Limite())
			if err != nil {
				return err
			}
			textoNumero, err = regla.Formatear(equipoCodigo, siguiente, fecha.Year())
			if err != nil {
				return err
			}
			reglaID = regla.Version.ID
		} else if numero.Numero != "" {
			siguiente, err = numero.Validar(equipoCodigo, fecha.Year())
			if err != nil {
				return err
			}
			textoNumero = numero.Numero
		}
		resultado = dominio.ResultadoAceptacion{ID: id, Numero: textoNumero, Estado: dominio.Aceptada, Version: d.Version + 1, EquipoID: p.EquipoID, Anio: fecha.Year(), Correlativo: siguiente, ReglaID: reglaID, FechaActa: p.FechaActa}
		if err := s.d.Aceptaciones.Aceptar(ctx, p, resultado, a.UsuarioID, ahora); err != nil {
			return err
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.aceptar_derivacion", id, "ok")); err != nil {
			return err
		}
		return s.d.Aceptaciones.Recordar(ctx, a.UsuarioID, clave, ruta, huella, resultado, ahora)
	}()
	return resultado, fallo
}
