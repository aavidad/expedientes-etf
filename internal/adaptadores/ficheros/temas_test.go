package ficheros

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

func copiarTema(t *testing.T, base, nombre string) {
	t.Helper()
	dir := filepath.Join(base, nombre)
	if err := os.MkdirAll(filepath.Join(dir, "fuentes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, fichero := range []string{"tema.json", "tokens.json", "fuentes/public-sans.woff2", "fuentes/source-serif-4.woff2"} {
		b, err := os.ReadFile("../../../temas/expedientes-etf/" + fichero)
		if err != nil {
			t.Fatal(err)
		}
		if fichero == "tema.json" {
			var datos map[string]any
			if err := json.Unmarshal(b, &datos); err != nil {
				t.Fatal(err)
			}
			datos["identificador"] = nombre
			b, err = json.Marshal(datos)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, fichero), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTemasIncluidosCumplenContraste(t *testing.T) {
	p := NuevosPaquetes("../../../idiomas", "../../../temas")
	lista, err := p.Temas(context.Background())
	if err != nil || len(lista) != 2 {
		t.Fatalf("temas: %v %v", lista, err)
	}
	for _, tema := range lista {
		tokens, err := p.Tokens(context.Background(), tema.Identificador)
		if err != nil || !dominio.TokensValidos(tokens, tema.Tipo) {
			t.Fatalf("contraste de %s: %v", tema.Identificador, err)
		}
	}
}

func TestTemaRechazaRecursosActivosFalsosYEnlaces(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		datos  []byte
		enlace bool
	}{
		{"icono.svg", []byte(`<svg/>`), false}, {"estilo.css", []byte(`body{}`), false}, {"activo.js", []byte(`alert(1)`), false},
		{"icono.png", []byte(`no es PNG`), false}, {"fuente.woff2", []byte(`wOF2`), false}, {"oculto.png", nil, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			base := t.TempDir()
			copiarTema(t, base, "costa")
			dir := filepath.Join(base, "costa")
			if caso.enlace {
				if err := os.Symlink("tokens.json", filepath.Join(dir, caso.nombre)); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(filepath.Join(dir, caso.nombre), caso.datos, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NuevosPaquetes(base, base).Tokens(context.Background(), "costa"); err == nil {
				t.Fatal("recurso rechazable aceptado")
			}
		})
	}
}
func TestPNGValidoYRecursosNoEscapan(t *testing.T) {
	base := t.TempDir()
	copiarTema(t, base, "costa")
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "costa", "icono.png"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(base, base)
	r, err := p.RecursoTema(context.Background(), "costa", "icono.png")
	if err != nil || r.Tipo != "image/png" || !bytes.Equal(r.Contenido, b.Bytes()) {
		t.Fatalf("PNG: %v", err)
	}
	for _, ruta := range []string{"../costa/icono.png", "/etc/passwd", "tokens.json", "fuentes/../../icono.png"} {
		if _, err := p.RecursoTema(context.Background(), "costa", ruta); err == nil {
			t.Fatalf("ruta aceptada: %s", ruta)
		}
	}
}
func TestReservaPredeterminadaSiDiscoFalla(t *testing.T) {
	reserva := fstest.MapFS{}
	for _, fichero := range []string{"tema.json", "tokens.json", "fuentes/public-sans.woff2", "fuentes/source-serif-4.woff2"} {
		b, err := os.ReadFile("../../../temas/expedientes-etf/" + fichero)
		if err != nil {
			t.Fatal(err)
		}
		reserva["expedientes-etf/"+fichero] = &fstest.MapFile{Data: b}
	}
	base := t.TempDir()
	p := NuevosPaquetesConReserva(base, base, reserva)
	for _, conDisco := range []bool{false, true} {
		if conDisco {
			copiarTema(t, base, "expedientes-etf")
			if err := os.WriteFile(filepath.Join(base, "expedientes-etf", "tokens.json"), []byte(`{"color.fondo":"#ffffff","color.texto":"#ffffff"}`), 0600); err != nil {
				t.Fatal(err)
			}
		}
		recurso, err := p.RecursoTema(context.Background(), "expedientes-etf", "fuentes/public-sans.woff2")
		if err != nil || recurso.Tipo != "font/woff2" {
			t.Fatalf("fuente de reserva: %v", err)
		}
		tokens, err := p.Tokens(context.Background(), "expedientes-etf")
		if err != nil || !dominio.TokensValidos(tokens, "claro") {
			t.Fatalf("reserva: %v", err)
		}
		lista, err := p.Temas(context.Background())
		if err != nil || len(lista) != 1 {
			t.Fatalf("lista reserva: %v %v", lista, err)
		}
	}
}
func TestJSONRechazaDuplicadosYCamposDesconocidos(t *testing.T) {
	for _, datos := range []string{
		`{"codigo":"es","codigo":"en","nombre":"Castellano","version":"0.1.0"}`,
		`{"codigo":"es","nombre":"Castellano","version":"0.1.0","activo":true}`,
	} {
		var idioma dominio.Idioma
		if err := decodificarJSON([]byte(datos), &idioma); err == nil {
			t.Fatal("JSON ambiguo aceptado")
		}
	}
}

func TestWOFF2RechazaCabeceraCoherenteSinTablas(t *testing.T) {
	b := make([]byte, 49)
	copy(b, "wOF2")
	binary.BigEndian.PutUint32(b[4:8], 0x00010000)
	binary.BigEndian.PutUint32(b[8:12], 49)
	binary.BigEndian.PutUint16(b[12:14], 1)
	binary.BigEndian.PutUint32(b[16:20], 1)
	binary.BigEndian.PutUint32(b[20:24], 1)
	if woff2Valida(b) {
		t.Fatal("cabecera sin tablas aceptada")
	}
	for _, nombre := range []string{"public-sans", "source-serif-4"} {
		b, err := os.ReadFile("../../../temas/expedientes-etf/fuentes/" + nombre + ".woff2")
		if err != nil {
			t.Fatal(err)
		}
		if !woff2Valida(b) {
			t.Fatal("fuente incluida rechazada")
		}
		truncada := append([]byte(nil), b[:len(b)-100]...)
		binary.BigEndian.PutUint32(truncada[8:12], uint32(len(truncada)))
		if woff2Valida(truncada) {
			t.Fatal("fuente truncada aceptada")
		}
	}
}
