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
	"strconv"
	"strings"
	"time"
)

func registrarSPAFWord(mux *muxHTTP, servicio entrada.EmisionesSPAF) {
	plazas := make(chan struct{}, 2)
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if servicio == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		acceso := identidad.AccesoActual(ctx)
		actor := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, perfil := range acceso.Usuario.Perfiles {
			actor.Perfiles = append(actor.Perfiles, string(perfil))
		}
		rechazar := func(err error) {
			if servicio.RechazarSPAF(ctx, actor) != nil {
				errorDocumento(w, entrada.ErrNoDisponible)
				return
			}
			errorDocumento(w, err)
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" || len(r.Header.Values("Idempotency-Key")) != 1 || r.URL.RawQuery != "" {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.SolicitudSPAF
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		p.EquipoID = r.PathValue("etf")
		p.Clave = r.Header.Get("Idempotency-Key")
		p.Anio, err = strconv.Atoi(r.PathValue("anio"))
		if err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		mes, conSufijo := strings.CutSuffix(r.PathValue("mes"), ".docx")
		if !conSufijo {
			rechazar(entrada.ErrPeticion)
			return
		}
		p.Mes, err = strconv.Atoi(mes)
		if err != nil {
			rechazar(entrada.ErrPeticion)
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
		resultado, err := servicio.EmitirSPAF(ctx, actor, p)
		if err != nil {
			var conflicto *entrada.ConflictoSPAF
			if errors.As(err, &conflicto) {
				responderJSON(w, 409, map[string]any{"error": conflicto.Error(), "version_actual": conflicto.Actual})
				return
			}
			errorDocumento(w, err)
			return
		}
		defer clear(resultado.Contenido)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="spaf.docx"`)
		w.Header().Set("X-Emision-ID", resultado.Emision.ID)
		w.Header().Set("X-Documento-SHA256", resultado.Emision.HuellaResultado)
		w.Header().Set("X-Datos-SHA256", resultado.Emision.HuellaDatos)
		w.Header().Set("X-Plantilla-Version", resultado.Emision.Plantilla.Version)
		if resultado.Emision.Aviso != "" {
			w.Header().Set("X-Aviso-Clave", resultado.Emision.Aviso)
		}
		estado := http.StatusCreated
		if resultado.Repetida {
			estado = http.StatusOK
			w.Header().Set("X-Idempotencia-Repetida", "true")
		}
		w.WriteHeader(estado)
		_, _ = w.Write(resultado.Contenido)
	}), reglaRuta{Patron: "/api/equipos/{etf}/tabla-spaf/{anio}/{mes}", Metodos: []string{http.MethodPost}, Perfiles: []string{"coordinacion", "profesional_etf"}, ExigeAmbito: true, CSRF: true})
}
