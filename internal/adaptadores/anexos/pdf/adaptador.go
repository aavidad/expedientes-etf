package pdf

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

//go:embed perfil.json
var perfilPDF []byte

// Lector ejecuta cada PDF en un proceso efímero con límites y tuberías en memoria.
// El ejecutable debe atender --lector-anexo-pdf antes de cargar configuración.
type Lector struct{ plazas chan struct{} }

var _ expediente.LectorAnexo = (*Lector)(nil)

func NuevoLector() *Lector { return &Lector{plazas: make(chan struct{}, 1)} }
func (l *Lector) Leer(ctx context.Context, formato string, b []byte) (dominio.Extraccion, error) {
	if err := ctx.Err(); err != nil {
		return dominio.Extraccion{}, err
	}
	if formato != "pdf" {
		return dominio.Extraccion{}, dominio.ErrFormato
	}
	if len(b) > MaxBytes {
		return dominio.Extraccion{}, ErrTamano
	}
	select {
	case l.plazas <- struct{}{}:
		defer func() { <-l.plazas }()
	case <-ctx.Done():
		return dominio.Extraccion{}, ctx.Err()
	}
	executable, err := os.Executable()
	if err != nil {
		return dominio.Extraccion{}, dominio.ErrNoDisponible
	}
	ctx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	cmd := exec.CommandContext(ctx, executable, "--lector-anexo-pdf")
	cmd.Env = []string{"GOMEMLIMIT=64MiB", "GOGC=20", "GOMAXPROCS=1", "GOTRACEBACK=none"}
	cmd.Stdin = bytes.NewReader(b)
	salida := &salidaAcotada{maximo: 2 << 20}
	cmd.Stdout = salida
	stderr := &diagnosticoAcotado{}
	cmd.Stderr = stderr
	defer func() { clear(stderr.Buffer.Bytes()) }()
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return dominio.Extraccion{}, ctx.Err()
		}
		return dominio.Extraccion{}, &FalloTrabajador{Causa: err, Diagnostico: stderr.resumen()}
	}
	var r resultadoPDF
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		return dominio.Extraccion{}, errors.New("anexo.pdf_corrupto")
	}
	if r.Error != "" {
		return dominio.Extraccion{}, errors.New(r.Error)
	}
	return r.Extraccion, nil
}

type salidaAcotada struct {
	bytes.Buffer
	maximo int
}

func (s *salidaAcotada) Write(b []byte) (int, error) {
	if len(b) > s.maximo-s.Len() {
		return 0, dominio.ErrTamano
	}
	return s.Buffer.Write(b)
}

type resultadoPDF struct {
	Extraccion dominio.Extraccion
	Error      string
}

// EjecutarPDF se invoca antes del arranque HTTP. No carga configuración,
// credenciales, disco ni registro del servidor. Los límites son del proceso.
func EjecutarPDF(entrada io.Reader, salida io.Writer) error {
	if err := limitarProceso(); err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(entrada, dominio.MaxBytes+1))
	if err != nil {
		return err
	}
	defer clear(b)
	d, err := Leer(b, "dslipak", Limites{Paginas: 20, Objetos: 10000, Lecturas: 100000, Glifos: 200000})
	r := resultadoPDF{}
	if err != nil {
		r.Error = err.Error()
	} else {
		var perfil Perfil
		if err := json.Unmarshal(perfilPDF, &perfil); err != nil {
			return dominio.ErrNoDisponible
		}
		e := Extraer(d, perfil)
		r.Extraccion = dominio.Extraccion{PlantillaReconocida: e.PlantillaReconocida, Avisos: e.Avisos, Version: "pdf-1"}
		patronTipo := regexp.MustCompile(perfil.Tipo)
		for _, linea := range d.Lineas {
			if m := patronTipo.FindStringSubmatch(linea.Texto()); len(m) == 2 {
				r.Extraccion.Tipo = "anexo" + m[1]
				break
			}
		}

		if e.PlantillaReconocida {
			for _, codigo := range perfil.CamposManuales[r.Extraccion.Tipo] {
				e.Campos = append(e.Campos, Campo{Codigo: codigo, Dudoso: true, Motivos: []string{"anexo.confirmar_pdf", "anexo.campo_no_leido"}})
			}
		}
		for _, c := range e.Campos {
			r.Extraccion.Campos = append(r.Extraccion.Campos, dominio.Campo{Codigo: c.Codigo, Valor: c.Valor, Pagina: c.Pagina, Dudoso: c.Dudoso, Motivos: c.Motivos})
		}
	}
	return json.NewEncoder(salida).Encode(r)
}

// FalloTrabajador conserva sólo un diagnóstico técnico permitido. Error entrega
// exclusivamente la clave pública; ni HTTP ni auditoría reciben el stderr.
type FalloTrabajador struct {
	Causa       error
	Diagnostico string
}

func (e *FalloTrabajador) Error() string { return ErrRecursos.Error() }
func (e *FalloTrabajador) Unwrap() error { return ErrRecursos }

// diagnosticoAcotado drena stderr aunque supere el cupo para no bloquear al hijo.
// Sólo publica líneas exactas del runtime: una excepción del parser podría
// contener texto del PDF o argumentos de una pila.
type diagnosticoAcotado struct{ bytes.Buffer }

func (s *diagnosticoAcotado) Write(b []byte) (int, error) {
	n := len(b)
	_, _ = s.Buffer.Write(b[:min(n, max(0, 4096-s.Len()))])
	return n, nil
}

// Las firmas son protocolo técnico del runtime, nunca textos de pantalla.
//
//etf:tecnico
func (s *diagnosticoAcotado) resumen() string {
	for _, linea := range strings.Split(s.String(), "\n") {
		switch linea {
		case "fatal error: runtime: out of memory", "fatal error: runtime: cannot allocate memory", "fatal error: out of memory", "runtime: mmap: access denied":
			return linea
		}
	}
	return ""
}

// El formato ampliado se usa exclusivamente para diagnóstico interno.
//
//etf:tecnico
func (e *FalloTrabajador) Format(s fmt.State, verbo rune) {
	if verbo == 'v' && s.Flag('+') {
		_, _ = fmt.Fprintf(s, "%s: %v; stderr=%s", e.Error(), e.Causa, e.Diagnostico)
		return
	}
	_, _ = io.WriteString(s, e.Error())
}
