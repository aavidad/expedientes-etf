package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

func TestVozManifiestoAusenteYRecursosLimitados(t *testing.T) {
	sha := strings.Repeat("a", 64)
	audios := fstest.MapFS{"ayuda/manifiesto.json": {Data: []byte(`{"ayuda.disponible":{"fichero":"` + sha + `.mp3"},"ayuda.ausente":{"fichero":"` + strings.Repeat("b", 64) + `.mp3"},"ayuda.ruta":{"fichero":"../secreto"}}`)}, "ayuda/" + sha + ".mp3": {Data: []byte("audio-local")}}
	modulo := ModuloVoz{Audios: audios}
	for _, caso := range []struct {
		grupo, recurso string
		estado         int
	}{
		{"ayuda", "manifiesto.json", 200}, {"manuales", "manifiesto.json", 200}, {"curso", "manifiesto.json", 404},
		{"ayuda", "../secreto", 404}, {"ayuda", sha + ".mp3", 200}, {"ayuda", strings.Repeat("b", 64) + ".mp3", 404},
	} {
		r := httptest.NewRequest("GET", "/voz/"+caso.grupo+"/x", nil)
		r.SetPathValue("grupo", caso.grupo)
		r.SetPathValue("recurso", caso.recurso)
		w := httptest.NewRecorder()
		modulo.audio(w, r)
		if w.Code != caso.estado {
			t.Fatalf("%s/%s: %d", caso.grupo, caso.recurso, w.Code)
		}
		if caso.grupo == "ayuda" && caso.recurso == "manifiesto.json" {
			var datos map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &datos)
			if len(datos) != 1 || datos["ayuda.disponible"] == nil {
				t.Fatal("publica recurso ausente o ruta ajena")
			}
		}
	}
	r := httptest.NewRequest("GET", "/voz/ayuda/x", nil)
	r.SetPathValue("grupo", "ayuda")
	r.SetPathValue("recurso", sha+".mp3")
	r.Header.Set("Range", "bytes=0-4")
	w := httptest.NewRecorder()
	modulo.audio(w, r)
	if w.Code != 206 || w.Body.String() != "audio" {
		t.Fatal("Range no funciona")
	}
}

func TestVozManualesPorPerfilYConsultaVacia(t *testing.T) {
	docs := fstest.MapFS{}
	for _, nombre := range []string{"profesionales", "coordinacion", "administracion"} {
		docs["manuales/web/"+nombre+".json"] = &fstest.MapFile{Data: []byte(`{"titulo":"Manual editorial","secciones":[]}`)}
	}
	modulo := ModuloVoz{Documentos: docs}
	for _, caso := range []struct {
		perfiles []dominio.Perfil
		nombres  string
		cuenta   int
	}{
		{[]dominio.Perfil{dominio.ProfesionalETF}, "profesionales", 1}, {[]dominio.Perfil{dominio.Coordinacion}, "coordinacion", 1},
		{[]dominio.Perfil{dominio.Administracion}, "administracion", 1}, {[]dominio.Perfil{dominio.Consulta}, "", 0},
		{[]dominio.Perfil{dominio.Coordinacion, dominio.ProfesionalETF}, "profesionales", 2},
	} {
		r := httptest.NewRequest("GET", "/api/ayuda/manuales", nil).WithContext(entrada.ConAcceso(context.Background(), entrada.Acceso{Usuario: dominio.Usuario{Perfiles: caso.perfiles}}))
		w := httptest.NewRecorder()
		modulo.manuales(w, r)
		var datos []map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &datos) != nil || len(datos) != caso.cuenta {
			t.Fatal("manuales ajenos al perfil")
		}
		if caso.cuenta > 0 && datos[0]["nombre"] != caso.nombres {
			t.Fatal("manual incorrecto")
		}
	}
}

func TestVozExigeSesionYCSPPermiteMediaLocal(t *testing.T) {
	h, err := NuevoValidado(nil, nil, "es", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/api/ayuda/manuales", "/voz/ayuda/manifiesto.json", "/manuales/recursos/" + strings.Repeat("a", 64) + ".png"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", ruta, nil))
		if w.Code != 401 && w.Code != 403 {
			t.Fatalf("%s: %d", ruta, w.Code)
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "media-src 'self'") {
			t.Fatal("CSP bloquea audio local")
		}
	}
}

func TestVozNoPublicaRecursosDeManualAjeno(t *testing.T) {
	sha := strings.Repeat("a", 64)
	audios := fstest.MapFS{"manuales/manifiesto.json": {Data: []byte(`{"administracion-s0":{"fichero":"` + sha + `.mp3"}}`)}, "manuales/" + sha + ".mp3": {Data: []byte("mp3-local")}}
	docs := fstest.MapFS{"manuales/web/recursos.json": {Data: []byte(`{"` + sha + `.pdf":{"ruta":"manuales/administracion.pdf","manuales":["administracion"]}}`)}, "manuales/administracion.pdf": {Data: []byte("%PDF-editorial")}}
	modulo := ModuloVoz{Audios: audios, Documentos: docs}
	for _, perfil := range []dominio.Perfil{dominio.ProfesionalETF, dominio.Consulta, dominio.Coordinacion, dominio.Administracion} {
		contexto := entrada.ConAcceso(context.Background(), entrada.Acceso{Usuario: dominio.Usuario{Perfiles: []dominio.Perfil{perfil}}})
		r := httptest.NewRequest("GET", "/voz/manuales/manifiesto.json", nil).WithContext(contexto)
		r.SetPathValue("grupo", "manuales")
		r.SetPathValue("recurso", "manifiesto.json")
		w := httptest.NewRecorder()
		modulo.audio(w, r)
		permitido := perfil == dominio.Administracion
		var datos map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &datos)
		if (len(datos) == 1) != permitido {
			t.Fatal("manifiesto de manual ajeno")
		}
		r.SetPathValue("recurso", sha+".mp3")
		w = httptest.NewRecorder()
		modulo.audio(w, r)
		if (w.Code == 200) != permitido {
			t.Fatal("audio de manual ajeno")
		}
		r.SetPathValue("recurso", sha+".pdf")
		w = httptest.NewRecorder()
		modulo.imagen(w, r)
		if (w.Code == 200) != permitido {
			t.Fatal("PDF de manual ajeno")
		}
	}
}
