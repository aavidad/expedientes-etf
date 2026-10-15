package estadistica

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"slices"
	"strconv"
	"time"
)

type DependenciasCierre struct {
	Repositorio   salida.Cierres
	Fuente        entrada.FuenteCierreMemoria
	Transacciones plataforma.Transacciones
	Auditoria     audit.RegistroTransaccional
	Reloj         expediente.Reloj
}
type ServicioCierre struct{ d DependenciasCierre }

var _ entrada.CierresMemoria = (*ServicioCierre)(nil)

func NuevoCierre(d DependenciasCierre) *ServicioCierre { return &ServicioCierre{d} }
func (s *ServicioCierre) operar(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, accion string, coord bool, fn func(context.Context, d.FiltroMemoria, time.Time) error) error {
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil || s.d.Repositorio == nil {
		return d.ErrDisponibleMemoria
	}
	f, fallo := d.AutorizarMemoria(a, f)
	if coord && !slices.Contains(a.Perfiles, "coordinacion") {
		fallo = d.ErrPermisoMemoria
	}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}
	if !d.UUIDValidoSPAF(a.UsuarioID) {
		identidad = plataforma.Identidad{}
	}
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	memoriaID := ""
	evento := func(resultado string) auditoria.Datos {
		usuario := identidad.UsuarioID
		if usuario == "" {
			usuario = "sistema"
		}
		return auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: identidad.Perfiles, Accion: accion, Entidad: "estadistica", EntidadID: memoriaID, Resultado: resultado}
	}
	err := s.d.Transacciones.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		if fallo != nil {
			_, err := s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("denegado"))
			return err
		}
		if err := s.d.Repositorio.BloquearMemorias(ctx); err != nil {
			return err
		}
		ahora = s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
		previo, err := s.d.Repositorio.EstadoMemoria(ctx, f, false, a.UsuarioID, ahora)
		if err != nil {
			return err
		}
		memoriaID = previo.ID
		if err := fn(ctx, f, ahora); err != nil {
			return err
		}
		estado, err := s.d.Repositorio.EstadoMemoria(ctx, f, false, a.UsuarioID, ahora)
		if err != nil {
			return err
		}
		memoriaID = estado.ID
		_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("ok"))
		return err
	})
	if err != nil {
		var conflicto *d.ConflictoMemoria
		if errors.As(err, &conflicto) || errors.Is(err, d.ErrActoMemoria) || errors.Is(err, d.ErrCerradaMemoria) || errors.Is(err, d.ErrBorradorMemoria) || errors.Is(err, d.ErrIdempotenciaMemoria) || errors.Is(err, d.ErrRevisionMemoria) {
			fallo = err
		} else {
			fallo = d.ErrDisponibleMemoria
		}
		ctxAudit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.d.Transacciones.Ejecutar(ctxAudit, identidad, func(ctx context.Context) error {
			_, err := s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("error"))
			return err
		}); err != nil {
			return d.ErrDisponibleMemoria
		}
	}
	return fallo
}
func (s *ServicioCierre) RechazarCierreMemoria(ctx context.Context, a d.ActorMemoria) error {
	return s.operar(ctx, a, d.FiltroMemoria{Anio: 1}, "estadistica.memoria_rechazar", false, func(context.Context, d.FiltroMemoria, time.Time) error { return d.ErrActoMemoria })
}
func (s *ServicioCierre) EstadoMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (r d.EstadoMemoria, err error) {
	err = s.operar(ctx, a, f, "estadistica.memoria_estado", false, func(ctx context.Context, f d.FiltroMemoria, t time.Time) error {
		var err error
		r, err = s.d.Repositorio.EstadoMemoria(ctx, f, false, a.UsuarioID, t)
		return err
	})
	if err != nil {
		return d.EstadoMemoria{}, err
	}
	return r, nil
}
func (s *ServicioCierre) InstantaneaMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, revision int) (r d.RevisionMemoria, err error) {
	err = s.operar(ctx, a, f, "estadistica.memoria_instantanea", false, func(ctx context.Context, f d.FiltroMemoria, _ time.Time) error {
		if revision < 0 {
			return d.ErrActoMemoria
		}
		var err error
		r, err = s.d.Repositorio.LeerRevision(ctx, f, revision)
		return err
	})
	if err != nil {
		return d.RevisionMemoria{}, err
	}
	return r, nil
}
func (s *ServicioCierre) CerrarMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, clave string, p d.ActoMemoria) (d.ResultadoActoMemoria, error) {
	return s.acto(ctx, a, f, clave, p, false)
}
func (s *ServicioCierre) ReabrirMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, clave string, p d.ActoMemoria) (d.ResultadoActoMemoria, error) {
	return s.acto(ctx, a, f, clave, p, true)
}
func (s *ServicioCierre) acto(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, clave string, p d.ActoMemoria, reabrir bool) (r d.ResultadoActoMemoria, err error) {
	accion := "cerrar"
	if reabrir {
		accion = "reabrir"
	}
	err = s.operar(ctx, a, f, "estadistica.memoria_"+accion, reabrir, func(ctx context.Context, f d.FiltroMemoria, ahora time.Time) error {
		if err := p.Validar(clave, reabrir); err != nil {
			return err
		}
		ruta := "/api/estadistica/memoria/" + accion + "?anio=" + strconv.Itoa(f.Anio) + "&etf=" + f.EquipoID
		b, err := json.Marshal(p)
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		clear(b)
		huella := hex.EncodeToString(h[:])
		repetida := false
		r, repetida, err = s.d.Repositorio.RepetirActo(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil || repetida {
			return err
		}
		e, err := s.d.Repositorio.EstadoMemoria(ctx, f, false, a.UsuarioID, ahora)
		if err != nil {
			return err
		}
		if e.Version != p.Version {
			return &d.ConflictoMemoria{Actual: e.Version}
		}
		var revision *d.RevisionMemoria
		var bases []d.InformeMemoria
		if reabrir {
			if e.Estado != "cerrada" {
				return d.ErrBorradorMemoria
			}
		} else {
			if e.Estado == "cerrada" {
				return d.ErrCerradaMemoria
			}
			if s.d.Fuente == nil {
				return d.ErrDisponibleMemoria
			}
			informe, err := s.d.Fuente.MemoriaEnTransaccion(ctx, a, f, ahora)
			if err != nil {
				return err
			}
			huella, err := d.HuellaMemoria(informe)
			if err != nil {
				return err
			}
			revision = &d.RevisionMemoria{Informe: informe, Motor: d.VersionMotorMemoria, Huella: huella}
			equipos := []string{f.EquipoID}
			if f.EquipoID == "" {
				equipos, err = s.d.Fuente.EquiposMemoria(ctx)
				if err != nil {
					return err
				}
			}
			for _, equipo := range equipos {
				base := informe
				if f.EquipoID == "" {
					base.EquipoID = equipo
					base, err = s.d.Fuente.MemoriaFijadaEnTransaccion(ctx, a, base, ahora)
					if err != nil {
						return err
					}
				}
				base.Narrativas = d.Narrativas{}
				base.Composicion = nil
				base.Fuentes = nil
				base.Desglose = nil
				bases = append(bases, base)
			}
		}
		r, err = s.d.Repositorio.GuardarActo(ctx, e, p, revision, bases, a.UsuarioID, clave, ruta, huella, ahora)
		return err
	})
	if err != nil {
		return d.ResultadoActoMemoria{}, err
	}
	return r, nil
}
func (s *ServicioCierre) DiscrepanciasMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (r d.DiscrepanciasMemoria, err error) {
	r = d.DiscrepanciasMemoria{Avisos: []d.DiscrepanciaMemoria{}, Diferencias: []d.DiferenciaMemoria{}}
	err = s.operar(ctx, a, f, "estadistica.memoria_discrepancias", true, func(ctx context.Context, f d.FiltroMemoria, ahora time.Time) error {
		var err error
		r.Avisos, err = s.d.Repositorio.Discrepancias(ctx, f)
		if err != nil {
			return err
		}
		if s.d.Fuente == nil {
			return d.ErrDisponibleMemoria
		}
		revisiones, err := s.d.Repositorio.RevisionesCerradas(ctx, f)
		if err != nil {
			return err
		}
		for _, revision := range revisiones {
			actual, err := s.d.Fuente.MemoriaFijadaEnTransaccion(ctx, a, revision.Informe, ahora)
			if err != nil {
				return err
			}
			codigos := d.ApartadosDiferentes(revision.Informe, actual)
			if len(codigos) == 0 {
				continue
			}
			diferencia := d.DiferenciaMemoria{FiltroMemoria: actual.FiltroMemoria, Revision: revision.Revision, Apartados: []d.DiferenciaApartadoMemoria{}}
			for _, codigo := range codigos {
				v := d.DiferenciaApartadoMemoria{Codigo: codigo}
				for _, ap := range revision.Informe.Apartados {
					if ap.Codigo == codigo {
						v.Cerrado = ap
					}
				}
				for _, ap := range actual.Apartados {
					if ap.Codigo == codigo {
						v.Actual = ap
					}
				}
				diferencia.Apartados = append(diferencia.Apartados, v)
			}
			r.Diferencias = append(r.Diferencias, diferencia)
		}
		return nil
	})
	if err != nil {
		return d.DiscrepanciasMemoria{}, err
	}
	return r, nil
}
