package auditoria

import (
	"errors"
	"strings"
	"time"
)

const MaxEventosExportacion = 100000

type ActorExportacion struct {
	UsuarioID string
	Perfiles  []string
}

func (a ActorExportacion) Autorizar() error {
	if !identificadorValido(a.UsuarioID) {
		return errors.New("auditoria.permiso_denegado")
	}
	for _, p := range a.Perfiles {
		if p == "administracion" {
			return nil
		}
	}
	return errors.New("auditoria.permiso_denegado")
}

// Hasta es exclusivo. Las fechas se interpretan como días UTC completos.
type FiltroExportacion struct {
	Desde        time.Time `json:"desde"`
	Hasta        time.Time `json:"hasta"`
	TipoEvento   string    `json:"tipo_evento"`
	Accion       string    `json:"accion"`
	UsuarioID    string    `json:"usuario_id"`
	ExpedienteID string    `json:"expediente_id"`
}

func (f FiltroExportacion) Validar() error {
	if f.TipoEvento != "" {
		valido := false
		for _, tipo := range []string{"identidad", "catalogo", "paquete", "expediente", "evaluacion", "documento", "actividad", "estadistica", "auditoria"} {
			if f.TipoEvento == tipo {
				valido = true
			}
		}
		if !valido {
			return errors.New("auditoria.filtros_invalidos")
		}
	}

	if f.Desde.IsZero() || f.Hasta.IsZero() || !f.Hasta.After(f.Desde) || f.Hasta.Sub(f.Desde) > 31*24*time.Hour || f.Desde.Year() < 1 || f.Hasta.Year() > 9999 {
		return errors.New("auditoria.fechas_invalidas")
	}
	if f.Accion != "" && !codigoValido(f.Accion) || f.UsuarioID != "" && f.UsuarioID != "sistema" && !identificadorValido(f.UsuarioID) || f.ExpedienteID != "" && !identificadorValido(f.ExpedienteID) {
		return errors.New("auditoria.filtros_invalidos")
	}
	return nil
}

func (f FiltroExportacion) Incluye(e Evento) bool {
	return (f.TipoEvento == "" || e.Entidad == f.TipoEvento || strings.HasPrefix(e.Entidad, f.TipoEvento+".")) && !e.OcurridoEn.Before(f.Desde) && e.OcurridoEn.Before(f.Hasta) && (f.Accion == "" || e.Accion == f.Accion) && (f.UsuarioID == "" || e.UsuarioID == f.UsuarioID) && (f.ExpedienteID == "" || e.ExpedienteID == f.ExpedienteID)
}

type TramoExportacion struct {
	Anterior Punto `json:"anterior"`
	Final    Punto `json:"final"`
}

type ManifiestoExportacion struct {
	Version        string            `json:"version"`
	Formato        string            `json:"formato"`
	Filtro         FiltroExportacion `json:"filtro"`
	Tramo          TramoExportacion  `json:"tramo"`
	Filas          int               `json:"filas"`
	EventosCadena  int               `json:"eventos_cadena"`
	HuellaRegistro string            `json:"huella_registro"`
	HuellaCadena   string            `json:"huella_cadena"`
}
