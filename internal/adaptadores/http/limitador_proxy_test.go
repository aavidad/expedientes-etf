package http

// Regresión F5.02: el límite de intentos y de peticiones de /api/sesion debe
// contar a cada cliente por separado aunque todos lleguen por el proxy HTTPS,
// y las entradas correctas no deben gastar el cupo de los demás.

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type directorioLimite struct{}

func (directorioLimite) Autenticar(_ context.Context, cuenta, clave string) (dominio.IdentidadDirectorio, error) {
	if cuenta == "ana.navas" && clave == "correcta" {
		return dominio.IdentidadDirectorio{GUID: dominio.GUID{1, 2, 3}, CuentaAD: cuenta, NombreMostrado: "Ana Navas"}, nil
	}
	return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
}

type usuariosLimite struct {
	mu  sync.Mutex
	por map[dominio.GUID]dominio.Usuario
}

func (u *usuariosLimite) RegistrarAcceso(_ context.Context, i dominio.IdentidadDirectorio, _ time.Time) (dominio.Usuario, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if x, ok := u.por[i.GUID]; ok {
		return x, nil
	}
	x := dominio.Usuario{ID: "u-1", IdentidadDirectorio: i, Estado: dominio.Activo, Perfiles: []dominio.Perfil{dominio.Consulta}}
	u.por[i.GUID] = x
	return x, nil
}
func (u *usuariosLimite) Obtener(_ context.Context, id string) (dominio.Usuario, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, x := range u.por {
		if x.ID == id {
			return x, nil
		}
	}
	return dominio.Usuario{}, dominio.ErrUsuarioNoDisponible
}

type sesionesLimite struct {
	mu  sync.Mutex
	por map[dominio.Huella]dominio.Sesion
}

func (s *sesionesLimite) Crear(_ context.Context, x dominio.Sesion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.por[x.Huella] = x
	return nil
}
func (s *sesionesLimite) Modificar(ctx context.Context, h dominio.Huella, f func(context.Context, *dominio.Sesion) error) error {
	s.mu.Lock()
	x, ok := s.por[h]
	s.mu.Unlock()
	if !ok {
		return dominio.ErrSesionNoValida
	}
	if err := f(ctx, &x); err != nil {
		return err
	}
	s.mu.Lock()
	s.por[h] = x
	s.mu.Unlock()
	return nil
}
func (s *sesionesLimite) Eliminar(_ context.Context, h dominio.Huella) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.por, h)
	return nil
}

type relojLimite struct{}

func (relojLimite) Ahora() time.Time { return time.Now().UTC() }

type azarLimite struct{}

func (azarLimite) Llenar(b []byte) error { _, err := rand.Read(b); return err }

const proxyLimite = "10.250.4.10"

// Misma política que cmd/etf/identidad.go.
func servidorLimite(t *testing.T, proxies ...netip.Prefix) http.Handler {
	t.Helper()
	l, err := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: 15 * time.Minute, PorUsuario: 5, PorIP: 30, MaxClaves: 10000}, dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 100, MaxClaves: 10000})
	if err != nil {
		t.Fatal(err)
	}
	s, err := aplicacion.NuevoServicio(directorioLimite{}, &usuariosLimite{por: map[dominio.GUID]dominio.Usuario{}}, &sesionesLimite{por: map[dominio.Huella]dominio.Sesion{}}, relojLimite{}, azarLimite{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: 8 * time.Hour}, l)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32), Proxies: proxies}
	return Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg})
}

func pedirLimite(h http.Handler, metodo, remota, reenviada string, cookies []*http.Cookie, testigo, cuerpo string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, "https://etf.interno/api/sesion", strings.NewReader(cuerpo))
	r.RemoteAddr = remota
	if reenviada != "" {
		r.Header.Set("X-Forwarded-For", reenviada)
	}
	if metodo == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("X-CSRF-Token", testigo)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// entrarLimite repite lo que hace la SPA: pide el testigo previo y entra.
func entrarLimite(t *testing.T, h http.Handler, remota, reenviada, cuenta, clave string) int {
	t.Helper()
	w := pedirLimite(h, http.MethodGet, remota, reenviada, nil, "", "")
	if w.Code != http.StatusOK {
		return w.Code
	}
	testigo := datosSesion(t, w)["csrf"].(string)
	cuerpo := fmt.Sprintf(`{"cuenta":%q,"contrasena":%q}`, cuenta, clave)
	return pedirLimite(h, http.MethodPost, remota, reenviada, w.Result().Cookies(), testigo, cuerpo).Code
}

func TestLimiteSesionSeparaClientesTrasProxyDeclarado(t *testing.T) {
	h := servidorLimite(t, netip.MustParsePrefix("10.250.4.0/28"))
	for i := 1; i <= 30; i++ {
		if c := entrarLimite(t, h, proxyLimite+":40001", "10.1.1.1", fmt.Sprintf("inventada%d", i), "mala"); c != http.StatusUnauthorized {
			t.Fatalf("intento %d: %d", i, c)
		}
	}
	if c := entrarLimite(t, h, proxyLimite+":40002", "10.1.1.1", "otra.cuenta", "mala"); c != http.StatusTooManyRequests {
		t.Fatalf("el cliente que falla debe quedar limitado: %d", c)
	}
	if c := entrarLimite(t, h, proxyLimite+":51234", "10.1.1.2", "ana.navas", "correcta"); c != http.StatusOK {
		t.Fatalf("otra persona tras el mismo proxy no puede entrar: %d", c)
	}
}

func TestLimiteSesionIgnoraReenvioDeOrigenNoDeclarado(t *testing.T) {
	h := servidorLimite(t, netip.MustParsePrefix("10.250.4.0/28"))
	for i := 1; i <= 30; i++ {
		// Cambiar la cabecera desde una IP que no es proxy no renueva el cupo.
		entrarLimite(t, h, "10.9.9.9:1000", fmt.Sprintf("10.2.0.%d", i), fmt.Sprintf("inventada%d", i), "mala")
	}
	if c := entrarLimite(t, h, "10.9.9.9:1000", "10.2.0.200", "otra.cuenta", "mala"); c != http.StatusTooManyRequests {
		t.Fatalf("una cabecera falsificada eludió el límite: %d", c)
	}
	// Sin proxies declarados la cabecera tampoco cuenta.
	h = servidorLimite(t)
	if w := pedirLimite(h, http.MethodGet, proxyLimite+":1", "texto no válido", nil, "", ""); w.Code != http.StatusOK {
		t.Fatalf("sin proxies declarados la cabecera debe ignorarse: %d", w.Code)
	}
}

func TestLimiteSesionRechazaReenvioMalFormadoDesdeProxy(t *testing.T) {
	h := servidorLimite(t, netip.MustParsePrefix("10.250.4.0/28"))
	for _, reenviada := range []string{"", "no-es-ip", "10.1.1.1, basura", "10.250.4.1"} {
		if w := pedirLimite(h, http.MethodGet, proxyLimite+":2", reenviada, nil, "", ""); w.Code != http.StatusBadRequest {
			t.Fatalf("%q desde el proxy debe rechazarse: %d", reenviada, w.Code)
		}
	}
	// Se toma la última dirección que no es un proxy declarado.
	if w := pedirLimite(h, http.MethodGet, proxyLimite+":3", "203.0.113.9, 10.1.1.3, 10.250.4.2", nil, "", ""); w.Code != http.StatusOK {
		t.Fatalf("cadena de proxies válida rechazada: %d", w.Code)
	}
}

func TestLimiteSesionNoGastaCupoConPeticionesDeOtrosTrasProxy(t *testing.T) {
	h := servidorLimite(t, netip.MustParsePrefix("10.250.4.0/28"))
	for i := 1; i <= 100; i++ {
		if w := pedirLimite(h, http.MethodGet, fmt.Sprintf("%s:%d", proxyLimite, 30000+i), "10.1.1.1", nil, "", ""); w.Code != http.StatusOK {
			t.Fatalf("petición %d: %d", i, w.Code)
		}
	}
	if w := pedirLimite(h, http.MethodGet, proxyLimite+":61000", "10.1.1.2", nil, "", ""); w.Code != http.StatusOK {
		t.Fatalf("otra persona tras el proxy no carga la aplicación: %d", w.Code)
	}
}

func TestLimiteSesionEntradasCorrectasNoGastanCupo(t *testing.T) {
	h := servidorLimite(t)
	for i := 1; i <= 35; i++ {
		if c := entrarLimite(t, h, fmt.Sprintf("10.3.3.3:%d", 2000+i), "", "ana.navas", "correcta"); c != http.StatusOK {
			t.Fatalf("entrada correcta %d rechazada: %d", i, c)
		}
	}
	// Los fallos siguen contando: cinco por cuenta en la ventana.
	for i := 0; i < 5; i++ {
		entrarLimite(t, h, "10.4.4.4:1", "", "ana.navas", "mala")
	}
	if c := entrarLimite(t, h, "10.4.4.4:1", "", "ana.navas", "correcta"); c != http.StatusTooManyRequests {
		t.Fatalf("cinco fallos seguidos deben limitar la cuenta: %d", c)
	}
}
