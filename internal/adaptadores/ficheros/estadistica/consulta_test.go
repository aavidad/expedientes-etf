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
