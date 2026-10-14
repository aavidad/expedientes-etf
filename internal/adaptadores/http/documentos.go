package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

type ConfiguracionDocumentos struct {
	Servicio          entrada.Emisiones
	Registros         entrada.RegistrosSalida
	Plantillas        entrada.Plantillas
	GestionPlantillas entrada.GestionPlantillas
	SPAF              entrada.EmisionesSPAF
}

func registrarDocumentos(mux *muxHTTP, c ConfiguracionDocumentos) {
	registrarRegistroSalida(mux, c)
	registrarPlantillas(mux, c)
	registrarGestionPlantillas(mux, c.GestionPlantillas)
	registrarSPAFWord(mux, c.SPAF)
	plazas := make(chan struct{}, 2)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if c.Servicio == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		var actor entrada.Actor
		rechazar := func(err error) {
			if fallo := c.Servicio.Rechazar(ctx, actor, err.Error()); fallo != nil {
				errorDocumento(w, entrada.ErrNoDisponible)
				return
			}
			errorDocumento(w, err)
		}
		// El middleware ya consultó la sesión vigente; ningún dato del cliente
		// puede aportar identidad, perfiles ni pertenencia al equipo.
		acceso := identidad.AccesoActual(r.Context())
		actor = entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, perfil := range acceso.Usuario.Perfiles {
			actor.Perfiles = append(actor.Perfiles, string(perfil))
		}
		tipoDocumento := ""
		switch r.Pattern {
		case "/api/expedientes/{id}/documentos/acta_cierre":
			tipoDocumento = entrada.ActaCierre
		case "/api/expedientes/{id}/documentos/anexo6":
			tipoDocumento = "salida/anexo6"
		case "/api/expedientes/{id}/documentos/anexo1":
			tipoDocumento = entrada.Anexo1
		case "/api/expedientes/{id}/documentos/anexo2":
			tipoDocumento = entrada.Anexo2
		}
		if r.PathValue("anexo") != "" {
			tipoDocumento = "finalizacion/" + r.PathValue("anexo")
		}
		if !entrada.IDActorEmisionValido(actor) {
			rechazar(entrada.ErrAcceso)
			return
		}
		if tipoDocumento != "" {
			if err := actor.AutorizarTipo(tipoDocumento); err != nil {
				rechazar(err)
				return
			}
		}

		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(entrada.ErrPeticion)
			return
		}
		if len(r.Header.Values("Idempotency-Key")) != 1 {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.Solicitud
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		var cuerpo any = &p
		var idioma struct {
			Idioma string `json:"idioma"`
		}
		if tipoDocumento == entrada.Anexo1 || tipoDocumento == entrada.Anexo2 {
			cuerpo = &idioma
		}
		if err := dec.Decode(cuerpo); err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		if tipoDocumento == entrada.Anexo1 || tipoDocumento == entrada.Anexo2 {
			p.Tipo, p.ExpedienteID, p.Idioma = tipoDocumento, r.PathValue("id"), idioma.Idioma
		} else if tipoDocumento != "" {
			if p.Tipo != "" || p.ExpedienteID != "" {
				rechazar(entrada.ErrPeticion)
				return
			}
			p.ExpedienteID = r.PathValue("id")
			p.Tipo = tipoDocumento
		}
		p.Clave = r.Header.Get("Idempotency-Key")
		if err := actor.AutorizarTipo(p.Tipo); err != nil {
			rechazar(err)
			return
		}
		if err := p.Validar(); err != nil {
			rechazar(err)
			return
		}
		select {
		case plazas <- struct{}{}:
			defer func() { <-plazas }()
		case <-ctx.Done():
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		resultado, err := c.Servicio.Emitir(ctx, actor, p)
		if err != nil {
			errorDocumento(w, err)
			return
		}
		defer clear(resultado.Contenido)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="documento.docx"`)
		w.Header().Set("X-Emision-ID", resultado.Emision.ID)
		w.Header().Set("X-Documento-SHA256", resultado.Emision.HuellaResultado)
		w.Header().Set("X-Datos-SHA256", resultado.Emision.HuellaDatos)
		w.Header().Set("X-Plantilla-Version", resultado.Emision.Plantilla.Version)
		if resultado.Emision.Aviso != "" {
			w.Header().Set("X-Aviso-Clave", resultado.Emision.Aviso)
		}
		if resultado.Repetida {
			w.Header().Set("X-Idempotencia-Repetida", "true")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write(resultado.Contenido)
	})
	mux.declarar(handler,
		reglaEmisionSPM(),
		escritura("/api/expedientes/{id}/documentos/acta_cierre", []string{http.MethodPost}, "coordinacion"),
		escritura("/api/expedientes/{id}/documentos/anexo6", []string{http.MethodPost}, "coordinacion", "profesional_etf"),
		escritura("/api/expedientes/{id}/documentos/anexo1", []string{http.MethodPost}, "coordinacion", "profesional_etf"),
		escritura("/api/expedientes/{id}/documentos/anexo2", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
	for _, anexo := range []string{"anexo11", "anexo12"} {
		regla := escritura("/api/expedientes/{id}/documentos/"+anexo, []string{http.MethodPost}, "profesional_etf")
		regla.ExigeAmbito = true
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.SetPathValue("anexo", anexo); handler.ServeHTTP(w, r) }), regla)
	}
}
func errorDocumento(w http.ResponseWriter, err error) {
	var detalle *entrada.DetalleMarcadores
	if errors.As(err, &detalle) {
		responderJSON(w, 422, map[string]any{"error": "documento.plantilla_marcadores", "antivirus": "limpio", "marcadores": detalle})
		return
	}
	if errors.Is(err, entrada.ErrMarcadores) {
		responderJSON(w, 422, map[string]any{"error": "documento.plantilla_marcadores", "antivirus": "limpio"})
		return
	}
	var plantilla *entrada.ConflictoPlantilla
	if errors.As(err, &plantilla) {
		responderJSON(w, 409, map[string]any{"error": plantilla.Error(), "version_actual": plantilla.VersionActual})
		return
	}
	var version entrada.ConflictoVersion
	if errors.As(err, &version) {
		responderJSON(w, 409, map[string]any{"error": map[string]any{"clave": "error.conflicto_version", "parametros": map[string]any{"version_actual": version.VersionActual}}})
		return
	}
	var registro *entrada.ConflictoRegistro
	if errors.As(err, &registro) {
		responderJSON(w, 409, map[string]any{"error": registro.Error(), "version_actual": registro.VersionActual})
		return
	}
	var conflicto *entrada.ConflictoReproduccion
	if errors.As(err, &conflicto) {
		responderJSON(w, 409, map[string]any{"error": conflicto.Error(), "emision": conflicto.Emision})
		return
	}
	estado, clave := 503, "documento.no_disponible"
	switch {
	case errors.Is(err, entrada.ErrTamanoPlantilla):
		estado, clave = 413, "documento.plantilla_tamano"
	case errors.Is(err, entrada.ErrMarcadores):
		estado, clave = 422, "documento.plantilla_marcadores"
	case errors.Is(err, entrada.ErrAntivirus):
		estado, clave = 503, "documento.plantilla_antivirus"
	case errors.Is(err, entrada.ErrInfectada):
		estado, clave = 422, "documento.plantilla_infectada"
	case errors.Is(err, entrada.ErrVersionPlantilla):
		estado, clave = 409, "documento.plantilla_version_usada"
	case errors.Is(err, entrada.ErrNoAceptada):
		estado, clave = 409, "documento.requiere_no_aceptada"
	case errors.Is(err, entrada.ErrListaEspera):
		estado, clave = 409, "documento.requiere_lista_espera"
	case errors.Is(err, entrada.ErrRegistroOficio):
		estado, clave = 422, "documento.registro_solo_oficios"
	case errors.Is(err, entrada.ErrAcceso):
		estado, clave = 403, "documento.acceso_denegado"
	case errors.Is(err, entrada.ErrPeticion):
		estado, clave = 400, "documento.peticion_invalida"
	case errors.Is(err, entrada.ErrPlantilla):
		estado, clave = 422, "documento.plantilla_no_disponible"
	case errors.Is(err, entrada.ErrHuella):
		estado, clave = 409, "documento.plantilla_alterada"
	case errors.Is(err, entrada.ErrFirmante):
		estado, clave = 422, "documento.firmante_no_disponible"
	case errors.Is(err, entrada.ErrIdempotencia):
		estado, clave = 422, "error.idempotencia_reutilizada"
	case err.Error() == "identidad.csrf_no_valido":
		estado, clave = 403, "identidad.csrf_no_valido"
	}
	responderError(w, estado, clave)
}

func reglaEmisionSPM() reglaRuta {
	r := escritura("/api/documentos/emisiones", []string{http.MethodPost}, "coordinacion", "profesional_etf")
	r.ExigeAmbito = true
	return r
}
