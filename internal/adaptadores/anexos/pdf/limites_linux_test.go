package pdf_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/pdf"
)

func TestLimitesProceso(t *testing.T) {
	if os.Getenv("ETF_PDF_PRUEBA_LIMITES") == "1" {
		if err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{Cur: 2, Max: 10}); err != nil {
			t.Fatal(err)
		}
		// El PDF corrupto sirve para instalar límites sin extraer datos.
		if err := pdf.EjecutarPDF(bytes.NewReader([]byte("%PDF-")), io.Discard); err != nil {
			t.Fatal(err)
		}
		limites := map[int]syscall.Rlimit{}
		for _, recurso := range []int{syscall.RLIMIT_AS, syscall.RLIMIT_DATA, syscall.RLIMIT_CPU, syscall.RLIMIT_CORE, syscall.RLIMIT_FSIZE} {
			var r syscall.Rlimit
			if err := syscall.Getrlimit(recurso, &r); err != nil {
				t.Fatal(err)
			}
			limites[recurso] = r
		}
		if err := json.NewEncoder(os.Stdout).Encode(limites); err != nil {
			t.Fatal(err)
		}
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestLimitesProceso$")
	cmd.Env = []string{"ETF_PDF_PRUEBA_LIMITES=1", "GOMEMLIMIT=64MiB", "GOMAXPROCS=1"}
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	var limites map[int]syscall.Rlimit
	if err := json.Unmarshal(b, &limites); err != nil {
		t.Fatal(err)
	}
	for recurso, maximo := range map[int]uint64{syscall.RLIMIT_AS: 4 << 30, syscall.RLIMIT_DATA: 256 << 20, syscall.RLIMIT_CPU: 2, syscall.RLIMIT_CORE: 0, syscall.RLIMIT_FSIZE: 0} {
		r := limites[recurso]
		if r.Cur > maximo || r.Max != r.Cur {
			t.Fatalf("recurso=%d límites=%+v máximo=%d", recurso, r, maximo)
		}
	}
}

func TestTrabajadorConPresionDeMemoria(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "--lector-anexo-pdf")
	cmd.Env = []string{"ETF_PDF_PRUEBA_PRESION=1", "GOMEMLIMIT=64MiB", "GOGC=20", "GOMAXPROCS=1", "GOTRACEBACK=none"}
	cmd.Stdin = bytes.NewReader(corpus(t, "anexo4.pdf"))
	var salida, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &salida, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v: %s", err, stderr.Bytes())
	}
	if !json.Valid(salida.Bytes()) || bytes.Contains(salida.Bytes(), []byte(`"Error":"anexo.`)) {
		t.Fatalf("salida inesperada: %s", salida.Bytes())
	}
}
