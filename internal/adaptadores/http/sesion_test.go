package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type servicioSesionPrueba struct {
	falloAdmision                error
	entradas, consultas, salidas int
	rechazos                     int
	falloAuditoria               error
	fallo                        error
	inicio                       entrada.Inicio
}

func (s *servicioSesionPrueba) Entrar(_ context.Context, i entrada.Inicio) (entrada.SesionAbierta, error) {
	s.entradas++
	s.inicio = i
	return entrada.SesionAbierta{Identificador: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)), Acceso: entrada.Acceso{Estado: dominio.PendienteAlta}}, s.fallo
}
func (s *servicioSesionPrueba) Consultar(context.Context, string) (entrada.Acceso, error) {
	s.consultas++
	return entrada.Acceso{Estado: dominio.PendienteAlta}, s.fallo
}
func (s *servicioSesionPrueba) Salir(context.Context, string) error { s.salidas++; return s.fallo }
func servidorSesionPrueba(s *servicioSesionPrueba) http.Handler {
	return Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}})
}
func pedirSesion(h http.Handler, metodo, cuerpo string, cookies []*http.Cookie, testigo, origen string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, "https://etf.interno/api/sesion", strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	if testigo != "" {
		r.Header.Set("X-CSRF-Token", testigo)
	}
	if origen != "" {
		r.Header.Set("Origin", origen)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func datosSesion(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var datos map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &datos); err != nil {
		t.Fatal(err)
	}
	return datos
}
func TestSesionHTTPEntrarPendienteSalir(t *testing.T) {
	s := &servicioSesionPrueba{}
	h := servidorSesionPrueba(s)
	inicial := pedirSesion(h, "GET", "", nil, "", "")
	csrf := datosSesion(t, inicial)["csrf"].(string)
	abierta := pedirSesion(h, "POST", `{"cuenta":"ines.moreno","contrasena":"clave-ficticia"}`, inicial.Result().Cookies(), csrf, "https://etf.interno")
	if abierta.Code != 200 || s.entradas != 1 || s.inicio.IP == (dominio.IP{}) {
		t.Fatalf("entrada: %d %s", abierta.Code, abierta.Body.String())
	}
	cookies := abierta.Result().Cookies()
	var sesion *http.Cookie
	for _, c := range cookies {
		if c.Name == cookieSesion {
			sesion = c
		}
	}
	if sesion == nil || !sesion.HttpOnly || !sesion.Secure || sesion.SameSite != http.SameSiteStrictMode || sesion.Path != "/" || sesion.Domain != "" || !opaco(sesion.Value) {
		t.Fatal("cookie insegura")
	}
	consulta := pedirSesion(h, "GET", "", []*http.Cookie{sesion}, "", "")
	if consulta.Code != 200 || datosSesion(t, consulta)["estado"] != "pendiente_alta" || strings.Contains(consulta.Body.String(), sesion.Value) {
		t.Fatal("consulta insegura")
	}
	cierre := pedirSesion(h, "DELETE", "", []*http.Cookie{sesion}, datosSesion(t, consulta)["csrf"].(string), "https://etf.interno")
	if cierre.Code != 204 || s.salidas != 1 {
		t.Fatalf("cierre: %d", cierre.Code)
	}
	for _, c := range cierre.Result().Cookies() {
		if c.MaxAge != -1 {
			t.Fatal("cookie no eliminada")
		}
	}
	for _, cabecera := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Cache-Control", "X-Frame-Options", "Permissions-Policy"} {
		if consulta.Header().Get(cabecera) == "" {
			t.Fatal(cabecera)
		}
	}
}
func TestSesionHTTPCSRFImpideEntradaYSalida(t *testing.T) {
	for _, metodo := range []string{"POST", "DELETE"} {
		for _, origen := range []string{"", "https://intruso.interno", "null", "https://etf.interno"} {
			s := &servicioSesionPrueba{}
			h := servidorSesionPrueba(s)
			inicial := pedirSesion(h, "GET", "", nil, "", "")
			csrf := datosSesion(t, inicial)["csrf"].(string)
			if origen == "https://etf.interno" {
				csrf = "incorrecto"
			}
			respuesta := pedirSesion(h, metodo, `{"cuenta":"ines.moreno","contrasena":"clave-ficticia"}`, inicial.Result().Cookies(), csrf, origen)
			if respuesta.Code != 403 || s.entradas+s.salidas != 0 {
				t.Fatal("CSRF llegó al servicio")
			}
		}
	}
}
func TestSesionHTTPTestigoNoSePuedeUsarEnOtraSesion(t *testing.T) {
	s := &servicioSesionPrueba{}
	h := servidorSesionPrueba(s)
	uno := pedirSesion(h, "GET", "", nil, "", "")
	dos := pedirSesion(h, "GET", "", nil, "", "")
	respuesta := pedirSesion(h, "POST", `{}`, dos.Result().Cookies(), datosSesion(t, uno)["csrf"].(string), "https://etf.interno")
	if respuesta.Code != 403 || s.entradas != 0 {
		t.Fatal("testigo transferido")
	}
}
func TestSesionHTTPValidaJSONYNoExponeErrores(t *testing.T) {
	for _, cuerpo := range []string{`{`, `{ "cuenta":"ines.moreno","contrasena":"clave-ficticia","perfil":"administracion"}`, `{} {}`, strings.Repeat("x", 5000)} {
		s := &servicioSesionPrueba{}
		h := servidorSesionPrueba(s)
		inicial := pedirSesion(h, "GET", "", nil, "", "")
		respuesta := pedirSesion(h, "POST", cuerpo, inicial.Result().Cookies(), datosSesion(t, inicial)["csrf"].(string), "https://etf.interno")
		if respuesta.Code != 400 || s.entradas != 0 {
			t.Fatal("JSON no validado")
		}
	}
	for _, caso := range []struct {
		err    error
		estado int
	}{{dominio.ErrDirectorioNoDisponible, 503}, {dominio.ErrCredencialesInvalidas, 401}, {dominio.ErrIntentosLimitados, 429}, {errors.New("contraseña-reservada"), 503}} {
		s := &servicioSesionPrueba{fallo: caso.err}
		h := servidorSesionPrueba(s)
		inicial := pedirSesion(h, "GET", "", nil, "", "")
		respuesta := pedirSesion(h, "POST", `{"cuenta":"ines.moreno","contrasena":"clave-ficticia"}`, inicial.Result().Cookies(), datosSesion(t, inicial)["csrf"].(string), "https://etf.interno")
		if respuesta.Code != caso.estado || strings.Contains(respuesta.Body.String(), "reservada") {
			t.Fatal("error inseguro")
		}
		for _, c := range respuesta.Result().Cookies() {
			if c.Name == cookieSesion && c.MaxAge != -1 {
				t.Fatal("sesión ante error")
			}
		}
	}
}
func TestSesionHTTPFalloAlCerrarConservaCookie(t *testing.T) {
	s := &servicioSesionPrueba{}
	h := servidorSesionPrueba(s)
	inicial := pedirSesion(h, "GET", "", nil, "", "")
	abierta := pedirSesion(h, "POST", `{"cuenta":"ines.moreno","contrasena":"clave-ficticia"}`, inicial.Result().Cookies(), datosSesion(t, inicial)["csrf"].(string), "https://etf.interno")
	s.fallo = dominio.ErrPersistencia
	cierre := pedirSesion(h, "DELETE", "", abierta.Result().Cookies(), datosSesion(t, abierta)["csrf"].(string), "https://etf.interno")
	if cierre.Code != 503 || len(cierre.Result().Cookies()) != 0 {
		t.Fatal("cierre confirmado sin persistir")
	}
}

func (s *servicioSesionPrueba) RegistrarRechazo(context.Context, string, string) error {
	s.rechazos++
	return s.falloAuditoria
}
func TestSesionHTTPFalloAuditoriaNoLlegaADirectorio(t *testing.T) {
	servicio := &servicioSesionPrueba{falloAuditoria: dominio.ErrPersistencia}
	respuesta := pedirSesion(servidorSesionPrueba(servicio), "POST", `{}`, nil, "", "")
	if respuesta.Code != 503 || servicio.rechazos != 1 || servicio.entradas != 0 {
		t.Fatal("rechazo sin confirmar auditoría")
	}
}

func (s *servicioSesionPrueba) AdmitirPeticion(context.Context) error { return s.falloAdmision }
func TestSesionHTTPLimiteTransporteNoEscribeAuditoria(t *testing.T) {
	servicio := &servicioSesionPrueba{falloAdmision: dominio.ErrIntentosLimitados}
	respuesta := pedirSesion(servidorSesionPrueba(servicio), "POST", `{}`, nil, "", "")
	if respuesta.Code != 429 || servicio.rechazos != 0 || servicio.entradas != 0 {
		t.Fatal("petición limitada llegó al caso de uso o auditoría")
	}
}

func TestSesionProyectaNombreYPantallasAutorizadas(t *testing.T) {
	s := &sesionHTTP{clave: bytes.Repeat([]byte{9}, 32)}
	for _, caso := range []struct {
		nombre    string
		estado    dominio.Estado
		perfiles  []dominio.Perfil
		pantallas int
	}{
		{"pendiente", dominio.PendienteAlta, nil, 0},
		{"administracion", dominio.Activo, []dominio.Perfil{dominio.Administracion}, 13},
		{"combinados", dominio.Activo, []dominio.Perfil{dominio.Administracion, dominio.Coordinacion}, 25},
		{"profesional", dominio.Activo, []dominio.Perfil{dominio.ProfesionalETF}, 0},
		{"consulta", dominio.Activo, []dominio.Perfil{dominio.Consulta}, 2},
		{"coordinacion", dominio.Activo, []dominio.Perfil{dominio.Coordinacion}, 13},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.responderAcceso(w, entrada.Acceso{Estado: caso.estado, Usuario: dominio.Usuario{
				ID: "persona-ines", Estado: caso.estado, IdentidadDirectorio: dominio.IdentidadDirectorio{CuentaAD: "ines.moreno", NombreMostrado: "Inés Moreno Castillo"}, Perfiles: caso.perfiles,
			}}, "sesion-inventada")
			datos := datosSesion(t, w)
			if caso.estado == dominio.PendienteAlta {
				if datos["usuario"] != nil || datos["pantallas"] != nil || strings.Contains(w.Body.String(), "Inés") {
					t.Fatal("datos de persona en pendiente")
				}
				return
			}
			if datos["usuario"].(map[string]any)["nombre"] != "Inés Moreno Castillo" {
				t.Fatal("falta el nombre del personal")
			}
			pantallas := datos["pantallas"].([]any)
			if len(pantallas) != caso.pantallas {
				t.Fatalf("pantallas: %v", pantallas)
			}
			vistas := map[string]bool{}
			for _, item := range pantallas {
				pantalla := item.(map[string]any)
				ruta := pantalla["ruta"].(string)
				if vistas[ruta] {
					t.Fatal("pantalla duplicada", ruta)
				}
				vistas[ruta] = true
				for _, regla := range tablaRutas() {
					if regla.Patron == ruta && (!regla.Pantalla || regla.ClaveTitulo != pantalla["clave"]) {
						t.Fatal("pantalla fuera de la tabla", ruta)
					}
				}
			}
		})
	}
}
