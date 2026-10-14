package pdf

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDiagnosticoSinContenidoDelAnexo(t *testing.T) {
	for _, contenido := range []string{
		"panic: Familia Ortega Salas\n",
		"fatal error: runtime: out of memory: Familia Ortega Salas\n",
		"runtime stack:\nFamilia Ortega Salas\nfatal error: runtime: cannot allocate memory\n",
		strings.Repeat("x", 4096) + "fatal error: runtime: out of memory\n",
	} {
		var stderr diagnosticoAcotado
		n, err := stderr.Write([]byte(contenido))
		if n != len(contenido) || err != nil || stderr.Len() > 4096 {
			t.Fatal(n, err, stderr.Len())
		}
		fallo := &FalloTrabajador{Causa: errors.New("exit status 2"), Diagnostico: stderr.resumen()}
		if fallo.Error() != "anexo.limite_recursos" || !errors.Is(fallo, ErrRecursos) {
			t.Fatal(fallo)
		}
		tecnico := fmt.Sprintf("%+v", fallo)
		if strings.Contains(tecnico, "Familia") || strings.Contains(tecnico, "runtime stack") {
			t.Fatal(tecnico)
		}
		if strings.HasPrefix(contenido, "runtime stack:") && !strings.Contains(tecnico, "fatal error: runtime: cannot allocate memory") {
			t.Fatal(tecnico)
		}
	}
}
