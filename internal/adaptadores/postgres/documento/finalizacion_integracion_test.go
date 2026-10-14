package documento_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	entradaIdentidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	dominioExp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	paquete "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type idiomaDistribucion struct{}

func (idiomaDistribucion) Referencias(context.Context, string) ([]paquete.Referencia, error) {
	return nil, nil
}

type sesionFinalizacion struct {
	sesiones
	equipo   string
	perfiles []identidad.Perfil
}

func (s sesionFinalizacion) Consultar(context.Context, string) (entradaIdentidad.Acceso, error) {
	return entradaIdentidad.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: usuario, EquipoID: s.equipo, Estado: identidad.Activo, Perfiles: s.perfiles}}, nil
}
func TestFinalizacionDescargaHTTPPostgres(t *testing.T) {
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
	servicioExp := expedientes.Nuevo(expedientes.Dependencias{Repositorio: r, Propuestas: r, MotivosCierre: c, Aceptaciones: r, Transacciones: b, Registro: auditoria, RegistroTransaccional: auditoria, Reloj: reloj{}})
	plantillas := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: idiomaDistribucion{}}
	servicioDoc := app.Nuevo(app.Dependencias{Datos: app.DatosEntrada{Expedientes: expedientes.ConsultaDocumentos{Repositorio: r}, Catalogos: c, Referencias: &paquetes.Referencias{}, Presentacion: plantillas, Reloj: reloj{}}, Repositorio: documentos.Repositorio{}, Firmantes: firmantes.Firmantes{Catalogo: c}, Motor: word.Motor{}, Idiomas: plantillas, Ficheros: plantillas, Transacciones: b, Registro: auditoria, RegistroTransaccional: auditoria, Reloj: reloj{}, Identificadores: ids{}})
	for _, anexo := range []string{"anexo11", "anexo12"} {
		id := "77777777-7777-4777-8777-000000000001"
		if anexo == "anexo12" {
			id = "77777777-7777-4777-8777-000000000101"
		}
		var equipo string
		var version, hitos int
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			if anexo == "anexo12" {
				if _, err := e.Exec(ctx, `UPDATE expediente.expediente SET subprograma='REUNIFICACION' WHERE id=$1`, id); err != nil {
					return err
				}
			}
			if err := e.QueryRow(ctx, `SELECT equipo_id::text,version FROM expediente.expediente WHERE id=$1`, id).Scan(&equipo, &version); err != nil {
				return err
			}
			return e.QueryRow(ctx, `SELECT count(*) FROM expediente.hito WHERE expediente_id=$1`, id).Scan(&hitos)
		})
		if err != nil {
			t.Fatal(err)
		}
		actor := dominioExp.Actor{UsuarioID: usuario, EquipoID: equipo, Perfiles: []string{"profesional_etf"}}
		guardada, err := servicioExp.GuardarFinalizacion(ctx, actor, id, "final-"+anexo, dominioExp.Finalizacion{Version: version, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true, ObjetivosPendientes: "No se ha iniciado la intervención por falta de consentimiento familiar."})
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range []d.Actor{
			{UsuarioID: usuario, EquipoID: equipo, Perfiles: []string{"coordinacion"}},
			{UsuarioID: usuario, EquipoID: equipo, Perfiles: []string{"consulta"}},
			{UsuarioID: usuario, EquipoID: equipo, Perfiles: []string{"administracion"}},
			{UsuarioID: usuario, EquipoID: "88888888-8888-4888-8888-888888888888", Perfiles: []string{"profesional_etf"}},
			{UsuarioID: usuario, EquipoID: "88888888-8888-4888-8888-888888888888", Perfiles: []string{"profesional_etf", "coordinacion"}},
		} {
			_, err := servicioDoc.Emitir(ctx, a, d.Solicitud{Tipo: "finalizacion/" + anexo, ExpedienteID: id, Idioma: "es", Version: guardada.Version, Clave: "negativa-" + anexo})
			if !errors.Is(err, d.ErrAcceso) {
				t.Fatal("perfil/equipo", a.Perfiles, err)
			}
		}
		for _, idioma := range []string{"es", "en"} {
			server := httptest.NewUnstartedServer(nil)
			origen := "https://" + server.Listener.Addr().String()
			server.Config.Handler = httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionFinalizacion{equipo: equipo, perfiles: []identidad.Perfil{identidad.ProfesionalETF}}, Origen: origen, ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: httpETF.ConfiguracionDocumentos{Servicio: servicioDoc}, Expedientes: httpETF.ConfiguracionAceptaciones{Propuestas: servicioExp}})
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
			if err != nil || res.StatusCode != 200 || acceso.CSRF == "" {
				server.Close()
				t.Fatal(res.StatusCode, err)
			}
			ruta := "/api/expedientes/" + id + "/documentos/" + anexo
			descargar := func(ver int, clave string) (int, []byte, http.Header) {
				req, _ := http.NewRequestWithContext(ctx, "POST", origen+ruta, strings.NewReader(fmt.Sprintf(`{"idioma":%q,"version":%d}`, idioma, ver)))
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
			clave := "emitir-" + anexo + "-" + idioma
			status, contenido, headers := descargar(guardada.Version, clave)
			if status != 201 {
				server.Close()
				t.Fatalf("%s %s HTTP %d: %s", anexo, idioma, status, contenido)
			}
			// La RLS rechaza la emisión ajena incluso con ambos perfiles,
			// sin depender de la autorización previa de HTTP ni del caso de uso.
			err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"profesional_etf", "coordinacion"}, EquipoID: "88888888-8888-4888-8888-888888888888"}, func(ctx context.Context) error {
				e, _ := postgres.DesdeContexto(ctx)
				_, err := e.Exec(ctx, `INSERT INTO documento.emision(id,tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_datos,huella_sha256,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave)
 SELECT gen_random_uuid(),tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_datos,huella_sha256,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave
 FROM documento.emision WHERE id=$1`, headers.Get("X-Emision-ID"))
				return err
			})
			var denegada *pgconn.PgError
			if !errors.As(err, &denegada) || denegada.Code != "42501" {
				t.Fatal("RLS permite emisión con equipo ajeno y ambos perfiles", err)
			}
			z, err := zip.NewReader(bytes.NewReader(contenido), int64(len(contenido)))
			if err != nil {
				t.Fatal(err)
			}
			texto := ""
			for _, archivo := range z.File {
				if strings.HasSuffix(archivo.Name, ".xml") {
					rd, err := archivo.Open()
					if err != nil {
						t.Fatal(err)
					}
					xml, _ := io.ReadAll(rd)
					rd.Close()
					texto += string(xml)
				}
			}
			if strings.Contains(texto, "{{") || !strings.Contains(texto, "No se ha iniciado") || !strings.Contains(texto, "Ortega Salas") || headers.Get("X-Documento-SHA256") != d.Huella(contenido) {
				t.Fatal("DOCX sin sustituir o huella distinta")
			}
			status, otra, headers := descargar(guardada.Version, clave)
			if status != 200 || !bytes.Equal(otra, contenido) || headers.Get("X-Idempotencia-Repetida") != "true" {
				t.Fatal("repetición distinta", status)
			}
			status, _, _ = descargar(guardada.Version-1, clave+"-obsoleta")
			if status != 409 {
				t.Fatal("versión sin 409", status)
			}
			server.Close()
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			var estado string
			var v, n int
			err := e.QueryRow(ctx, `SELECT estado,version,(SELECT count(*) FROM expediente.hito WHERE expediente_id=$1) FROM expediente.expediente WHERE id=$1`, id).Scan(&estado, &v, &n)
			if err == nil && (estado != "aceptada" || v != guardada.Version || n != hitos) {
				t.Fatal("emisión cambió estado, versión o hito")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Descarga HTTPS real Anexos 11/12 ES/EN: DOCX abierto, marcadores sustituidos, SHA256 verificado; repetición idéntica, 409 de versión, estado e hitos intactos")
}
