package pdf_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/pdf"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--lector-anexo-pdf" {
		var reserva []byte
		if os.Getenv("ETF_PDF_PRUEBA_PRESION") == "1" {
			reserva = make([]byte, 80<<20)
			for i := 0; i < len(reserva); i += 4096 {
				reserva[i] = 1
			}
		}
		if err := pdf.EjecutarPDF(os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		runtime.KeepAlive(reserva)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func corpus(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTrabajadorCamposWordYConfirmacion(t *testing.T) {
	for _, nombre := range []string{"anexo3", "anexo4"} {
		t.Run(nombre, func(t *testing.T) {
			e, err := pdf.NuevoLector().Leer(context.Background(), "pdf", corpus(t, nombre+".pdf"))
			if err != nil || !e.PlantillaReconocida || e.Tipo != nombre {
				t.Fatal(e, err)
			}
			w, err := docx.Leer(corpus(t, "../../docx/testdata/"+nombre+".docx"))
			if err != nil {
				t.Fatal(err)
			}
			codigos := map[string]bool{}
			for _, c := range e.Campos {
				codigos[c.Codigo] = true
				if !c.Dudoso || len(c.Motivos) == 0 {
					t.Fatal("dato sin confirmación", c.Codigo)
				}
				if c.Valor != "" && c.Pagina == 0 {
					t.Fatal("dato sin página de origen", c.Codigo)
				}
			}
			for _, c := range w.Campos {
				if len(c.Codigo) >= 9 && c.Codigo[:9] == "valorame." {
					continue
				}
				if !codigos[c.Codigo] {
					t.Error("campo Word ausente", c.Codigo)
				}
			}
		})
	}
}

func TestTrabajadorEscaneadoCorruptoYLimites(t *testing.T) {
	lector := pdf.NuevoLector()
	e, err := lector.Leer(context.Background(), "pdf", corpus(t, "sin_texto.pdf"))
	if err != nil || len(e.Campos) != 0 || e.PlantillaReconocida || !reflect.DeepEqual(e.Avisos, []string{"anexo.pdf_sin_texto"}) {
		t.Fatal(e, err)
	}
	for _, caso := range []struct {
		nombre string
		datos  []byte
		clave  string
	}{
		{"corrupto", corpus(t, "corrupto.pdf"), "anexo.pdf_corrupto"},
		{"grande", make([]byte, pdf.MaxBytes+1), "anexo.limite_bytes"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, err := lector.Leer(context.Background(), "pdf", caso.datos)
			if err == nil || err.Error() != caso.clave || len(e.Campos) != 0 {
				t.Fatal(e, err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := lector.Leer(ctx, "pdf", corpus(t, "anexo3.pdf")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := lector.Leer(context.Background(), "docx", nil); !errors.Is(err, dominio.ErrFormato) {
		t.Fatal(err)
	}
}
