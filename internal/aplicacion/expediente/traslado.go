package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"strings"
	"time"
)

func huellaActo(p any) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", d.ErrPeticion
	}
	defer clear(b)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func falloTrasladoSeguimiento(err error) error {
	var c d.Conflicto
	if err == nil || errors.As(err, &c) || errors.Is(err, d.ErrTraslado) || errors.Is(err, d.ErrSeguimiento) || errors.Is(err, d.ErrCierre) || errors.Is(err, d.ErrDobleIntervencion) || errors.Is(err, d.ErrCierreSPM) || errors.Is(err, d.ErrFinalizacion) || errors.Is(err, d.ErrEstado) || errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrIdempotencia) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrClave) || errors.Is(err, d.ErrPeticion) {
		return err
	}
	return d.ErrNoDisponible
}
func (s *Servicio) Trasladar(ctx context.Context, a d.Actor, id, clave string, p d.PeticionTraslado) (r d.ResultadoTraslado, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarPropuesta(ctx, a, fallo.Error()) != nil {
			r, fallo = d.ResultadoTraslado{}, d.ErrNoDisponible
		}
	}()
	if err := a.Autorizar(); err != nil {
		return r, err
	}
	if !d.IDValido(id) || p.Version < 1 {
		return r, d.ErrTraslado
	}
	if !d.ClaveValida(clave) {
		return r, d.ErrClave
	}
	if s.d.Traslados == nil || s.d.Cierres == nil || s.d.ConsultaCierreSPM == nil || s.d.EmisionesFinalizacion == nil || s.d.Familias == nil || s.d.Menores == nil || s.d.Repositorio == nil || s.d.Catalogo == nil || s.d.Identificadores == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil {
		return r, d.ErrNoDisponible
	}
	h, err := huellaActo(p)
	if err != nil {
		return r, err
	}
	ruta := "/api/expedientes/" + id + "/trasladar"
	ahora := s.d.Reloj.Ahora().Truncate(time.Microsecond)
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		fuente, err := s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		previa, repetida, err := s.d.Traslados.RepetirTraslado(ctx, a.UsuarioID, clave, ruta, h, ahora)
		if err != nil {
			return err
		}
		if repetida {
			r = previa
			return nil
		}
		permite, err := s.d.ConsultaCierreSPM.PermiteCierreOrdinarioSPM(ctx, a, id)
		if err != nil {
			return err
		}
		situacion, err := s.d.Cierres.BloquearCierre(ctx, id)
		if err != nil {
			return err
		}
		if situacion.Version != p.Version {
			return d.Conflicto{VersionActual: situacion.Version}
		}
		if err = p.ValidarDestino(situacion); err != nil {
			return err
		}
		v, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", ahora)
		if err != nil {
			return err
		}
		catalogo, err := s.d.Catalogo.Datos(ctx, v)
		if err != nil {
			return err
		}
		activo := false
		for _, e := range catalogo.Equipos {
			if e.ID == p.EquipoDestinoID && d.FechaCivil(ahora) >= d.FechaCivil(e.ActivoDesde) && (e.ActivoHasta == nil || d.FechaCivil(ahora) < d.FechaCivil(*e.ActivoHasta)) {
				activo = true
			}
		}
		if !activo {
			return d.ErrTraslado
		}
		// La copia se toma bajo el bloqueo del agregado, antes de cerrar.
		fuente, err = s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		familia, err := s.d.Familias.ObtenerFamilia(ctx, id)
		if err != nil {
			return err
		}
		menores, err := s.d.Menores.ObtenerMenores(ctx, id)
		if err != nil {
			return err
		}
		copia := d.CopiaTraslado{Datos: datosMenoresTraslado(fuente.Datos, menores.Menores), Familia: familia.Familia, Adultos: familia.Adultos, Menores: menores.Menores}
		for i := range copia.Adultos {
			copia.Adultos[i].ID, err = s.d.Identificadores.Nuevo()
			if err != nil {
				return err
			}
		}
		for i := range copia.Menores {
			copia.Menores[i].ID, err = s.d.Identificadores.Nuevo()
			if err != nil {
				return err
			}
		}
		destino, err := s.d.Identificadores.Nuevo()
		if err != nil {
			return err
		}
		lectura, err := s.d.Identificadores.Nuevo()
		if err != nil {
			return err
		}
		r.Cierre, err = s.confirmarCierre(ctx, a, id, situacion, p.PeticionCierre, permite, ahora)
		if err != nil {
			return err
		}
		if err = s.d.Traslados.CrearTraslado(ctx, id, destino, lectura, p, copia, a.UsuarioID, ahora); err != nil {
			return err
		}
		r.DestinoID, r.EquipoDestinoID, r.VersionDestino = destino, p.EquipoDestinoID, 1
		for _, recurso := range []struct{ accion, id string }{{"expediente.cerrar", id}, {"expediente.trasladar", id}, {"expediente.recibir_traslado", destino}} {
			if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, recurso.accion, recurso.id, "ok")); err != nil {
				return err
			}
		}
		return s.d.Traslados.RecordarTraslado(ctx, a.UsuarioID, clave, ruta, h, r, ahora)
	})
	return r, falloTrasladoSeguimiento(err)
}

// La derivación del destino refleja la colección vigente, incluidas sus altas
// y bajas. Los nombres aún no editados se conservan desde la lectura de origen.
func datosMenoresTraslado(datos d.DatosDerivacion, menores []d.Menor) d.DatosDerivacion {
	anteriores := map[string][]string{}
	campos := make([]d.Campo, 0, len(datos.Extraccion.Campos)+len(menores)*3)
	for _, c := range datos.Extraccion.Campos {
		if strings.HasPrefix(c.Codigo, "menores.persona.") {
			anteriores[c.Codigo] = append(anteriores[c.Codigo], c.Valor)
		} else {
			campos = append(campos, c)
		}
	}
	for i := range menores {
		m := &menores[i]
		ordenAnterior := m.Orden - 1
		valor := func(clave string) string {
			if ordenAnterior >= 0 && ordenAnterior < len(anteriores[clave]) {
				return anteriores[clave][ordenAnterior]
			}
			return ""
		}
		sinFicha := m.DatosDesdeAnexo
		if sinFicha {
			m.Nombre = valor("menores.persona.nombre")
		}
		m.Orden = i + 1
		nacimiento, escolarizacion := "", ""
		if sinFicha {
			nacimiento, escolarizacion = valor("menores.persona.nacimiento"), valor("menores.persona.escolarizacion")
		}
		if m.Nacimiento != nil {
			nacimiento = *m.Nacimiento
		}
		if m.Escolarizacion != nil {
			escolarizacion = *m.Escolarizacion
		}
		campos = append(campos,
			d.Campo{Codigo: "menores.persona.nombre", Valor: strings.TrimSpace(m.Nombre + " " + m.Apellidos)},
			d.Campo{Codigo: "menores.persona.nacimiento", Valor: nacimiento},
			d.Campo{Codigo: "menores.persona.escolarizacion", Valor: escolarizacion})
	}
	datos.Extraccion.Campos = campos
	return datos
}
