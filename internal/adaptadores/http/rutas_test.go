package http

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

func TestTodasLasRutasDelMuxTienenRegla(t *testing.T) {
	mux, _ := nuevoMux(nil, nil, "", "", nil, nil, nil, Opciones{})
	reglas := tablaRutas()
	for _, patron := range mux.patrones {
		if !slices.ContainsFunc(reglas, func(r reglaRuta) bool { return r.Patron == patron }) {
			t.Errorf("ruta registrada sin regla: %s", patron)
		}
	}
	pares := make(map[string]bool)
	for _, regla := range reglas {
		if !slices.Contains(mux.patrones, regla.Patron) {
			t.Errorf("regla sin ruta: %s", regla.Patron)
		}
		if len(regla.Metodos) == 0 || regla.Publica == (len(regla.Perfiles) > 0) {
			t.Errorf("regla incompleta: %+v", regla)
		}
		for _, metodo := range regla.Metodos {
			par := metodo + " " + regla.Patron
			if pares[par] {
				t.Errorf("regla duplicada: %s", par)
			}
			pares[par] = true
			if metodo != "GET" && metodo != "HEAD" && !regla.CSRF {
				t.Errorf("escritura sin CSRF: %s", par)
			}
		}
		if strings.HasPrefix(regla.Patron, "/api/admin/") || strings.HasPrefix(regla.Patron, "/admin/") {
			if regla.Publica || !slices.Equal(regla.Perfiles, []string{"administracion"}) {
				t.Errorf("permiso administrativo incorrecto: %+v", regla)
			}
		}
		if strings.HasPrefix(regla.Patron, "/api/coordinacion/") {
			if regla.Publica || !slices.Equal(regla.Perfiles, []string{"coordinacion"}) {
				t.Errorf("permiso de coordinación incorrecto: %+v", regla)
			}
		}
		if regla.Patron == "/api/documentos/emisiones" && (regla.Publica || !regla.ExigeAmbito || !slices.Equal(regla.Perfiles, []string{"coordinacion", "profesional_etf"})) {
			t.Errorf("permiso Anexo 5 incorrecto: %+v", regla)
		}
		if strings.HasPrefix(regla.Patron, "/api/documentos/") && regla.Patron != "/api/documentos/emisiones" && (regla.Publica || regla.ExigeAmbito || !slices.Equal(regla.Perfiles, []string{"coordinacion"})) {
			t.Errorf("ámbito o perfiles de emisión incorrectos: %+v", regla)
		}
	}
	t.Logf("%d rutas registradas; %d reglas de método cubiertas", len(mux.patrones), len(pares))
}

func caminoPrueba(patron string) string {
	partes := strings.Split(patron, "/")
	for i, parte := range partes {
		if strings.HasPrefix(parte, "{") {
			partes[i] = "valor"
		}
		if parte == "{$}" {
			partes[i] = ""
		}
	}
	return strings.Join(partes, "/")
}

func TestMatrizDePerfilesGeneradaDesdeTabla(t *testing.T) {
	casos := []struct {
		nombre string
		acceso entrada.Acceso
		cookie bool
	}{
		{nombre: "anonimo"},
		{nombre: "pendiente", acceso: entrada.Acceso{Estado: dominio.PendienteAlta}, cookie: true},
		{nombre: "administracion", acceso: accesoPrueba(dominio.Administracion), cookie: true},
		{nombre: "coordinacion", acceso: accesoPrueba(dominio.Coordinacion), cookie: true},
		{nombre: "profesional_etf", acceso: accesoPrueba(dominio.ProfesionalETF), cookie: true},
		{nombre: "consulta", acceso: accesoPrueba(dominio.Consulta), cookie: true},
		{nombre: "multiples", acceso: accesoPrueba(dominio.Administracion, dominio.Coordinacion), cookie: true},
	}
	sinEquipo := accesoPrueba(dominio.ProfesionalETF)
	sinEquipo.Usuario.EquipoID = ""
	casos = append(casos, struct {
		nombre string
		acceso entrada.Acceso
		cookie bool
	}{"profesional_sin_equipo", sinEquipo, true})
	cantidad := 0
	for _, regla := range tablaRutas() {
		for _, metodo := range regla.Metodos {
			for _, caso := range casos {
				t.Run(caso.nombre+"/"+metodo+regla.Patron, func(t *testing.T) {
					cfg := sesionAdministracionPrueba()
					s := &sesionAutorizacionPrueba{acceso: caso.acceso}
					cfg.Servicio = s
					sesion := nuevaSesionHTTP(cfg)
					llamadas := 0
					h := sesion.autorizar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						llamadas++
						if !regla.Publica && entrada.AccesoActual(r.Context()).Usuario.ID != caso.acceso.Usuario.ID {
							t.Fatal("actor distinto de la sesión")
						}
						w.WriteHeader(204)
					}))
					r := httptest.NewRequest(metodo, "https://etf.interno"+caminoPrueba(regla.Patron), nil)
					secreto := cookieAdministracionPrueba().Value
					if caso.cookie {
						r.AddCookie(cookieAdministracionPrueba())
					} else {
						r.AddCookie(&http.Cookie{Name: cookieCSRF, Value: secreto})
					}
					r.Header.Set("Origin", cfg.Origen)
					r.Header.Set("X-CSRF-Token", sesion.testigo(secreto))
					// Estos valores del cliente nunca conceden permisos.
					r.Header.Set("X-Perfil", "administracion")
					r.Header.Set("X-Equipo", "otro-equipo")
					permitido := regla.Publica
					for _, perfil := range regla.Perfiles {
						if caso.cookie && caso.acceso.Permite(perfil) && (!regla.ExigeAmbito || perfil != "profesional_etf" || caso.acceso.Usuario.EquipoID != "") {
							permitido = true
						}
					}
					esperado := 403
					if permitido {
						esperado = 204
					} else if !caso.cookie {
						esperado = 401
					}
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != esperado || (llamadas == 1) != permitido {
						t.Fatalf("estado=%d llamadas=%d; esperado=%d", w.Code, llamadas, esperado)
					}
					if !regla.Publica && caso.cookie && !permitido && s.actorRechazo != caso.acceso.Usuario.ID {
						t.Fatalf("rechazo sin actor de sesión: %q", s.actorRechazo)
					}
					if !regla.Publica && caso.cookie && s.consultas != 1 {
						t.Fatalf("consultas de sesión=%d", s.consultas)
					}
				})
				cantidad++
			}
		}
	}
	t.Logf("%d combinaciones de perfil, ruta y método", cantidad)
}

func TestMetodosYCSRFProcedenDeTabla(t *testing.T) {
	for _, regla := range tablaRutas() {
		cfg := sesionAdministracionPrueba()
		if len(regla.Perfiles) > 0 {
			cfg.Servicio = &sesionAutorizacionPrueba{acceso: accesoPrueba(dominio.Perfil(regla.Perfiles[0]))}
		}
		sesion := nuevaSesionHTTP(cfg)
		h := sesion.autorizar(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
		ruta := "https://etf.interno" + caminoPrueba(regla.Patron)
		for _, metodo := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			permitido := false
			var permitidos []string
			for _, otra := range tablaRutas() {
				if otra.Patron == regla.Patron {
					permitidos = append(permitidos, otra.Metodos...)
					permitido = permitido || slices.Contains(otra.Metodos, metodo)
				}
			}
			if permitido {
				continue
			}
			r := httptest.NewRequest(metodo, ruta, nil)
			r.AddCookie(cookieAdministracionPrueba())
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 405 || w.Header().Get("Allow") != strings.Join(permitidos, ", ") {
				t.Errorf("%s %s: %d Allow=%s", metodo, regla.Patron, w.Code, w.Header().Get("Allow"))
			}
		}
		if !regla.CSRF {
			continue
		}
		for _, metodo := range regla.Metodos {
			for _, origen := range []string{"", "https://ajeno.interno", cfg.Origen} {
				r := httptest.NewRequest(metodo, ruta, nil)
				r.AddCookie(cookieAdministracionPrueba())
				r.Header.Set("Origin", origen)
				if origen != cfg.Origen {
					r.Header.Set("X-CSRF-Token", sesion.testigo(cookieAdministracionPrueba().Value))
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != 403 || !strings.Contains(w.Body.String(), "identidad.csrf_no_valido") {
					t.Errorf("CSRF %s %s: %d %s", metodo, regla.Patron, w.Code, w.Body.String())
				}
			}
		}
	}
}

func TestRutaRegistradaSinReglaSigueDenegada(t *testing.T) {
	mux := nuevoMuxHTTP()
	llamadas := 0
	mux.HandleFunc("/api/futura", func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(204) })
	h := nuevaSesionHTTP(sesionAdministracionPrueba()).autorizar(mux)
	for _, ruta := range []string{"/api/futura", "/pantalla-futura", "/admin/futura", "/api/admin/futura", "/api/coordinacion/futura", "/api/idiomas/es/futura", "/api/temas/tema/futura", "//", "//admin/usuarios", "/admin/../admin/usuarios", "/api//admin/usuarios", "/api/admin/catalogos/../usuarios"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", ruta, nil)
		r.AddCookie(cookieAdministracionPrueba())
		h.ServeHTTP(w, r)
		if w.Code != 403 && w.Code != 404 {
			t.Errorf("%s: %d", ruta, w.Code)
		}
	}
	if llamadas != 0 {
		t.Fatal("ruta sin regla ejecutada")
	}
}

func TestEmisionCoordinacionUsaUnaConsultaDeSesion(t *testing.T) {
	cfg := sesionAdministracionPrueba()
	servicio := &sesionAutorizacionPrueba{acceso: accesoPrueba(dominio.Coordinacion)}
	cfg.Servicio = servicio
	emision := &emisionesPrueba{}
	h := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Documentos: ConfiguracionDocumentos{Servicio: emision}})
	r := httptest.NewRequest("POST", "/api/documentos/emisiones", bytes.NewBufferString(`{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`))
	autorizarPeticionPrueba(r)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "solicitud-profesional")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || emision.llamadas != 1 || servicio.consultas != 1 || emision.actor.EquipoID != servicio.acceso.Usuario.EquipoID {
		t.Fatalf("%d llamadas=%d consultas=%d equipo=%s: %s", w.Code, emision.llamadas, servicio.consultas, emision.actor.EquipoID, w.Body.String())
	}
}
