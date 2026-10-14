package pdf_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/memoria"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// La prueba monta HTTP, caso de uso, validación y lector reales. Persistencia,
// sesión, catálogo y auditoría son dobles con datos inventados; no acredita RLS.
type entornoSubida struct {
	datos   map[string]dominio.Derivacion
	eventos []auditoria.Datos
	ids     int
}

func (e *entornoSubida) Guardar(_ context.Context, d dominio.Derivacion) error {
	e.datos[d.ID] = d
	return nil
}
func (e *entornoSubida) Obtener(_ context.Context, id string) (dominio.Derivacion, error) {
	return e.datos[id], nil
}
func (e *entornoSubida) Ejecutar(ctx context.Context, _ plataforma.Identidad, f func(context.Context) error) error {
	return f(ctx)
}
func (e *entornoSubida) Confirmar(_ context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	if err := d.Validar(); err != nil {
		return auditoria.Evento{}, err
	}
	e.eventos = append(e.eventos, d)
	return auditoria.Evento{Datos: d}, nil
}
func (e *entornoSubida) RegistrarEnTransaccion(ctx context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	return e.Confirmar(ctx, d)
}
func (*entornoSubida) Ahora() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }
func (e *entornoSubida) Nuevo() (string, error) {
	e.ids++
	return fmt.Sprintf("ab60479d-b778-407f-a83e-%012d", e.ids), nil
}
func (*entornoSubida) Vigente(context.Context, string, time.Time) (catalogo.Version, error) {
	return catalogo.Version{}, nil
}
func (*entornoSubida) Versiones(context.Context, string) ([]catalogo.Version, error) { return nil, nil }
func (*entornoSubida) Datos(context.Context, catalogo.Version) (catalogo.Datos, error) {
	return catalogo.Datos{}, nil
}
func (*entornoSubida) Nombres(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

type sesionSubida struct{}

func (sesionSubida) AdmitirPeticion(context.Context) error { return nil }
func (sesionSubida) Entrar(context.Context, identidadEntrada.Inicio) (identidadEntrada.SesionAbierta, error) {
	return identidadEntrada.SesionAbierta{}, nil
}
func (sesionSubida) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	return identidadEntrada.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: "ab60479d-b778-407f-a83e-012345678901", Estado: identidad.Activo, Perfiles: []identidad.Perfil{identidad.Coordinacion}}}, nil
}
func (sesionSubida) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesionSubida) Salir(context.Context, string) error                    { return nil }

type antivirusLimpio struct{}

func (antivirusLimpio) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusLimpio, nil
}

func TestSubidaHTTPPDF(t *testing.T) { probarSubida(t, antivirusLimpio{}, false) }

// ETF_PRUEBA_CLAMAV_DIRECCION activa el mismo recorrido contra clamd de compose.
// Un antivirus caído hace fallar esta prueba; no se admite el aviso como éxito.
func TestSubidaHTTPPDFClamAV(t *testing.T) {
	direccion := os.Getenv("ETF_PRUEBA_CLAMAV_DIRECCION")
	if direccion == "" {
		t.Skip("ETF_PRUEBA_CLAMAV_DIRECCION")
	}
	av, err := clamav.Nuevo(clamav.Configuracion{Direccion: direccion, TiempoLimite: 3 * time.Second, MaximoBytes: dominio.MaxBytes})
	if err != nil {
		t.Fatal(err)
	}
	probarSubida(t, av, true)
}

func probarSubida(t *testing.T, av plataforma.Antivirus, eicar bool) {
	t.Helper()
	e := &entornoSubida{datos: map[string]dominio.Derivacion{}}
	servicio := aplicacion.Nuevo(aplicacion.Dependencias{Borradores: memoria.NuevosBorradores(time.Minute, 32), Repositorio: e, Transacciones: e, Registro: e, RegistroTransaccional: e, Validador: lectura.Validador{}, Antivirus: av, Lector: lectura.NuevoLector(), Catalogo: e, NombresMunicipios: e, Reloj: e, Identificadores: e})
	clave := bytes.Repeat([]byte{9}, 32)
	const origen = "https://etf.interno"
	h := adaptadorHTTP.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{
		Sesion:       adaptadorHTTP.ConfiguracionSesion{Servicio: sesionSubida{}, Origen: origen, ClaveCSRF: clave},
		Derivaciones: adaptadorHTTP.ConfiguracionDerivaciones{Servicio: servicio},
	})
	servidor := httptest.NewTLSServer(h)
	defer servidor.Close()
	cliente := servidor.Client()
	cliente.Timeout = 10 * time.Second
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write([]byte(cookie))
	csrf := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	subir := func(t *testing.T, nombre string, b []byte) (*http.Response, []byte) {
		t.Helper()
		var cuerpo bytes.Buffer
		m := multipart.NewWriter(&cuerpo)
		p, err := m.CreateFormFile("anexo", nombre)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = p.Write(b); err != nil {
			t.Fatal(err)
		}
		if err = m.Close(); err != nil {
			t.Fatal(err)
		}
		r, err := http.NewRequest("POST", servidor.URL+"/api/coordinacion/derivaciones", &cuerpo)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", m.FormDataContentType())
		r.Header.Set("Origin", origen)
		r.Header.Set("X-CSRF-Token", csrf)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		resp, err := cliente.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp, body
	}
	for _, nombre := range []string{"anexo3.pdf", "anexo4.pdf", "sin_texto.pdf"} {
		t.Run(nombre, func(t *testing.T) {
			resp, b := subir(t, nombre, corpus(t, nombre))
			if resp.StatusCode != 200 {
				t.Fatalf("HTTP %d: %s", resp.StatusCode, b)
			}
			var d dominio.Derivacion
			if err := json.Unmarshal(b, &d); err != nil {
				t.Fatal(err)
			}
			if d.ID != "" || d.Estado != "" || len(e.datos) != 0 || resp.Header.Get("Location") != "" || d.Numero != nil || d.Lectura.Formato != "pdf" || d.Lectura.Antivirus != "limpio" || resp.Header.Get("Cache-Control") != "no-store" {
				t.Fatal(d.Lectura)
			}
			if nombre == "sin_texto.pdf" {
				if len(d.Datos.Extraccion.Campos) != 0 || !slices.Contains(d.Datos.Extraccion.Avisos, "anexo.pdf_sin_texto") {
					t.Fatal(d.Datos)
				}
			} else if !d.Datos.Extraccion.PlantillaReconocida || len(d.Lectura.CamposDudosos) == 0 {
				t.Fatal(d.Datos)
			}
			t.Logf("POST=200 sin expediente antivirus=%s campos_dudosos=%d", d.Lectura.Antivirus, len(d.Lectura.CamposDudosos))
		})
	}
	for _, caso := range []struct {
		nombre string
		b      []byte
		estado int
	}{
		{"corrupto.pdf", corpus(t, "corrupto.pdf"), 422},
		{"grande.pdf", make([]byte, dominio.MaxBytes+1), 413},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			antes := len(e.datos)
			resp, _ := subir(t, caso.nombre, caso.b)
			if resp.StatusCode != caso.estado {
				t.Fatal(resp.StatusCode)
			}
			if len(e.datos) != antes {
				t.Fatalf("rechazo guardó una derivación: antes=%d después=%d", antes, len(e.datos))
			}
		})
	}
	if eicar {
		b := corpus(t, "anexo3.pdf")
		firma := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$" + "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
		b = bytes.Replace(b, []byte("%%EOF"), []byte("%"+firma+"\n%%EOF"), 1)
		antes := len(e.datos)
		resp, body := subir(t, "infectado.pdf", b)
		if resp.StatusCode != 422 || !bytes.Contains(body, []byte("expediente.anexo_infectado")) {
			t.Fatalf("EICAR HTTP %d: %s", resp.StatusCode, body)
		}
		if len(e.datos) != antes {
			t.Fatalf("rechazo guardó una derivación: antes=%d después=%d", antes, len(e.datos))
		}
		t.Log("EICAR PDF: HTTP=422 expediente.anexo_infectado")
	}
	if len(e.datos) != 0 {
		t.Fatal("rechazo guardó una derivación", len(e.datos))
	}
	if len(e.eventos) < 5 {
		t.Fatal("falta auditoría de análisis o rechazo", len(e.eventos))
	}
}
