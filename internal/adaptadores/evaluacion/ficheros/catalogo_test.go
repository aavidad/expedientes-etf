package ficheros

import (
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	"os"
	"path/filepath"
	"testing"
)

func TestDefinicionesPublicadas(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "evaluacion.reglas")
	catalogo := Catalogo{Ruta: ruta}
	for _, caso := range []struct {
		contenido string
		valida    bool
	}{
		{`{"codigos":["SCORE15"],"escalas":[]}`, true},
		{`{"codigos":["SCORE15"],"escalas":[{"id":"11111111-1111-4111-8111-111111111111","instrumento":"SCORE15","version":1,"items":[{"codigo":"item01","tipo":"numerico","minimo":0,"maximo":2,"paso":1}]}]}`, true},
		{`{"codigos":["SCORE15"],"escalas":[{"id":"11111111-1111-4111-8111-111111111111","instrumento":"SCORE15","version":1,"items":[]}]}`, false},
	} {
		if err := os.WriteFile(ruta, []byte(caso.contenido), 0600); err != nil {
			t.Fatal(err)
		}
		r, err := catalogo.Definiciones(context.Background())
		if caso.valida && (err != nil || len(r.Instrumentos) != 1) {
			t.Fatal(r, err)
		}
		if !caso.valida && !errors.Is(err, d.ErrVersionEscala) {
			t.Fatal(err)
		}
	}
}
