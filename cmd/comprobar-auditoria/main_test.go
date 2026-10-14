package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

func fichero(t *testing.T) (string, auditoria.Exportacion) {
	t.Helper()
	d := auditoria.Datos{OcurridoEn: time.Date(2026, 10, 13, 8, 42, 11, 0, time.UTC), UsuarioID: "sistema", Perfiles: []string{}, Accion: "auditoria.exportar", Entidad: "auditoria", Resultado: "ok"}
	e, err := auditoria.Encadenar(1, d, auditoria.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := auditoria.NuevaExportacion([]auditoria.Evento{e}, auditoria.Inicio(), e.Punto())
	if err != nil {
		t.Fatal(err)
	}
	b, err := exportacion.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "auditoria.json")
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta, exportacion
}

func comprobar(t *testing.T, args []string, codigo int, clave string) resultado {
	t.Helper()
	var salida bytes.Buffer
	if obtenido := ejecutar(args, &salida); obtenido != codigo {
		t.Fatalf("código %d, esperado %d: %s", obtenido, codigo, salida.String())
	}
	var r resultado
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Clave != clave {
		t.Fatalf("clave %s, esperada %s", r.Clave, clave)
	}
	return r
}

func TestComandoValidoYSelloExterno(t *testing.T) {
	ruta, e := fichero(t)
	comprobar(t, []string{ruta}, 0, "auditoria.cadena_coherente")
	comprobar(t, []string{ruta, strconv.FormatUint(e.Final.ID, 10), e.Final.Huella}, 0, "auditoria.sello_externo_coincide")
	comprobar(t, []string{ruta, "1", strings.Repeat("a", 64)}, 1, "auditoria.sello_externo_no_coincide")
	// Truncar y cambiar también el sello del fichero conserva su coherencia,
	// pero deja de coincidir con el sello custodiado por separado.
	vacia, err := auditoria.NuevaExportacion(nil, auditoria.Inicio(), auditoria.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	b, err := vacia.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	comprobar(t, []string{ruta}, 0, "auditoria.cadena_coherente")
	comprobar(t, []string{ruta, "1", e.Final.Huella}, 1, "auditoria.sello_externo_no_coincide")
}

func TestComandoPrimerEventoAlterado(t *testing.T) {
	ruta, e := fichero(t)
	e.Eventos[0].Resultado = "denegado"
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	r := comprobar(t, []string{ruta}, 1, "auditoria.huella_invalida")
	if r.Indice == nil || *r.Indice != 0 || r.EventoID != 1 {
		t.Fatalf("primer fallo: %+v", r)
	}
}

func TestSelloExternoDetectaBorradoDelPrefijo(t *testing.T) {
	ruta, e := fichero(t)
	// Se omite el único evento y se finge un tramo vacío después de él,
	// conservando el sello final original.
	parcial, err := auditoria.NuevaExportacion(nil, e.Final, e.Final)
	if err != nil {
		t.Fatal(err)
	}
	b, err := parcial.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	comprobar(t, []string{ruta}, 0, "auditoria.cadena_coherente")
	comprobar(t, []string{ruta, "1", e.Final.Huella}, 1, "auditoria.ancla_externa_no_coincide")
	// El tramo vacío sí es válido si ambos límites se esperan por separado.
	comprobar(t, []string{ruta, "1", e.Final.Huella, "1", e.Final.Huella}, 0, "auditoria.sello_externo_coincide")
}

func TestComandoTramoParcialConAmbosLimites(t *testing.T) {
	ruta, e := fichero(t)
	segundo, err := auditoria.Encadenar(7, e.Eventos[0].Datos, e.Final)
	if err != nil {
		t.Fatal(err)
	}
	parcial, err := auditoria.NuevaExportacion([]auditoria.Evento{segundo}, e.Final, segundo.Punto())
	if err != nil {
		t.Fatal(err)
	}
	b, err := parcial.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	comprobar(t, []string{ruta, "7", segundo.Huella}, 1, "auditoria.ancla_externa_no_coincide")
	comprobar(t, []string{ruta, "1", e.Final.Huella, "7", segundo.Huella}, 0, "auditoria.sello_externo_coincide")
	comprobar(t, []string{ruta, "1", strings.Repeat("a", 64), "7", segundo.Huella}, 1, "auditoria.ancla_externa_no_coincide")
	comprobar(t, []string{ruta, "no_es_id", e.Final.Huella, "7", segundo.Huella}, 2, "auditoria.ancla_invalida")
	comprobar(t, []string{ruta, "8", e.Final.Huella, "7", segundo.Huella}, 2, "auditoria.ancla_invalida")
}

func TestComandoEntradasInvalidas(t *testing.T) {
	ruta, _ := fichero(t)
	for _, args := range [][]string{nil, {}, {ruta, "1"}, {ruta, "1", strings.Repeat("a", 64), "extra"}} {
		comprobar(t, args, 2, "auditoria.comprobador_uso")
	}
	for _, args := range [][]string{{ruta, "no_es_id", strings.Repeat("a", 64)}, {ruta, "1", "huella"}, {ruta, "0", strings.Repeat("a", 64)}} {
		comprobar(t, args, 2, "auditoria.sello_invalido")
	}
	comprobar(t, []string{filepath.Join(t.TempDir(), "ausente")}, 2, "auditoria.fichero_invalido")
	comprobar(t, []string{t.TempDir()}, 2, "auditoria.fichero_invalido")
	for _, contenido := range []string{"null", "{", `{"nombre":"Inés Moreno"}`} {
		if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		comprobar(t, []string{ruta}, 1, "auditoria.formato_invalido")
	}
	if err := os.Truncate(ruta, auditoria.MaxBytesExportacion+1); err != nil {
		t.Fatal(err)
	}
	comprobar(t, []string{ruta}, 1, "auditoria.exportacion_demasiado_grande")
	fifo := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	comprobar(t, []string{fifo}, 2, "auditoria.fichero_invalido")
}

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("destino_cerrado") }

func TestSalidaFallidaNoDevuelveExito(t *testing.T) {
	ruta, _ := fichero(t)
	if ejecutar([]string{ruta}, escritorFallido{}) != 2 {
		t.Fatal("éxito sin salida")
	}
}

func TestProcesoComprobador(t *testing.T) {
	// Reejecuta el binario de pruebas como proceso para comprobar salida y
	// código reales de main; el proceso hereda el sandbox de estas pruebas.
	if os.Getenv("ETF_PRUEBA_COMPROBADOR") == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				main()
				return
			}
		}
		os.Exit(3)
	}
	ruta, e := fichero(t)
	for _, caso := range []struct {
		args   []string
		codigo int
		clave  string
	}{
		{[]string{ruta, "1", e.Final.Huella}, 0, "auditoria.sello_externo_coincide"},
		{[]string{ruta, "1", strings.Repeat("b", 64)}, 1, "auditoria.sello_externo_no_coincide"},
		{nil, 2, "auditoria.comprobador_uso"},
	} {
		args := append([]string{"-test.run=^TestProcesoComprobador$", "--"}, caso.args...)
		cmd := exec.Command(os.Args[0], args...)
		cmd.Env = append(os.Environ(), "ETF_PRUEBA_COMPROBADOR=1")
		b, err := cmd.CombinedOutput()
		codigo := 0
		if err != nil {
			var fallo *exec.ExitError
			if !errors.As(err, &fallo) {
				t.Fatal(err)
			}
			codigo = fallo.ExitCode()
		}
		if codigo != caso.codigo {
			t.Fatalf("proceso: %d, salida %s", codigo, b)
		}
		var r resultado
		if err := json.Unmarshal(b, &r); err != nil || r.Clave != caso.clave {
			t.Fatalf("salida: %s, %v", b, err)
		}
	}
}
