package documento_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

func TestConsentimientosWordSinDNIRepeticionYEquipo(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		for _, tipo := range []string{d.Anexo1, d.Anexo2} {
			t.Run(idioma+"/"+tipo, func(t *testing.T) {
				_, deps, repo, fuente, actor, p := preparar(t)
				p.Tipo, p.Idioma = tipo, idioma
				b, err := os.ReadFile("../../../idiomas/" + idioma + "/plantillas/" + tipo + "/v1.0.0.docx")
				if err != nil {
					t.Fatal(err)
				}
				repo.plantilla.Tipo, repo.plantilla.Idioma, repo.plantilla.Ruta, repo.plantilla.Huella = tipo, idioma, "plantillas/"+tipo+"/v1.0.0.docx", d.Huella(b)
				actor.Perfiles, actor.EquipoID = []string{"profesional_etf"}, "66666666-6666-4666-8666-666666666666"
				fuente.datos.EquipoID = actor.EquipoID
				fuente.datos.Aceptacion.Selecciones = map[string]string{"fecha_documento": "13/10/2026", "menores_lista": "Nerea Ortega Salas"}
				s := app.Nuevo(deps)
				r, err := s.Emitir(t.Context(), actor, p)
				if err != nil {
					t.Fatal(err)
				}
				xml := textoWord(t, r.Contenido)
				if strings.Contains(xml, "{{") || strings.Contains(xml, "número de DNI") || !strings.Contains(xml, "Ortega Salas") || !strings.Contains(xml, "13/10/2026") || !strings.Contains(xml, "____________________") {
					t.Fatal("relleno o hueco DNI incorrecto")
				}
				repetida, err := s.Emitir(t.Context(), actor, p)
				if err != nil || !repetida.Repetida || !bytes.Equal(r.Contenido, repetida.Contenido) || len(repo.emisiones) != 1 {
					t.Fatal("repetición", err)
				}
				actor.EquipoID = "66666666-6666-4666-8666-666666666667"
				if _, err := s.Emitir(t.Context(), actor, p); !errors.Is(err, d.ErrAcceso) {
					t.Fatal("equipo ajeno", err)
				}
			})
		}
	}
}
