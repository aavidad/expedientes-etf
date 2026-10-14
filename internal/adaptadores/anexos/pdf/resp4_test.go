package pdf_test

import (
	"bytes"
	"context"
	"encoding/json"
	dslipak "github.com/dslipak/pdf"
	"os"
	"reflect"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/pdf"
)

func TestRESP4PDFSinFirmaYFirmado(t *testing.T) {
	var esperado, revision map[string]map[string][]string
	for fichero, dst := range map[string]any{"esperado.json": &esperado, "revision_pdf.json": &revision} {
		b, err := os.ReadFile("../odt/testdata/" + fichero)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(b, dst); err != nil {
			t.Fatal(err)
		}
	}
	for _, nombre := range []string{"anexo3", "anexo4"} {
		t.Run(nombre, func(t *testing.T) {
			var camposSinFirma any
			for _, sufijo := range []string{"", "-firmado"} {
				b, err := os.ReadFile("../odt/testdata/" + nombre + sufijo + ".pdf")
				if err != nil {
					t.Fatal(err)
				}
				lectorMetadata, err := dslipak.NewReader(bytes.NewReader(b), int64(len(b)))
				if err != nil {
					t.Fatal(err)
				}
				if lectorMetadata.Trailer().Key("Info").Key("Author").Text() != "Expedientes ETF" || !lectorMetadata.Trailer().Key("Root").Key("Metadata").IsNull() {
					t.Fatal("metadatos sin sanear")
				}
				if sufijo != "" && (!bytes.Contains(b, []byte("/ByteRange")) || !bytes.Contains(b, []byte("/adbe.pkcs7.detached"))) {
					t.Fatal("firma ausente")
				}
				e, err := pdf.NuevoLector().Leer(context.Background(), "pdf", b)
				if err != nil {
					t.Fatal(err)
				}
				if !e.PlantillaReconocida || e.Tipo != nombre {
					t.Fatal(e)
				}
				valores := map[string][]string{}
				for _, c := range e.Campos {
					if !c.Dudoso {
						t.Fatal("sin confirmación PDF")
					}
					valores[c.Codigo] = append(valores[c.Codigo], c.Valor)
				}
				want := map[string][]string{}
				for k, v := range esperado[nombre] {
					want[k] = v
				}
				for k, v := range revision[nombre] {
					if len(v) != 1 || v[0] != "" {
						t.Fatal("revisión no vacía", k)
					}
					want[k] = v
				}
				if !reflect.DeepEqual(valores, want) {
					for k, w := range want {
						if !reflect.DeepEqual(valores[k], w) {
							t.Errorf("%s: leído %q; esperado %q", k, valores[k], w)
						}
					}
					for k := range valores {
						if _, ok := want[k]; !ok {
							t.Error("campo sobrante", k)
						}
					}
				}
				if sufijo == "" {
					camposSinFirma = e.Campos
				} else if !reflect.DeepEqual(camposSinFirma, e.Campos) {
					t.Fatal("firma altera extracción")
				}
				t.Logf("%s%s: campos=%d revisión_manual=%d", nombre, sufijo, len(e.Campos), len(revision[nombre]))
			}
		})
	}
}
