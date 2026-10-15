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

func (s *Servicio) RechazarSalida(ctx context.Context, a d.Actor, clave string) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	if !d.IDValido(a.UsuarioID) {
		a = d.Actor{}
	}
	resultado := "error"
	if clave == d.ErrAcceso.Error() {
		resultado = "denegado"
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.salida_servicio_rechazada", "", resultado))
	return err
}

func (s *Servicio) CrearSalidaServicio(ctx context.Context, a d.Actor, id, clave string, p d.NuevaSalidaServicio) (d.SalidaServicio, error) {
	return s.salidaServicio(ctx, a, id, clave, p, nil)
}

func (s *Servicio) ConfirmarSalidaServicio(ctx context.Context, a d.Actor, id, clave string, p d.ConfirmarSalidaServicio) (d.SalidaServicio, error) {
	return s.salidaServicio(ctx, a, id, clave, d.NuevaSalidaServicio{}, &p)
}

func (s *Servicio) salidaServicio(ctx context.Context, a d.Actor, id, clave string, p d.NuevaSalidaServicio, hito *d.ConfirmarSalidaServicio) (recibo d.SalidaServicio, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarSalida(ctx, a, fallo.Error()) != nil {
			recibo, fallo = d.SalidaServicio{}, d.ErrNoDisponible
		}
	}()
	if err := a.AutorizarSalida(""); err != nil {
		return recibo, err
	}
	if !d.IDValido(id) || !d.ClaveValida(clave) {
		return recibo, d.ErrPeticion
	}
	var contenido any = p
	version := p.Version
	ruta := "/api/expedientes/" + id + "/derivaciones-salida"
	accion := "expediente.crear_salida_servicio"
	if hito != nil {
		if err := hito.Validar(); err != nil {
			return recibo, err
		}
		contenido, version = *hito, hito.Version
		ruta, accion = "/api/expedientes/"+id+"/hitos", "expediente.confirmar_salida_servicio"
	} else if err := p.Validar(); err != nil {
		return recibo, err
	}
	if s.d.SalidasServicio == nil || s.d.Transacciones == nil || s.d.Catalogo == nil || s.d.Reloj == nil || s.d.RegistroTransaccional == nil || s.d.Identificadores == nil {
		return recibo, d.ErrNoDisponible
	}
	b, err := json.Marshal(contenido)
	if err != nil {
		return recibo, d.ErrPeticion
	}
	huella := sha256.Sum256(b)
	sha := hex.EncodeToString(huella[:])
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if hito != nil {
		zona, err := time.LoadLocation("Europe/Madrid")
		if err != nil {
			return recibo, d.ErrNoDisponible
		}
		if hito.Fecha > d.FechaCivil(ahora.In(zona)) {
			return recibo, d.ErrPeticion
		}
	}
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		r, repetida, err := s.d.SalidasServicio.RepeticionSalida(ctx, a.UsuarioID, clave, ruta, sha, ahora)
		if err != nil {
			return err
		}
		// Se comprueba el ámbito incluso al recuperar una respuesta idempotente.
		exp, err := s.d.SalidasServicio.BloquearSalida(ctx, id)
		if err != nil {
			return err
		}
		if exp.EquipoID == nil {
			return d.ErrEstado
		}
		if err := a.AutorizarSalida(*exp.EquipoID); err != nil {
			return err
		}
		if repetida {
			recibo = r
			_, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok"))
			return err
		}
		if exp.Version != version {
			return d.Conflicto{VersionActual: exp.Version}
		}
		if !d.SalidaPermitida(exp.Estado) {
			return d.ErrEstado
		}
		if hito == nil {
			v, err := s.d.Catalogo.Vigente(ctx, "listas", ahora)
			if err != nil {
				return err
			}
			cat, err := s.d.Catalogo.Datos(ctx, v)
			if err != nil {
				return err
			}
			valido, tieneDestinos := false, false
			for _, o := range cat.Opciones {
				if o.Lista == "servicio_destino" && o.Activa {
					tieneDestinos = true
					if o.Codigo == p.Servicio {
						valido = true
					}
				}
			}
			if !valido && (tieneDestinos || p.ServicioTexto == "") {
				return d.ErrPeticion
			}
			salidaID, err := s.d.Identificadores.Nuevo()
			if err != nil {
				return err
			}
			catalogoID := v.ID
			if !tieneDestinos {
				catalogoID = ""
			}
			recibo = d.SalidaServicio{ID: salidaID, ExpedienteID: id, EquipoID: *exp.EquipoID, Tipo: p.Tipo, Servicio: p.Servicio, CatalogoID: catalogoID, ServicioTexto: strings.TrimSpace(p.ServicioTexto), Motivo: p.Motivo, Estado: "borrador", Version: 1, VersionExpediente: version + 1}
			if err := s.d.SalidasServicio.CrearSalida(ctx, recibo, a.UsuarioID, ahora); err != nil {
				return err
			}
		} else {
			r, err := s.d.SalidasServicio.LeerSalida(ctx, id, hito.SalidaID)
			if err != nil {
				return err
			}
			if r.Version != hito.VersionSalida {
				return d.Conflicto{VersionActual: r.Version}
			}
			if r.Estado != "borrador" {
				return d.ErrEstado
			}
			r.Fecha, r.Estado, r.Version, r.VersionExpediente = hito.Fecha, "emitida", r.Version+1, version+1
			recibo = r
			if err := s.d.SalidasServicio.ConfirmarSalida(ctx, r, a.UsuarioID, ahora); err != nil {
				return err
			}
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, accion, id, "ok")); err != nil {
			return err
		}
		return s.d.SalidasServicio.RecordarSalida(ctx, a.UsuarioID, clave, ruta, sha, recibo, ahora)
	})
	if err != nil {
		var conflicto d.Conflicto
		if errors.As(err, &conflicto) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrPeticion) || errors.Is(err, d.ErrEstado) || errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrIdempotencia) {
			return d.SalidaServicio{}, err
		}
		return d.SalidaServicio{}, d.ErrNoDisponible
	}
	return recibo, nil
}

// ParaAnexo6 participa en la transacción de emisión y no crea otro apunte.
func (s *Servicio) ParaAnexo6(ctx context.Context, a d.Actor, id, salidaID string) (d.DatosParaDocumento, d.SalidaServicio, error) {
	if err := a.AutorizarSalida(""); err != nil {
		return d.DatosParaDocumento{}, d.SalidaServicio{}, err
	}
	if !d.IDValido(id) || !d.IDValido(salidaID) {
		return d.DatosParaDocumento{}, d.SalidaServicio{}, d.ErrPeticion
	}
	if s.d.SalidasServicio == nil {
		return d.DatosParaDocumento{}, d.SalidaServicio{}, d.ErrNoDisponible
	}
	exp, err := s.d.SalidasServicio.DatosSalida(ctx, id)
	if err != nil {
		return exp, d.SalidaServicio{}, err
	}
	if exp.EquipoID == nil || !d.SalidaPermitida(exp.Estado) {
		return exp, d.SalidaServicio{}, d.ErrEstado
	}
	if err := a.AutorizarSalida(*exp.EquipoID); err != nil {
		return d.DatosParaDocumento{}, d.SalidaServicio{}, err
	}
	salida, err := s.d.SalidasServicio.LeerSalida(ctx, id, salidaID)
	return exp, salida, err
}
