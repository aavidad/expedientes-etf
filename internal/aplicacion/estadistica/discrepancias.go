package estadistica

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"slices"
	"time"
)

// RegistroConAvisos mantiene el adaptador de auditoría genérico. Las
// correcciones, su apunte y sus avisos se confirman o revierten juntos.
// El bloqueo de Memoria se toma antes del de auditoría, también al cerrar.
type RegistroConAvisos struct {
	Registro    audit.RegistroTransaccional
	Repositorio salida.Cierres
	Fuente      entrada.FuenteCierreMemoria
	Equipos     expediente.EquipoMemoria
}

var _ audit.RegistroTransaccional = RegistroConAvisos{}

func (s RegistroConAvisos) Confirmar(ctx context.Context, datos auditoria.Datos) (auditoria.Evento, error) {
	return s.Registro.Confirmar(ctx, datos)
}
func afectaMemoria(datos auditoria.Datos) bool {
	if datos.Resultado != "ok" || datos.ExpedienteID == "" {
		return false
	}
	switch datos.Accion {
	case "expediente.dejar_lista_espera", "expediente.cargado_para_revision", "expediente.confirmar_derivacion", "expediente.editar_adultos", "expediente.confirmar_carga_historica", "expediente.confirmar_salida_servicio", "expediente.no_aceptar_derivacion", "expediente.editar_familia", "expediente.editar_menores", "expediente.editar_intervencion", "expediente.aceptar_derivacion", "expediente.confirmar_hito", "expediente.cerrar", "expediente.reabrir", "expediente.trasladar", "expediente.no_aceptar", "expediente.cargar_historia":
		return true
	}
	return false
}
func (s RegistroConAvisos) RegistrarEnTransaccion(ctx context.Context, datos auditoria.Datos) (auditoria.Evento, error) {
	if !afectaMemoria(datos) {
		return s.Registro.RegistrarEnTransaccion(ctx, datos)
	}
	if s.Repositorio == nil || s.Fuente == nil || s.Equipos == nil {
		return auditoria.Evento{}, d.ErrDisponibleMemoria
	}
	if err := s.Repositorio.BloquearMemorias(ctx); err != nil {
		return auditoria.Evento{}, err
	}
	equipo, modificado, err := s.Equipos.EquipoMemoria(ctx, datos.ExpedienteID)
	if err != nil {
		return auditoria.Evento{}, err
	}
	equipos := []string{}
	if equipo != "" {
		equipos = append(equipos, equipo)
	}
	if lector, ok := s.Equipos.(expediente.EquiposMemoriaAfectados); ok {
		equipos, modificado, err = lector.EquiposAfectadosMemoria(ctx, datos.ExpedienteID)
		if err != nil {
			return auditoria.Evento{}, err
		}
		equipos = slices.Compact(equipos)
	}
	evento, err := s.Registro.RegistrarEnTransaccion(ctx, datos)
	if err != nil || len(equipos) == 0 {
		return evento, err
	}
	vistos := map[string]bool{}
	for _, equipo := range equipos {
		bases, err := s.Repositorio.BasesEquipo(ctx, equipo)
		if err != nil {
			return auditoria.Evento{}, err
		}
		a := d.ActorMemoria{UsuarioID: datos.UsuarioID, Perfiles: datos.Perfiles, EquipoID: equipo}
		for _, base := range bases {
			claveBase := base.MemoriaID + ":" + equipo
			if vistos[claveBase] {
				continue
			}
			vistos[claveBase] = true
			actual, err := s.Fuente.MemoriaFijadaEnTransaccion(ctx, a, base.Informe, datos.OcurridoEn)
			if err != nil {
				return auditoria.Evento{}, err
			}
			anterior, err := s.Fuente.MemoriaFijadaEnTransaccion(ctx, a, base.Informe, modificado.Add(-time.Microsecond))
			if err != nil {
				return auditoria.Evento{}, err
			}
			afectados := d.ApartadosDiferentes(anterior, actual)
			apartados := d.ApartadosDiferentes(base.Informe, actual)
			apartados = slices.DeleteFunc(apartados, func(ap d.Apartado) bool { return !slices.Contains(afectados, ap) })
			if len(apartados) == 0 {
				continue
			}
			dif := d.DiscrepanciaMemoria{MemoriaID: base.MemoriaID, Revision: base.Revision, EquipoID: equipo, ExpedienteID: datos.ExpedienteID, EventoID: evento.ID, Apartados: apartados, DetectadaEn: datos.OcurridoEn}
			if err := s.Repositorio.AvisarDiscrepancia(ctx, dif); err != nil {
				return auditoria.Evento{}, err
			}
			_, err = s.Registro.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: datos.OcurridoEn, UsuarioID: datos.UsuarioID, Perfiles: datos.Perfiles, Accion: "estadistica.memoria_discrepancia", Entidad: "estadistica", EntidadID: base.MemoriaID, ExpedienteID: datos.ExpedienteID, Resultado: "ok"})
			if err != nil {
				return auditoria.Evento{}, err
			}
		}
	}
	return evento, nil
}

// FuenteConCierre entrega siempre la revisión cerrada al Word. Generar un
// documento no altera el estado ni solicita un recálculo de la foto.
type FuenteConCierre struct {
	Fuente      entrada.FuenteMemoria
	Repositorio salida.Cierres
}

func (s FuenteConCierre) MemoriaEnTransaccion(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.InformeMemoria, error) {
	f, err := d.AutorizarMemoria(a, f)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	if err := s.Repositorio.BloquearMemorias(ctx); err != nil {
		return d.InformeMemoria{}, err
	}
	revision, cerrada, err := s.Repositorio.RevisionConocida(ctx, f, ahora)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	if cerrada {
		return revision.Informe, nil
	}
	return s.Fuente.MemoriaEnTransaccion(ctx, a, f, ahora)
}
