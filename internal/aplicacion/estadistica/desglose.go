package estadistica

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

// DesglosarMemoria mantiene la autorización y la auditoría en la lectura de la foto.
func (s ServicioNavegacion) DesglosarMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ap d.Apartado, seleccion d.SeleccionDesglose) (r d.DetalleMemoria, err error) {
	if s.Transacciones == nil || s.Auditoria == nil || s.Reloj == nil || s.Fuente == nil || s.Numeros == nil {
		return r, d.ErrDisponibleMemoria
	}
	ahora := s.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	f, err = d.AutorizarMemoria(a, f)
	if err == nil && (f.Anio > ahora.Year() || seleccion.Revision < 0) {
		err = d.ErrEntrada
	}
	evento := func(resultado string) auditoria.Datos {
		usuario, perfiles := a.UsuarioID, a.Perfiles
		if !d.UUIDValidoSPAF(usuario) {
			usuario, perfiles = "sistema", nil
		}
		return auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: perfiles, Accion: "estadistica.memoria_desglose", Entidad: "estadistica", EntidadID: f.EquipoID, Resultado: resultado}
	}
	if err == nil {
		err = s.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			var informe d.InformeMemoria
			var e error
			definitiva := false
			if seleccion.Revision > 0 {
				if s.Cierres == nil {
					return d.ErrDisponibleMemoria
				}
				foto, e := s.Cierres.LeerRevision(ctx, f, seleccion.Revision)
				if e != nil {
					return e
				}
				informe = foto.Informe
				definitiva = true
			} else {
				if s.Cierres != nil {
					if e := s.Cierres.BloquearMemorias(ctx); e != nil {
						return e
					}
					// Un cierre puede haberse confirmado durante la espera del bloqueo.
					ahora = s.Reloj.Ahora().UTC().Truncate(time.Microsecond)
					foto, cerrada, e := s.Cierres.RevisionConocida(ctx, f, ahora)
					if e != nil {
						return e
					}
					if cerrada {
						informe, definitiva = foto.Informe, true
					}
				}
				if !definitiva {
					informe, e = s.Fuente.MemoriaEnTransaccion(ctx, a, f, ahora)
					if e != nil {
						return e
					}
				}
			}
			if informe.FiltroMemoria != f {
				return d.ErrPermisoMemoria
			}
			r, e = d.SeleccionarDesglose(informe, ap, seleccion)
			if e != nil {
				return e
			}
			r.Definitivas = definitiva
			ids := []string{}
			for _, fila := range r.Filas {
				if f.EquipoID != "" && fila.EquipoID != f.EquipoID {
					return d.ErrPermisoMemoria
				}
				ids = append(ids, fila.ExpedienteID)
			}
			// La foto aporta el número histórico; la lectura conjunta sólo comprueba
			// el acceso actual a la ficha, con RLS y sin leer datos personales.
			numeros, e := s.Numeros.NumerosMemoria(ctx, ids)
			if e != nil {
				return e
			}
			for i := range r.Filas {
				_, r.Filas[i].Ficha = numeros[r.Filas[i].ExpedienteID]
			}
			_, e = s.Auditoria.RegistrarEnTransaccion(ctx, evento("ok"))
			return e
		})
	}
	if err == nil {
		return r, nil
	}
	resultado := "error"
	if errors.Is(err, d.ErrPermisoMemoria) || errors.Is(err, d.ErrEntrada) {
		resultado = "denegado"
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if _, e := s.Auditoria.Confirmar(auditCtx, evento(resultado)); e != nil {
		return d.DetalleMemoria{}, d.ErrDisponibleMemoria
	}
	if errors.Is(err, d.ErrPermisoMemoria) || errors.Is(err, d.ErrEntrada) || errors.Is(err, d.ErrRevisionMemoria) {
		return d.DetalleMemoria{}, err
	}
	return d.DetalleMemoria{}, d.ErrDisponibleMemoria
}
