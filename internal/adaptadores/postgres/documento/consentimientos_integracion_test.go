package documento_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expedienteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paqueteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	expedienteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entradaIdentidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	dominioExpediente "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type relojConsentimientos struct{}

func (relojConsentimientos) Ahora() time.Time {
	return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
}

type sesionesConsentimientos map[string]identidad.Usuario

func (sesionesConsentimientos) AdmitirPeticion(context.Context) error { return nil }
func (sesionesConsentimientos) Entrar(context.Context, entradaIdentidad.Inicio) (entradaIdentidad.SesionAbierta, error) {
	return entradaIdentidad.SesionAbierta{}, identidad.ErrSesionNoValida
}
func (s sesionesConsentimientos) Consultar(_ context.Context, clave string) (entradaIdentidad.Acceso, error) {
	u, ok := s[clave]
	if !ok {
		return entradaIdentidad.Acceso{}, identidad.ErrSesionNoValida
	}
	return entradaIdentidad.Acceso{Estado: identidad.Activo, Usuario: u}, nil
}
func (sesionesConsentimientos) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesionesConsentimientos) Salir(context.Context, string) error                    { return nil }

type estadoConsentimiento struct {
	Estado                             string
	Version, Hitos, Cambios, Emisiones int
	FechaAceptacion                    string
}

func TestConsentimientosPostgresDescargaTLSFirmaSeparadaYPermisos(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	coordinacion := plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	var equipo string
	if err := b.Ejecutar(ctx, coordinacion, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT equipo_id::text FROM expediente.expediente WHERE id='a2030000-0000-4000-8000-000000000001'`).Scan(&equipo)
	}); err != nil {
		t.Fatal(err)
	}
	if equipo == "" {
		t.Fatal("el fixture debe tener equipo real")
	}
	registro := postgres.NuevoRegistroAuditoria(b)
	raizIdiomas := os.Getenv("ETF_DOCUMENTO_IDIOMAS_RUTA")
	if raizIdiomas == "" {
		raizIdiomas = "../../../../idiomas"
	}
	archivos := ficheros.Plantillas{Raiz: raizIdiomas, Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	repositorioExpedientes := &expedienteRepo.Repositorio{}
	reloj := relojConsentimientos{}
	documentos := app.Nuevo(app.Dependencias{
		Repositorio: repo.Repositorio{}, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}},
		Datos: app.DatosEntrada{Expedientes: expedienteApp.ConsultaDocumentos{Repositorio: repositorioExpedientes},
			Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: reloj},
		Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b,
		Registro: registro, RegistroTransaccional: registro, Reloj: reloj, Identificadores: ids{},
	})
	hitos := expedienteApp.Nuevo(expedienteApp.Dependencias{Hitos: repositorioExpedientes, Repositorio: repositorioExpedientes,
		Transacciones: b, Registro: registro, RegistroTransaccional: registro, Reloj: reloj})
	sesiones := sesionesConsentimientos{}
	tokens := map[string]string{}
	for indice, actor := range []struct {
		nombre string
		perfil identidad.Perfil
		equipo string
	}{
		{"profesional", identidad.ProfesionalETF, equipo},
		{"coordinacion", identidad.Coordinacion, ""},
		{"administracion", identidad.Administracion, ""},
		{"consulta", identidad.Consulta, ""},
		{"ajeno", identidad.ProfesionalETF, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
	} {
		token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{byte(indice + 1)}, 32))
		tokens[actor.nombre] = token
		sesiones[token] = identidad.Usuario{ID: usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{actor.perfil}, EquipoID: actor.equipo}
	}
	mux := http.NewServeMux()
	servidor := httptest.NewTLSServer(mux)
	handler := adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil,
		slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{
			Sesion:      adaptadorHTTP.ConfiguracionSesion{Servicio: sesiones, Origen: servidor.URL, ClaveCSRF: bytes.Repeat([]byte{9}, 32)},
			Documentos:  adaptadorHTTP.ConfiguracionDocumentos{Servicio: documentos},
			Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{Servicio: hitos, Hitos: hitos},
		})
	mux.Handle("/", handler)
	defer servidor.Close()
	cliente := servidor.Client()
	csrf := map[string]string{}
	pedir := func(actor, metodo, ruta, clave, cuerpo, testigo string) (*http.Response, []byte) {
		t.Helper()
		peticion, err := http.NewRequestWithContext(ctx, metodo, servidor.URL+ruta, strings.NewReader(cuerpo))
		if err != nil {
			t.Fatal(err)
		}
		if actor != "" {
			peticion.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: tokens[actor]})
		}
		if metodo == http.MethodPost {
			peticion.Header.Set("Content-Type", "application/json")
			peticion.Header.Set("Origin", servidor.URL)
			peticion.Header.Set("Idempotency-Key", clave)
			peticion.Header.Set("X-CSRF-Token", testigo)
		}
		respuesta, err := cliente.Do(peticion)
		if err != nil {
			t.Fatal(err)
		}
		defer respuesta.Body.Close()
		contenido, err := io.ReadAll(respuesta.Body)
		if err != nil {
			t.Fatal(err)
		}
		return respuesta, contenido
	}
	for actor := range tokens {
		respuesta, contenido := pedir(actor, http.MethodGet, "/api/sesion", "", "", "")
		var acceso struct {
			CSRF string `json:"csrf"`
		}
		if respuesta.StatusCode != 200 || json.Unmarshal(contenido, &acceso) != nil || acceso.CSRF == "" {
			t.Fatalf("sesion %s: %d %s", actor, respuesta.StatusCode, contenido)
		}
		csrf[actor] = acceso.CSRF
	}
	leerEstado := func(id string) estadoConsentimiento {
		t.Helper()
		var estado estadoConsentimiento
		if err := b.Ejecutar(ctx, coordinacion, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			return tx.QueryRow(ctx, `SELECT estado,version,to_char(fecha_aceptacion,'YYYY-MM-DD'),
(SELECT count(*) FROM expediente.hito WHERE expediente_id=$1),
(SELECT count(*) FROM expediente.estado_cambio WHERE expediente_id=$1),
(SELECT count(*) FROM documento.emision WHERE expediente_id=$1)
FROM expediente.expediente WHERE id=$1`, id).Scan(&estado.Estado, &estado.Version, &estado.FechaAceptacion, &estado.Hitos, &estado.Cambios, &estado.Emisiones)
		}); err != nil {
			t.Fatal(err)
		}
		return estado
	}
	for _, caso := range []struct{ id, anexo, instrumento, actor string }{
		{"a2030000-0000-4000-8000-000000000003", "anexo1", "anexo7", "coordinacion"},
		{"a2030000-0000-4000-8000-000000000004", "anexo2", "anexo8", "coordinacion"},
		{"a2030000-0000-4000-8000-000000000001", "anexo1", "anexo7", "profesional"},
		{"a2030000-0000-4000-8000-000000000002", "anexo2", "anexo8", "profesional"},
	} {
		t.Run(caso.actor+"/"+caso.anexo, func(t *testing.T) {
			ruta := "/api/expedientes/" + caso.id + "/documentos/" + caso.anexo
			rutaFicha := "/api/expedientes/" + caso.id
			rFicha, contenidoFicha := pedir(caso.actor, http.MethodGet, rutaFicha, "", "", "")
			var ficha struct {
				Version int `json:"version"`
			}
			if rFicha.StatusCode != 200 || json.Unmarshal(contenidoFicha, &ficha) != nil || ficha.Version != 2 {
				t.Fatalf("lectura de versión: %d %s", rFicha.StatusCode, contenidoFicha)
			}
			if rAjeno, contenidoAjeno := pedir("ajeno", http.MethodGet, rutaFicha, "", "", ""); rAjeno.StatusCode != 404 || strings.Contains(string(contenidoAjeno), "Ortega") {
				t.Fatalf("ficha equipo ajeno: %d %s", rAjeno.StatusCode, contenidoAjeno)
			}
			antes := leerEstado(caso.id)
			if antes.Estado != "aceptada" || antes.Version != 2 || antes.Hitos != 1 || antes.Emisiones != 0 {
				t.Fatal("fixture inicial", antes)
			}
			for _, actor := range []string{"administracion", "consulta", "ajeno", ""} {
				r, contenido := pedir(actor, http.MethodPost, ruta, "denegada-"+actor+caso.anexo, `{"idioma":"es"}`, csrf[actor])
				esperado := 403
				if actor == "" {
					esperado = 401
				}
				if r.StatusCode != esperado || strings.Contains(string(contenido), "Ortega") {
					t.Fatalf("permiso %s: %d %s", actor, r.StatusCode, contenido)
				}
			}
			if r, contenido := pedir(caso.actor, http.MethodPost, ruta, "sin-csrf-"+caso.anexo, `{"idioma":"es"}`, ""); r.StatusCode != 403 {
				t.Fatalf("CSRF: %d %s", r.StatusCode, contenido)
			}
			for _, idioma := range []string{"es", "en"} {
				actor := caso.actor
				clave, cuerpo := "descarga-"+caso.actor+"-"+caso.anexo+"-"+idioma, `{"idioma":"`+idioma+`"}`
				r, contenido := pedir(actor, http.MethodPost, ruta, clave, cuerpo, csrf[actor])
				if r.StatusCode != http.StatusCreated {
					t.Fatalf("descarga real %s/%s/%s: %d %s", actor, caso.anexo, idioma, r.StatusCode, contenido)
				}
				xml := xmlDocumento(t, contenido)
				fechaEsperada := "13/10/2026"
				if idioma == "en" {
					fechaEsperada = "10/13/2026"
				}
				if strings.Contains(xml, "{{") || !strings.Contains(xml, "Lucía Ortega Salas") || !strings.Contains(xml, "Nerea Ortega Salas") || !strings.Contains(xml, "Alhama de Granada (Granada)") || !strings.Contains(xml, fechaEsperada) || !strings.Contains(xml, "____________________") || strings.Contains(xml, "número de DNI") {
					t.Fatal("contenido, fecha o hueco DNI incorrectos", caso.anexo, idioma)
				}
				if r.Header.Get("X-Documento-SHA256") != d.Huella(contenido) || len(r.Header.Get("X-Datos-SHA256")) != 64 || r.Header.Get("X-Emision-ID") == "" || r.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
					t.Fatal("cabeceras de descarga incompletas")
				}
				repetida, contenidoRepetido := pedir(actor, http.MethodPost, ruta, clave, cuerpo, csrf[actor])
				if repetida.StatusCode != 200 || repetida.Header.Get("X-Idempotencia-Repetida") != "true" || repetida.Header.Get("X-Emision-ID") != r.Header.Get("X-Emision-ID") || !bytes.Equal(contenido, contenidoRepetido) {
					t.Fatal("repetición no conserva emisión y bytes")
				}
			}
			despues := leerEstado(caso.id)
			sinEmisiones := despues
			sinEmisiones.Emisiones = antes.Emisiones
			if !reflect.DeepEqual(antes, sinEmisiones) || despues.Emisiones != 2 {
				t.Fatal("generar modifica negocio o duplica emisión", antes, despues)
			}
			rutaHitos := "/api/expedientes/" + caso.id + "/hitos"
			firma := `{"tipo":"consentimiento","fecha":"2026-10-13","version":2}`
			r, contenido := pedir(caso.actor, http.MethodPost, rutaHitos, "firma-"+caso.actor+"-"+caso.anexo, firma, csrf[caso.actor])
			var recibo dominioExpediente.ResultadoHito
			if r.StatusCode != 200 || json.Unmarshal(contenido, &recibo) != nil || recibo.Estado != "en_intervencion" || recibo.Version != 3 {
				t.Fatalf("firma separada: %d %s", r.StatusCode, contenido)
			}
			repetida, contenidoRepetido := pedir(caso.actor, http.MethodPost, rutaHitos, "firma-"+caso.actor+"-"+caso.anexo, firma, csrf[caso.actor])
			if repetida.StatusCode != 200 || !bytes.Equal(contenido, contenidoRepetido) {
				t.Fatal("firma repetida no conserva recibo")
			}
			r, contenido = pedir(caso.actor, http.MethodPost, ruta, "tras-firma-"+caso.actor+"-"+caso.anexo, `{"idioma":"es"}`, csrf[caso.actor])
			if r.StatusCode != 201 {
				t.Fatalf("regenerar tras firma: %d %s", r.StatusCode, contenido)
			}
			firmada := leerEstado(caso.id)
			if firmada.Estado != "en_intervencion" || firmada.Version != 3 || firmada.Hitos != 2 || firmada.Cambios != antes.Cambios+1 || firmada.Emisiones != 3 || firmada.FechaAceptacion != antes.FechaAceptacion {
				t.Fatal("regeneración duplica hito o cambia negocio", firmada)
			}
			r, contenido = pedir(caso.actor, http.MethodPost, rutaHitos, "version-vieja-"+caso.actor+"-"+caso.anexo, `{"tipo":"`+caso.instrumento+`","fecha":"2026-10-13","version":2}`, csrf[caso.actor])
			if r.StatusCode != 409 || !bytes.Contains(contenido, []byte(`"version_actual":3`)) {
				t.Fatalf("conflicto: %d %s", r.StatusCode, contenido)
			}
			r, contenido = pedir(caso.actor, http.MethodPost, rutaHitos, "instrumento-"+caso.actor+"-"+caso.anexo, `{"tipo":"`+caso.instrumento+`","fecha":"2026-10-13","version":3}`, csrf[caso.actor])
			if r.StatusCode != 200 || json.Unmarshal(contenido, &recibo) != nil || recibo.Version != 4 {
				t.Fatalf("fecha instrumento: %d %s", r.StatusCode, contenido)
			}
			r, contenido = pedir(caso.actor, http.MethodGet, rutaHitos, "", "", "")
			var fechas []dominioExpediente.Hito
			if r.StatusCode != 200 || json.Unmarshal(contenido, &fechas) != nil || len(fechas) != 3 {
				t.Fatalf("hitos persistidos: %d %s", r.StatusCode, contenido)
			}
			instrumentoRegistrado := false
			for _, hito := range fechas {
				if hito.Tipo == caso.instrumento && hito.Fecha == "2026-10-13" {
					instrumentoRegistrado = true
				}
			}
			if !instrumentoRegistrado {
				t.Fatal("fecha del instrumento no persistida")
			}
			final := leerEstado(caso.id)
			if final.Emisiones != firmada.Emisiones || final.Hitos != 3 || final.Version != 4 || final.Estado != "en_intervencion" {
				t.Fatal("instrumento generó documento o estado", final)
			}
			if r, _ := pedir(caso.actor, http.MethodPost, "/api/expedientes/"+caso.id+"/documentos/"+caso.instrumento, "no-generable-"+caso.instrumento, `{"idioma":"es"}`, csrf[caso.actor]); r.StatusCode < 400 {
				t.Fatal("Anexo 7/8 no debe generarse")
			}
		})
	}
	if !t.Failed() {
		t.Log("TLS real: cuatro plantillas, perfil profesional real, Word sin DNI, repetición idéntica; consentimiento separado, conflicto 409 y fechas 7/8 sin emisiones")
	}
}
