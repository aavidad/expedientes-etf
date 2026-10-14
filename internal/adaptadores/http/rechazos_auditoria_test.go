package http

// Regresión F5.02: las peticiones anónimas rechazadas en rutas protegidas no
// pueden escribir en la cadena de auditoría sin presupuesto por origen.

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type operacionesContadas struct{ apuntes atomic.Int64 }

func (o *operacionesContadas) EjecutarIdentidad(ctx context.Context, f func(context.Context) error) error {
	return f(ctx)
}
func (o *operacionesContadas) RegistrarFallo(context.Context, string, string) error {
	o.apuntes.Add(1)
	return nil
}

func servidorRechazos(t *testing.T) (http.Handler, *operacionesContadas) {
	t.Helper()
	l, err := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: 15 * time.Minute, PorUsuario: 5, PorIP: 30, MaxClaves: 10000}, dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 100, MaxClaves: 10000})
	if err != nil {
		t.Fatal(err)
	}
	ops := &operacionesContadas{}
	s, err := aplicacion.NuevoServicio(directorioLimite{}, &usuariosLimite{por: map[dominio.GUID]dominio.Usuario{}}, &sesionesLimite{por: map[dominio.Huella]dominio.Sesion{}}, relojLimite{}, azarLimite{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: 8 * time.Hour}, l, ops)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
	return Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg}), ops
}

func TestRechazosAnonimosNoEscribenAuditoriaSinPresupuesto(t *testing.T) {
	inventada := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	for _, cookie := range []string{"", inventada} {
		h, ops := servidorRechazos(t)
		limitadas := 0
		for i := 0; i < 250; i++ {
			r := httptest.NewRequest(http.MethodGet, "https://etf.interno/api/admin/usuarios", nil)
			r.RemoteAddr = "192.0.2.50:4000"
			if cookie != "" {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code == http.StatusTooManyRequests {
				limitadas++
			} else if w.Code != http.StatusUnauthorized {
				t.Fatalf("estado inesperado %d", w.Code)
			}
		}
		if n := ops.apuntes.Load(); n > 100 || limitadas == 0 {
			t.Fatalf("con cookie %t: %d apuntes de auditoría y %d respuestas limitadas", cookie != "", n, limitadas)
		}
		// Otro origen sigue recibiendo su propio rechazo auditado.
		r := httptest.NewRequest(http.MethodGet, "https://etf.interno/api/admin/usuarios", nil)
		r.RemoteAddr = "192.0.2.51:4000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("otro origen: %d", w.Code)
		}
	}
}
