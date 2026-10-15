package estadistica_test

import (
	"context"
	"encoding/json"
	f "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"os"
	"path/filepath"
	"testing"
)

func TestTextosCifrasUsaReferenciaInmutableDelIdioma(t *testing.T) {
	raiz := t.TempDir()
	for _, ruta := range []string{"es", ".versiones/es/v2/es"} {
		directorio := filepath.Join(raiz, ruta, "textos")
		if err := os.MkdirAll(directorio, 0700); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(map[string]string{"estadistica.consulta_fecha": ruta, "estadistica.consulta_definitiva": ruta, "estadistica.consulta_provisional": ruta})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directorio, "consulta_memoria.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	lector := f.TextosCifras{Raiz: raiz}
	textos, err := lector.LeerTextosCifras(context.Background(), ".versiones/es/v2/es")
	if err != nil || textos["estadistica.consulta_fecha"] != ".versiones/es/v2/es" {
		t.Fatal("ignora versión activa", textos, err)
	}
	for _, ruta := range []string{"../es", "es/../en", ".versiones/en/v2/es"} {
		if _, err := lector.LeerTextosCifras(context.Background(), ruta); err == nil {
			t.Fatal("ruta no permitida", ruta)
		}
	}
}

func TestTextosCifrasCompletaClavesNuevasDesdeLaDistribucion(t *testing.T) {
	raiz := t.TempDir()
	escribir := func(ruta string, textos map[string]string) {
		t.Helper()
		directorio := filepath.Join(raiz, ruta, "textos")
		if err := os.MkdirAll(directorio, 0700); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(textos)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directorio, "consulta_memoria.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir("es", map[string]string{"estadistica.consulta_fecha": "base", "estadistica.consulta_definitiva": "base", "estadistica.consulta_provisional": "base", "estadistica.consulta_oculto": "Dato oculto"})
	escribir(".versiones/es/v2/es", map[string]string{"estadistica.consulta_fecha": "publicada", "estadistica.consulta_definitiva": "publicada", "estadistica.consulta_provisional": "publicada"})
	lector := f.TextosCifras{Raiz: raiz}
	textos, err := lector.LeerTextosCifras(context.Background(), ".versiones/es/v2/es")
	if err != nil || textos["estadistica.consulta_fecha"] != "publicada" || textos["estadistica.consulta_oculto"] != "Dato oculto" {
		t.Fatal("no se completó la clave nueva o se sustituyó la publicada", textos, err)
	}
	textos, err = lector.LeerTextosCifras(context.Background(), "es")
	if err != nil || textos["estadistica.consulta_fecha"] != "base" {
		t.Fatal("distribución sin versión publicada", textos, err)
	}
	// Un idioma sin la clave en ninguna de sus capas la toma del castellano.
	escribir("en", map[string]string{"estadistica.consulta_fecha": "Date", "estadistica.consulta_definitiva": "Final", "estadistica.consulta_provisional": "Provisional"})
	textos, err = lector.LeerTextosCifras(context.Background(), "en")
	if err != nil || textos["estadistica.consulta_fecha"] != "Date" || textos["estadistica.consulta_oculto"] != "Dato oculto" {
		t.Fatal("sin reserva del castellano", textos, err)
	}
	// Una capa de reserva dañada no se ignora.
	if err := os.WriteFile(filepath.Join(raiz, "es", "textos", "consulta_memoria.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := lector.LeerTextosCifras(context.Background(), ".versiones/es/v2/es"); err == nil {
		t.Fatal("distribución dañada aceptada")
	}
}
