package documento_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	cat "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documentos "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paquetes "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	expedientes "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	dominioExp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type sesionActaCierre struct {
	sesiones
	actor d.Actor
}

func (s sesionActaCierre) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	u := identidad.Usuario{ID: s.actor.UsuarioID, EquipoID: s.actor.EquipoID, Estado: identidad.Activo}
	for _, p := range s.actor.Perfiles {
		u.Perfiles = append(u.Perfiles, identidad.Perfil(p))
	}
	return identidadEntrada.Acceso{Estado: identidad.Activo, Usuario: u}, nil
}
func TestActaCierreDescargaHTTPSPostgresYEmisionSeparada(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	c := &cat.Repositorio{}
	r := &exp.Repositorio{}
	auditoria := postgres.NuevoRegistroAuditoria(b)
	servicioExp := expedientes.Nuevo(expedientes.Dependencias{Repositorio: r, Propuestas: r, Cierres: r, MotivosCierre: c, Aceptaciones: r, EmisionesFinalizacion: documentos.Repositorio{}, ConsultaCierreSPM: expedientes.ConsultaCierreSPM{Repositorio: r, Salidas: r}, Transacciones: b, Registro: auditoria, RegistroTransaccional: auditoria, Reloj: reloj{}})
	plantillas := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: idiomaDistribucion{}}
	deps := app.Dependencias{Datos: app.DatosEntrada{Expedientes: expedientes.ConsultaDocumentos{Repositorio: r}, Catalogos: c, Referencias: &paquetes.Referencias{}, Presentacion: plantillas, Reloj: reloj{}}, Repositorio: documentos.Repositorio{}, Firmantes: firmantes.Firmantes{Catalogo: c}, Motor: word.Motor{}, Idiomas: plantillas, Ficheros: plantillas, Transacciones: b, Registro: auditoria, RegistroTransaccional: auditoria, Reloj: reloj{}, Identificadores: ids{}}
	servicioDoc := app.Nuevo(deps)
	const id = "77777777-7777-4777-8777-000000000501"
	const otra = "11111111-1111-4111-8111-111111111113"
	var equipo string
	var version int
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT equipo_id::text,version FROM expediente.expediente WHERE id=$1`, id).Scan(&equipo, &version)
	})
	if err != nil {
		t.Fatal(err)
	}
	profesional := dominioExp.Actor{UsuarioID: usuario, EquipoID: equipo, Perfiles: []string{"profesional_etf"}}
	final, err := servicioExp.GuardarFinalizacion(ctx, profesional, id, "final-acta", dominioExp.Finalizacion{Version: version, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true})
	if err != nil {
		t.Fatal(err)
	}
	// El Anexo se emite antes de la propuesta; el cierre reconoce su versión de datos.
	if _, err = servicioDoc.Emitir(ctx, d.Actor{UsuarioID: usuario, EquipoID: equipo, Perfiles: profesional.Perfiles}, d.Solicitud{Tipo: d.Anexo11, ExpedienteID: id, Idioma: "es", Version: final.Version, Clave: "anexo-previo-acta"}); err != nil {
		t.Fatal(err)
	}
	prop, err := servicioExp.ProponerCierre(ctx, profesional, id, "propuesta-acta", dominioExp.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	coord := d.Actor{UsuarioID: otra, Perfiles: []string{"coordinacion"}}
	for _, a := range []d.Actor{{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, {UsuarioID: otra, EquipoID: equipo, Perfiles: []string{"profesional_etf"}}, {UsuarioID: otra, Perfiles: []string{"consulta"}}, {UsuarioID: otra, Perfiles: []string{"administracion"}}} {
		if _, err = servicioDoc.Emitir(ctx, a, d.Solicitud{Tipo: d.ActaCierre, ExpedienteID: id, Idioma: "es", Version: prop.Version, Clave: "denegada-acta"}); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(a.Perfiles, err)
		}
	}
	for _, idioma := range []string{"es", "en"} {
		server := httptest.NewUnstartedServer(nil)
		origen := "https://" + server.Listener.Addr().String()
		server.Config.Handler = httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionActaCierre{actor: coord}, Origen: origen, ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: httpETF.ConfiguracionDocumentos{Servicio: servicioDoc}})
		server.StartTLS()
		client := server.Client()
		cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))}
		req, _ := http.NewRequestWithContext(ctx, "GET", origen+"/api/sesion", nil)
		req.AddCookie(cookie)
		res, err := client.Do(req)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		var acceso struct {
			CSRF string `json:"csrf"`
		}
		err = json.NewDecoder(res.Body).Decode(&acceso)
		res.Body.Close()
		if err != nil || res.StatusCode != 200 {
			server.Close()
			t.Fatal(res.StatusCode, err)
		}
		descargar := func(v int, clave string) (int, []byte, http.Header) {
			cuerpo, _ := json.Marshal(map[string]any{"idioma": idioma, "version": v})
			req, _ := http.NewRequestWithContext(ctx, "POST", origen+"/api/expedientes/"+id+"/documentos/acta_cierre", bytes.NewReader(cuerpo))
			req.AddCookie(cookie)
			req.Header.Set("Origin", origen)
			req.Header.Set("X-CSRF-Token", acceso.CSRF)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", clave)
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			contenido, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			return res.StatusCode, contenido, res.Header
		}
		clave := "acta-" + idioma
		status, contenido, h := descargar(prop.Version, clave)
		if status != 201 {
			server.Close()
			t.Fatal(status, string(contenido))
		}
		xml := xmlDocumento(t, contenido)
		titulo := "ACTA DE CIERRE"
		if idioma == "en" {
			titulo = "CLOSURE RECORD"
		}
		if strings.Contains(xml, "{{") || !strings.Contains(xml, titulo) || !strings.Contains(xml, "Nerea Ortega Salas") || !strings.Contains(xml, "Eva Ortega Salas") || h.Get("X-Documento-SHA256") != d.Huella(contenido) {
			server.Close()
			t.Fatal("Acta incompleta o huella distinta")
		}
		status, rep, h := descargar(prop.Version, clave)
		if status != 200 || !bytes.Equal(rep, contenido) || h.Get("X-Idempotencia-Repetida") != "true" {
			server.Close()
			t.Fatal("repetición", status, string(rep))
		}
		status, _, _ = descargar(prop.Version-1, clave+"-obsoleta")
		if status != 409 {
			server.Close()
			t.Fatal("versión", status)
		}
		server.Close()
	}
	vista, err := servicioExp.Consultar(ctx, dominioExp.Actor{UsuarioID: otra, Perfiles: coord.Perfiles}, id)
	if err != nil || vista.Estado != dominioExp.Aceptada || vista.Version != prop.Version {
		t.Fatal(vista, err)
	}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err = app.Nuevo(fallo).Emitir(ctx, coord, d.Solicitud{Tipo: d.ActaCierre, ExpedienteID: id, Idioma: "es", Version: prop.Version, Clave: "acta-auditoria-fallida"}); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	cerrado, err := servicioExp.Cerrar(ctx, dominioExp.Actor{UsuarioID: otra, Perfiles: coord.Perfiles}, id, "confirmar-acta", dominioExp.PeticionCierre{Version: prop.Version, Fecha: "2026-10-13", MotivoActaTexto: "La familia ha decidido finalizar la atención del equipo."})
	if err != nil || cerrado.Aviso != "" {
		t.Fatal(cerrado, err)
	}
	resultado, err := servicioDoc.Emitir(ctx, coord, d.Solicitud{Tipo: d.ActaCierre, ExpedienteID: id, Idioma: "es", Version: cerrado.Version, Clave: "acta-confirmada"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xmlDocumento(t, resultado.Contenido), "La familia ha decidido finalizar") {
		t.Fatal("motivo manual ausente")
	}
	vista, err = servicioExp.Consultar(ctx, dominioExp.Actor{UsuarioID: otra, Perfiles: coord.Perfiles}, id)
	if err != nil || vista.Estado != dominioExp.Cerrada || vista.Version != cerrado.Version {
		t.Fatal(vista, err)
	}
	t.Log("Acta oficial HTTPS ES/EN con 15 menores; 201/200 idempotente y SHA256, 409 y perfiles; emisión no cierra; cierre separado reconoce Anexo; Acta posterior conserva fecha y motivo manual")
}
