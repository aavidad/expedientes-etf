package estadistica

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type DependenciasSPAF struct {
	Catalogo      catalogo.Repositorio
	Etiquetas     paquetes.PublicacionCatalogos
	Lectura       expediente.LecturaSPAF
	Transacciones plataforma.Transacciones
	Auditoria     registro.RegistroTransaccional
	Reloj         expediente.Reloj
}

type ServicioSPAF struct{ d DependenciasSPAF }

var _ entrada.ConsultaSPAF = (*ServicioSPAF)(nil)

func NuevoSPAF(d DependenciasSPAF) *ServicioSPAF { return &ServicioSPAF{d: d} }

func (s *ServicioSPAF) ConsultarSPAF(ctx context.Context, a dominio.ActorSPAF, f dominio.FiltroSPAF) (dominio.TablaSPAF, error) {
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
	}
	ahora := s.d.Reloj.Ahora()
	f, fallo := a.AutorizarSPAF(f)
	if fallo == nil && s.d.Lectura == nil {
		fallo = dominio.ErrDisponibleSPAF
	}
	var tabla dominio.TablaSPAF
	if fallo == nil {
		corte := time.Date(f.Anio, time.Month(f.Mes)+1, 0, 0, 0, 0, 0, time.UTC)
		// Sólo meses vencidos, con fechas civiles de la zona del reloj recibido.
		y, m, d := ahora.Date()
		if !time.Date(y, m, d, 0, 0, 0, 0, time.UTC).After(corte) {
			fallo = dominio.ErrEntrada
		} else {
			fallo = s.d.Transacciones.Ejecutar(ctx, identidadSPAF(a), func(ctx context.Context) error {
				var err error
				tabla, err = s.CalcularEnTransaccion(ctx, a, f, ahora)
				if err != nil {
					return err
				}
				_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, datosAuditoriaSPAF(a, ahora, f.EquipoID, "ok"))
				return err
			})
		}
	}
	if fallo == nil {
		return tabla, nil
	}
	// El fallo se registra después de revertir; ninguna tabla parcial sale.
	resultado := "error"
	if errors.Is(fallo, dominio.ErrPermisoSPAF) {
		resultado = "denegado"
	}
	if _, err := s.d.Auditoria.Confirmar(ctx, datosAuditoriaSPAF(a, ahora, "", resultado)); err != nil {
		return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
	}
	if errors.Is(fallo, dominio.ErrPermisoSPAF) || errors.Is(fallo, dominio.ErrEntrada) {
		return dominio.TablaSPAF{}, fallo
	}
	return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
}

func identidadSPAF(a dominio.ActorSPAF) plataforma.Identidad {
	return plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
}

func datosAuditoriaSPAF(a dominio.ActorSPAF, ahora time.Time, equipo, resultado string) auditoria.Datos {
	usuario, perfiles := a.UsuarioID, a.Perfiles
	if !dominio.UUIDValidoSPAF(usuario) {
		usuario, perfiles = "sistema", nil
	}
	return auditoria.Datos{OcurridoEn: ahora.UTC().Truncate(time.Microsecond), UsuarioID: usuario, Perfiles: perfiles, Accion: "estadistica.consultar_spaf", Entidad: "estadistica.tabla_spaf", EntidadID: equipo, Resultado: resultado}
}

// CalcularEnTransaccion reutiliza las reglas de F3.04 sin abrir otra transacción.
func (s *ServicioSPAF) CalcularEnTransaccion(ctx context.Context, a dominio.ActorSPAF, f dominio.FiltroSPAF, ahora time.Time) (dominio.TablaSPAF, error) {
	f, err := a.AutorizarSPAF(f)
	if err != nil {
		return dominio.TablaSPAF{}, err
	}
	corte := time.Date(f.Anio, time.Month(f.Mes)+1, 0, 0, 0, 0, 0, time.UTC)
	y, m, d := ahora.Date()
	if !time.Date(y, m, d, 0, 0, 0, 0, time.UTC).After(corte) {
		return dominio.TablaSPAF{}, dominio.ErrEntrada
	}
	if s.d.Lectura == nil {
		return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
	}
	expedientes, reglas, err := s.d.Lectura.ConsultarSPAF(ctx, f, corte, ahora)
	if err != nil {
		return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
	}
	tabla, err := dominio.CalcularSPAF(dominio.EntradaSPAF{Filtro: f, FechaCalculo: ahora, ConocidoEn: ahora, Expedientes: expedientes, Reglas: reglas})
	if err != nil {
		return dominio.TablaSPAF{}, dominio.ErrDisponibleSPAF
	}
	return tabla, nil
}
