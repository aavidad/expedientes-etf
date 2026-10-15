package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
)

type TextosAuditoria interface {
	Textos(context.Context, string) (map[string]string, error)
}
type ConfiguracionAuditoria struct {
	Servicio entrada.Exportador
	Textos   TextosAuditoria
	Registro *slog.Logger
}
type ModuloAuditoria struct{ Configuracion ConfiguracionAuditoria }

func (m ModuloAuditoria) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarExportacionAuditoria(mux, m.Configuracion) }, pantalla("/admin/auditoria", "auditoria.titulo", "administracion")), "/admin/auditoria")
}

func registrarExportacionAuditoria(mux *muxHTTP, c ConfiguracionAuditoria) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceso := identidad.AccesoActual(r.Context())
		if !acceso.Permite("administracion") {
			responderError(w, 403, "auditoria.permiso_denegado")
			return
		}
		if c.Servicio == nil {
			responderError(w, 503, "auditoria.no_disponible")
			return
		}
		actor := entrada.ActorExportacion{UsuarioID: acceso.Usuario.ID}
		for _, v := range acceso.Usuario.Perfiles {
			actor.Perfiles = append(actor.Perfiles, string(v))
		}
		rechazar := func(estado int, clave string) {
			if c.Servicio.Rechazar(r.Context(), actor) != nil {
				responderError(w, 503, "auditoria.no_disponible")
				return
			}
			responderError(w, estado, clave)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			rechazar(415, "auditoria.filtros_invalidos")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		defer r.Body.Close()
		var p struct {
			TipoEvento   string `json:"tipo_evento"`
			Desde        string `json:"desde"`
			Hasta        string `json:"hasta"`
			Accion       string `json:"accion"`
			UsuarioID    string `json:"usuario_id"`
			ExpedienteID string `json:"expediente_id"`
			Formato      string `json:"formato"`
			Idioma       string `json:"idioma"`
		}
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF {
			rechazar(400, "auditoria.filtros_invalidos")
			return
		}
		desde, e1 := time.Parse("2006-01-02", p.Desde)
		hasta, e2 := time.Parse("2006-01-02", p.Hasta)
		if e1 != nil || e2 != nil {
			rechazar(400, "auditoria.fechas_invalidas")
			return
		}
		instrucciones, err := textoAuditoria(r.Context(), c.Textos, p.Idioma)
		if err != nil {
			rechazar(503, "auditoria.no_disponible")
			return
		}

		ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancelar()
		resultado, err := c.Servicio.Exportar(ctx, actor, entrada.FiltroExportacion{TipoEvento: p.TipoEvento, Desde: desde, Hasta: hasta.AddDate(0, 0, 1), Accion: p.Accion, UsuarioID: p.UsuarioID, ExpedienteID: p.ExpedienteID}, p.Formato, instrucciones)
		if err != nil {
			estado := http.StatusServiceUnavailable
			clave := "auditoria.no_disponible"
			switch err.Error() {
			case "auditoria.permiso_denegado":
				clave = err.Error()
				estado = 403
			case "auditoria.fechas_invalidas", "auditoria.filtros_invalidos", "auditoria.formato_invalido":
				estado = 400
				clave = err.Error()
			case "auditoria.exportacion_demasiado_grande":
				estado = 413
				clave = err.Error()
			case "auditoria.ocupada":
				estado = 429
				clave = err.Error()
			case "auditoria.cadena_alterada":
				clave = err.Error()
			}
			responderError(w, estado, clave)
			return
		}
		defer resultado.Contenido.Close()
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="auditoria.zip"`)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", strconv.FormatInt(resultado.Tamano, 10))
		w.Header().Set("X-Auditoria-Filas", strconv.Itoa(resultado.Filas))
		w.Header().Set("X-Auditoria-Huella", resultado.Final.Huella)
		// El plazo de escritura limita también a clientes que dejan de leer.
		controlador := http.NewResponseController(w)
		_ = controlador.SetWriteDeadline(time.Now().Add(30 * time.Second))
		if _, err = io.Copy(w, resultado.Contenido); err != nil {
			if fallo := c.Servicio.Rechazar(r.Context(), actor); fallo != nil && c.Registro != nil {
				c.Registro.ErrorContext(context.WithoutCancel(r.Context()), "auditoria.registro_fallido")
			}
			return
		}
	}), escritura("/api/admin/auditoria/exportaciones", []string{http.MethodPost}, "administracion"))
}

func textoAuditoria(ctx context.Context, lector TextosAuditoria, idioma string) (string, error) {
	if lector == nil || idioma == "" {
		return "", errors.New("auditoria.filtros_invalidos")
	}
	textos, err := lector.Textos(ctx, idioma)
	if err != nil {
		return "", err
	}
	texto := textos["auditoria.instrucciones_verificacion"]
	if texto == "" {
		return "", errors.New("auditoria.no_disponible")
	}
	return texto, nil
}
