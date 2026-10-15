package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	catalogo "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

func opcionesUbicacion(datos catalogo.Datos, ahora time.Time) dominio.OpcionesUbicacion {
	o := dominio.OpcionesUbicacion{Municipios: []dominio.OpcionUbicacion{}, Centros: []dominio.OpcionUbicacion{}, Equipos: []dominio.OpcionUbicacion{}}
	for _, m := range datos.Municipios {
		o.Municipios = append(o.Municipios, dominio.OpcionUbicacion{ID: m.ID, ClaveTexto: m.ClaveTexto, CentroID: m.CentroID, EquipoID: m.EquipoID, FueraAmbito: !m.AmbitoDiputacion})
	}
	activo := func(e catalogo.Equipo) bool {
		return !ahora.Before(catalogo.FechaCivil(e.ActivoDesde)) && (e.ActivoHasta == nil || ahora.Before(catalogo.FechaCivil(*e.ActivoHasta)))
	}
	for _, c := range datos.Centros {
		if activo(c) {
			o.Centros = append(o.Centros, dominio.OpcionUbicacion{ID: c.ID, ClaveTexto: c.ClaveTexto})
		}
	}
	for _, e := range datos.Equipos {
		if activo(e) {
			o.Equipos = append(o.Equipos, dominio.OpcionUbicacion{ID: e.ID, ClaveTexto: e.ClaveTexto})
		}
	}
	return o
}
func (s *Servicio) Etiquetas(ctx context.Context, a dominio.Actor, idioma string) (map[string]string, error) {
	if err := a.Autorizar(); err != nil {
		return nil, err
	}
	if s.d.Etiquetas == nil {
		return nil, dominio.ErrNoDisponible
	}
	var nombres map[string]string
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		nombres, err = s.d.Etiquetas.NombresEn(ctx, idioma)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_etiquetas", "", "ok"))
		return err
	})
	if err != nil {
		return nil, dominio.ErrNoDisponible
	}
	return nombres, nil
}
func (s *Servicio) Descartar(ctx context.Context, a dominio.Actor, id string) error {
	if err := a.Autorizar(); err != nil {
		return err
	}
	if !dominio.IDValido(id) {
		return dominio.ErrPeticion
	}
	if s.d.Borradores == nil {
		return dominio.ErrNoDisponible
	}
	return s.d.Borradores.Descartar(ctx, a.UsuarioID, id)
}
func (s *Servicio) ConfirmarDerivacion(ctx context.Context, a dominio.Actor, clave string, p dominio.Confirmacion) (recibo dominio.ReciboDerivacion, fallo error) {
	defer func() {
		if fallo != nil {
			if err := s.Rechazar(ctx, a, fallo.Error()); err != nil {
				fallo = dominio.ErrNoDisponible
			}
		}
	}()
	if err := a.Autorizar(); err != nil {
		return recibo, err
	}
	if !dominio.ClaveValida(clave) {
		return recibo, dominio.ErrClave
	}
	if !dominio.IDValido(p.LecturaID) {
		return recibo, dominio.ErrPeticion
	}
	if s.d.Borradores == nil || s.d.Confirmaciones == nil || s.d.Repositorio == nil {
		return recibo, dominio.ErrNoDisponible
	}
	b, err := json.Marshal(p)
	if err != nil {
		return recibo, dominio.ErrPeticion
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.Confirmaciones.RepetirConfirmacion(ctx, a.UsuarioID, clave, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			recibo = r
			return nil
		}
		d, err := s.d.Borradores.Obtener(ctx, a.UsuarioID, p.LecturaID)
		if err != nil {
			return err
		}
		if err := p.Validar(d.Datos.Extraccion); err != nil {
			return err
		}
		v, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", ahora)
		if err != nil {
			return err
		}
		datos, err := s.d.Catalogo.Datos(ctx, v)
		if err != nil {
			return err
		}
		opciones := opcionesUbicacion(datos, ahora)
		contiene := func(lista []dominio.OpcionUbicacion, id string) bool {
			for _, o := range lista {
				if o.ID == id {
					return true
				}
			}
			return false
		}
		if !contiene(opciones.Municipios, p.Ubicacion.MunicipioID) || !contiene(opciones.Centros, p.Ubicacion.CentroID) || !contiene(opciones.Equipos, p.Ubicacion.EquipoID) {
			return dominio.ErrObligatorios
		}
		if err := s.d.Confirmaciones.BloquearLectura(ctx, p.LecturaID); err != nil {
			return err
		}
		id, err := s.d.Identificadores.Nuevo()
		if err != nil {
			return err
		}
		d.ID, d.Estado, d.Version = id, dominio.DerivacionRecibida, 1
		// La procedencia y el antivirus se conservan del análisis, nunca de la petición.
		d.Datos.Extraccion.Campos = p.Campos
		d.Datos.Extraccion.Tipo = p.Tipo
		d.Datos.Extraccion = dominio.DatosPermitidos(d.Datos.Extraccion)
		d.Datos.UbicacionConfirmada = &p.Ubicacion
		d.Datos.OpcionesUbicacion = dominio.OpcionesUbicacion{}
		if err := s.d.Repositorio.Guardar(ctx, d); err != nil {
			return err
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.confirmar_derivacion", id, "ok")); err != nil {
			return err
		}
		recibo = dominio.ReciboDerivacion{ID: id, Estado: d.Estado, Version: d.Version}
		return s.d.Confirmaciones.RecordarConfirmacion(ctx, a.UsuarioID, clave, huella, recibo, ahora)
	})
	if err != nil {
		recibo = dominio.ReciboDerivacion{}
		if errors.Is(err, dominio.ErrPeticion) || errors.Is(err, dominio.ErrNoEncontrada) || errors.Is(err, dominio.ErrObligatorios) || errors.Is(err, dominio.ErrIdempotencia) || errors.Is(err, dominio.ErrEstado) {
			return recibo, err
		}
		return recibo, dominio.ErrNoDisponible
	}
	_ = s.d.Borradores.Descartar(context.WithoutCancel(ctx), a.UsuarioID, p.LecturaID)
	return recibo, nil
}
