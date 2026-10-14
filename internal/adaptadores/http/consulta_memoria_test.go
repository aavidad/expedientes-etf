package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type consultaHTTP struct {
	llamadas int
	actor    entrada.ActorMemoria
}

func (s *consultaHTTP) ConsultarCifras(_ context.Context, a entrada.ActorMemoria, f entrada.FiltroMemoria) (entrada.CifrasConsulta, error) {
	s.llamadas++
	s.actor = a
	if err := d.AutorizarConsultaMemoria(a, f); err != nil {
		return entrada.CifrasConsulta{}, err
	}
	return entrada.CifrasConsulta{Anio: f.Anio, EquipoID: f.EquipoID, Apartados: []d.ApartadoConsulta{{Codigo: d.CasosAtendidos, Total: d.TotalConsulta{Familias: 2}}}}, nil
}
func (s *consultaHTTP) DescargarCifras(ctx context.Context, a entrada.ActorMemoria, f entrada.FiltroMemoria, _ string) ([]byte, error) {
	_, err := s.ConsultarCifras(ctx, a, f)
	return []byte("PK-cifras"), err
}

type navegacionHTTP struct{ llamadas int }

func (s *navegacionHTTP) ConsultarMemoria(_ context.Context, a entrada.ActorMemoria, f entrada.FiltroMemoria) (entrada.VistaMemoria, error) {
	s.llamadas++
	f, err := d.AutorizarMemoria(a, f)
	return entrada.VistaMemoria{FiltroMemoria: f}, err
}
func (s *navegacionHTTP) ComposicionMemoria(_ context.Context, a entrada.ActorMemoria, f entrada.FiltroMemoria, ap entrada.Apartado) ([]entrada.DesgloseMemoria, error) {
	s.llamadas++
	_, err := d.AutorizarMemoria(a, f)
	return []entrada.DesgloseMemoria{{Apartado: ap, Numeros: []string{"08-02-26"}}}, err
}
func TestConsultaMemoriaHTTPNegativasYContratoNumerico(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo, ruta, cuerpo string
		perfil                       identidad.Perfil
		cookie, csrf                 bool
		estado                       int
	}{
		{"cifras provincia", "GET", "/api/consulta/memoria/2026", "", identidad.Consulta, true, false, 200},
		{"cifras equipo", "GET", "/api/consulta/memoria/2025?equipo=33333333-3333-4333-8333-333333333333", "", identidad.Consulta, true, false, 200},
		{"sin sesión", "GET", "/api/consulta/memoria/2026", "", identidad.Consulta, false, false, 401},
		{"administración", "GET", "/api/consulta/memoria/2026", "", identidad.Administracion, true, false, 403},
		{"sin perfil", "GET", "/api/consulta/memoria/2026", "", "", true, false, 403},
		{"composición prohibida", "GET", "/api/memoria/2026/cifras/1?etf=33333333-3333-4333-8333-333333333333&perfil=coordinacion", "", identidad.Consulta, true, false, 403},
		{"informe prohibido", "GET", "/api/memoria/2026", "", identidad.Consulta, true, false, 403},
		{"desglose inyectado", "GET", "/api/consulta/memoria/2026?composicion=true", "", identidad.Consulta, true, false, 400},
		{"perfil inyectado", "GET", "/api/consulta/memoria/2026?perfil=coordinacion", "", identidad.Consulta, true, false, 400},
		{"narrativas inyectadas", "GET", "/api/consulta/memoria/2026?narrativas=true", "", identidad.Consulta, true, false, 400},
		{"fuentes inyectadas", "GET", "/api/consulta/memoria/2026?fuentes=true&revision=1", "", identidad.Consulta, true, false, 400},
		{"año repetido", "GET", "/api/consulta/memoria/2026?anio=2025&anio=2026", "", identidad.Consulta, true, false, 400},
		{"equipo repetido", "GET", "/api/consulta/memoria/2026?equipo=a&equipo=b", "", identidad.Consulta, true, false, 400},
		{"equipo malformado", "GET", "/api/consulta/memoria/2026?equipo=otro", "", identidad.Consulta, true, false, 400},
		{"año malformado", "GET", "/api/consulta/memoria/no", "", identidad.Consulta, true, false, 400},
		{"método prohibido", "PUT", "/api/consulta/memoria/2026", "{}", identidad.Consulta, true, true, 405},
		{"Word cifras", "POST", "/api/consulta/memoria/2026/word", `{"idioma":"es"}`, identidad.Consulta, true, true, 200},
		{"Word sin CSRF", "POST", "/api/consulta/memoria/2026/word", `{"idioma":"es"}`, identidad.Consulta, true, false, 403},
		{"Word narra", "POST", "/api/consulta/memoria/2026/word", `{"idioma":"es","narrativas":true}`, identidad.Consulta, true, true, 400},
		{"Word compone", "POST", "/api/consulta/memoria/2026/word?composicion=true", `{"idioma":"es"}`, identidad.Consulta, true, true, 400},
		{"Word atribuye usuario", "POST", "/api/consulta/memoria/2026/word", `{"idioma":"es","usuario_id":"33333333-3333-4333-8333-333333333333"}`, identidad.Consulta, true, true, 400},
		{"Word de coordinación", "POST", "/api/estadistica/memoria/word", `{"anio":2026,"idioma":"es"}`, identidad.Consulta, true, true, 403},
		{"equipo ajeno profesional", "GET", "/api/memoria/2026?etf=33333333-3333-4333-8333-333333333333", "", identidad.ProfesionalETF, true, false, 403},
		{"números coordinación", "GET", "/api/memoria/2026/cifras/1", "", identidad.Coordinacion, true, false, 200},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			consulta := &consultaHTTP{}
			nav := &navegacionHTTP{}
			cfg := ConfiguracionSesion{Servicio: &sesionCoordinacion{perfiles: []identidad.Perfil{caso.perfil}}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			h, err := NuevoValidado(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Modulos: []Modulo{ModuloConsultaMemoria{consulta}, ModuloNavegacionMemoria{nav}, ModuloMemoriaWord{ConfiguracionMemoriaWord{&memoriaWordPrueba{}}}}})
			if err != nil {
				t.Fatal(err)
			}
			token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			r := httptest.NewRequest(caso.metodo, "https://etf.interno"+caso.ruta, strings.NewReader(caso.cuerpo))
			if caso.cookie {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			}
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Perfil", "coordinacion")
			if caso.csrf {
				r.Header.Set("Origin", cfg.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(token))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatalf("estado %d esperado %d: %s", w.Code, caso.estado, w.Body.String())
			}
			if w.Code == 200 && strings.HasPrefix(caso.ruta, "/api/consulta/") && caso.metodo == "GET" {
				var mapa map[string]any
				if json.Unmarshal(w.Body.Bytes(), &mapa) != nil {
					t.Fatal("JSON")
				}
				for _, clave := range []string{"Fuentes", "fuentes", "Narrativas", "narrativas", "Composicion", "composicion", "HuellaFuentes"} {
					if _, ok := mapa[clave]; ok {
						t.Fatal("campo filtrado", clave)
					}
				}
				if strings.Join(consulta.actor.Perfiles, ",") != "consulta" || w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("perfil o caché")
				}
			}
			if w.Code == 403 && consulta.llamadas != 0 {
				t.Fatal("se llamó Consulta sin permiso")
			}
			if caso.nombre == "composición prohibida" && nav.llamadas != 0 {
				t.Fatal("se calculó composición para Consulta")
			}
		})
	}
}

func (*consultaHTTP) RechazarPeticionMemoria(context.Context, entrada.ActorMemoria) error { return nil }
func (*navegacionHTTP) RechazarPeticionMemoria(context.Context, entrada.ActorMemoria) error {
	return nil
}
