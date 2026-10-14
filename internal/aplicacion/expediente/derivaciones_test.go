package expediente

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/memoria"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--lector-anexo-pdf" {
		if err := lectura.EjecutarPDF(os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type entorno struct {
	datos          map[string]dominio.Derivacion
	eventos        []auditoria.Datos
	falloAuditoria bool
	antivirus      plataforma.EstadoAntivirus
	falloAV        error
	ids            int
	llamadasAV     int
	llamadasLector int
	extraida       *dominio.Extraccion
	municipios     []catalogo.Municipio
}

func (e *entorno) Guardar(_ context.Context, d dominio.Derivacion) error {
	e.datos[d.ID] = d
	return nil
}
func (e *entorno) Obtener(_ context.Context, id string) (dominio.Derivacion, error) {
	d, ok := e.datos[id]
	if !ok {
		return d, dominio.ErrNoEncontrada
	}
	return d, nil
}
func (e *entorno) Ejecutar(ctx context.Context, _ plataforma.Identidad, f func(context.Context) error) error {
	anterior := make(map[string]dominio.Derivacion)
	for k, v := range e.datos {
		anterior[k] = v
	}
	n := len(e.eventos)
	if err := f(ctx); err != nil {
		e.datos = anterior
		e.eventos = e.eventos[:n]
		return err
	}
	return nil
}
func (e *entorno) Confirmar(_ context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	if err := d.Validar(); err != nil {
		return auditoria.Evento{}, err
	}
	if e.falloAuditoria {
		return auditoria.Evento{}, dominio.ErrNoDisponible
	}
	e.eventos = append(e.eventos, d)
	return auditoria.Evento{Datos: d}, nil
}
func (e *entorno) RegistrarEnTransaccion(ctx context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	return e.Confirmar(ctx, d)
}
func (e *entorno) Revisar(ctx context.Context, _ []byte) (plataforma.EstadoAntivirus, error) {
	e.llamadasAV++
	return e.antivirus, e.falloAV
}
func (e *entorno) Leer(ctx context.Context, f string, b []byte) (dominio.Extraccion, error) {
	e.llamadasLector++
	if e.extraida != nil {
		return *e.extraida, nil
	}
	return lectura.NuevoLector().Leer(ctx, f, b)
}
func (*entorno) Ahora() time.Time { return time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC) }
func (e *entorno) Nuevo() (string, error) {
	e.ids++
	return fmt.Sprintf("ab60479d-b778-407f-a83e-%012d", e.ids), nil
}
func (*entorno) Vigente(context.Context, string, time.Time) (catalogo.Version, error) {
	return catalogo.Version{}, nil
}
func (*entorno) Versiones(context.Context, string) ([]catalogo.Version, error) { return nil, nil }
func (e *entorno) Datos(context.Context, catalogo.Version) (catalogo.Datos, error) {
	if e.municipios != nil {
		return catalogo.Datos{Municipios: e.municipios}, nil
	}
	return catalogo.Datos{Municipios: []catalogo.Municipio{{INE: "18057", ClaveTexto: "municipio.18057", AmbitoDiputacion: true}}}, nil
}

func TestPropuestaConVariantesDelPaqueteReal(t *testing.T) {
	for _, c := range []struct{ direccion, ine, clave string }{
		{"Huétor Santillán", "18099", "municipio.18099"},
		{"Alhama", "18013", "municipio.18013"},
	} {
		t.Run(c.ine, func(t *testing.T) {
			s, e, a, b := preparar(t)
			e.municipios = []catalogo.Municipio{{INE: c.ine, ClaveTexto: c.clave, AmbitoDiputacion: true}}
			e.extraida = &dominio.Extraccion{Campos: []dominio.Campo{{Codigo: "familia.direccion", Valor: c.direccion}}}
			s.d.NombresMunicipios = nombres.NuevosMunicipios("../../../idiomas", "es")
			d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
			if err != nil || len(d.Datos.PropuestaMunicipio.Candidatos) != 1 || d.Datos.PropuestaMunicipio.Candidatos[0].Codigo != c.ine || d.MunicipioID != nil {
				t.Fatal("variante no propuesta", err)
			}
		})
	}
}
func (*entorno) Nombres(context.Context) (map[string]string, error) {
	return map[string]string{"municipio.18057": "Cúllar Vega"}, nil
}

func preparar(t *testing.T) (*Servicio, *entorno, entrada.Actor, []byte) {
	t.Helper()
	e := &entorno{datos: map[string]dominio.Derivacion{}, antivirus: plataforma.AntivirusLimpio}
	s := Nuevo(Dependencias{Borradores: memoria.NuevosBorradores(time.Minute, 32), Repositorio: e, Transacciones: e, Registro: e, RegistroTransaccional: e, Validador: lectura.Validador{}, Antivirus: e, Lector: e, Catalogo: e, NombresMunicipios: e, Reloj: e, Identificadores: e})
	b, err := os.ReadFile("../../adaptadores/anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	return s, e, entrada.Actor{UsuarioID: "ab60479d-b778-407f-a83e-012345678901", Perfiles: []string{"coordinacion"}}, b
}

func TestSubidaDOCXRealYLecturaAuditadas(t *testing.T) {
	s, e, a, b := preparar(t)
	d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "" || d.Estado != "" || len(e.datos) != 0 || d.Numero != nil || d.EquipoID != nil || d.MunicipioID != nil || d.Version != 0 || len(d.Lectura.HuellaSHA256) != 64 || d.Lectura.Antivirus != "limpio" {
		t.Fatal(d)
	}
	if e.llamadasAV != 1 || e.llamadasLector != 1 || len(e.eventos) != 1 || e.eventos[0].Accion != "expediente.subir_anexo" {
		t.Fatal(e)
	}
	leida, err := s.d.Borradores.Obtener(context.Background(), a.UsuarioID, d.Lectura.ID)
	if err != nil || leida.ID != d.ID || len(e.eventos) != 1 {
		t.Fatal(leida, err)
	}
	for _, c := range d.Datos.Extraccion.Campos {
		if strings.HasPrefix(c.Codigo, "valorame.") {
			t.Fatal("escala guardada en expediente")
		}
	}
}

func TestClamAVCaidoAceptadoConAvisoAuditado(t *testing.T) {
	s, e, a, b := preparar(t)
	e.antivirus, e.falloAV = plataforma.AntivirusNoDisponible, errors.New("antivirus.no_disponible")
	d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil || d.Lectura.Antivirus != "no_revisado" || d.Lectura.AntivirusDetalle != "expediente.antivirus_no_revisado" || len(e.eventos) != 2 || e.eventos[1].Accion != "expediente.antivirus_no_revisado" {
		t.Fatal(d, err, e.eventos)
	}
}

func TestInfectadoNoSeLeeNiGuarda(t *testing.T) {
	s, e, a, b := preparar(t)
	e.antivirus = plataforma.AntivirusInfectado
	_, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if !errors.Is(err, dominio.ErrInfectado) || len(e.datos) != 0 || e.llamadasLector != 0 || len(e.eventos) != 1 || e.eventos[0].Accion != "expediente.anexo_infectado" {
		t.Fatal(err, e)
	}
}

func TestFalloAuditoriaRevierteYNoEntrega(t *testing.T) {
	s, e, a, b := preparar(t)
	e.falloAuditoria = true
	d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err == nil || d.ID != "" || len(e.datos) != 0 {
		t.Fatal(d, err, e)
	}
	e.falloAuditoria = false
	d, err = s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil {
		t.Fatal(err)
	}
	e.falloAuditoria = true
	d, err = s.Consultar(context.Background(), a, d.ID)
	if err == nil || d.ID != "" {
		t.Fatal("lectura entregada sin auditoría")
	}
}

func TestAutorizacionValidacionYCancelacion(t *testing.T) {
	for _, perfil := range []string{"administracion", "consulta", "profesional_etf", ""} {
		t.Run(perfil, func(t *testing.T) {
			s, e, a, b := preparar(t)
			a.Perfiles = []string{perfil}
			if perfil == "" {
				a.Perfiles = nil
			}
			_, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
			if !errors.Is(err, dominio.ErrAcceso) || e.llamadasAV != 0 || len(e.datos) != 0 {
				t.Fatal(err, e)
			}
		})
	}
	s, e, a, b := preparar(t)
	if _, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Macro.docm", Contenido: b}); err == nil || e.llamadasAV != 0 {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: bytes.Clone(b)}); err == nil || len(e.datos) != 0 {
		t.Fatal(err)
	}
}

func TestPDFSinTextoAceptadoParaRevisionManual(t *testing.T) {
	s, e, a, _ := preparar(t)
	b, err := os.ReadFile("../../adaptadores/anexos/pdf/testdata/sin_texto.pdf")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Subir(context.Background(), a, entrada.Subida{Nombre: "Derivacion.pdf", Contenido: b})
	if err != nil || d.Lectura.Formato != "pdf" || len(d.Datos.Extraccion.Campos) != 0 || d.Datos.Extraccion.Avisos[0] != "anexo.pdf_sin_texto" || len(e.datos) != 0 {
		t.Fatal(d, err)
	}
}
