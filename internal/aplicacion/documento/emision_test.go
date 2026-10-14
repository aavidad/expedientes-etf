package documento_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

const usuario = "11111111-1111-4111-8111-111111111111"
const expediente = "22222222-2222-4222-8222-222222222222"

type fuente struct {
	datos    d.Datos
	llamadas int
}

func (f *fuente) Obtener(context.Context, d.Actor, d.Solicitud) (d.Datos, error) {
	f.llamadas++
	return f.datos, nil
}

type firmantes struct{}

func (firmantes) Vigente(context.Context, string, time.Time) (d.Firmante, error) {
	return d.Firmante{ID: "33333333-3333-4333-8333-333333333333", Version: 1, Nombre: "Inés Moreno Castillo"}, nil
}

type reloj struct{}

func (reloj) Ahora() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }

type ids struct{ n int }

func (i *ids) Nuevo() (string, error) {
	i.n++
	return fmt.Sprintf("44444444-4444-4444-8444-%012d", i.n), nil
}

type referencias struct{}

func (referencias) Referencias(context.Context, string) ([]p.Referencia, error) { return nil, nil }

type peticion struct {
	e      d.Emision
	huella string
}
type memoria struct {
	mu                          sync.Mutex
	plantilla                   d.Plantilla
	peticiones                  map[string]peticion
	emisiones                   []d.Emision
	eventos                     []auditoria.Datos
	registros                   map[string]d.RegistroSalida
	registroPeticiones          map[string]peticionRegistro
	fallaAuditoria, fallaCommit bool
}

func (*memoria) Bloquear(context.Context, d.Actor, d.Solicitud) error { return nil }
func (m *memoria) Peticion(_ context.Context, a d.Actor, c string) (d.Emision, string, error) {
	p := m.peticiones[a.UsuarioID+c]
	return p.e, p.huella, nil
}
func (m *memoria) Activa(context.Context, string, string) (d.Plantilla, error) {
	if m.plantilla.ID == "" {
		return d.Plantilla{}, d.ErrPlantilla
	}
	return m.plantilla, nil
}
func (m *memoria) Ultima(context.Context, d.Solicitud) (d.Emision, error) {
	if len(m.emisiones) == 0 {
		return d.Emision{}, nil
	}
	return m.emisiones[len(m.emisiones)-1], nil
}
func (m *memoria) Guardar(_ context.Context, a d.Actor, p d.Solicitud, h string, e d.Emision) error {
	if d.EsOficio(e.Tipo) {
		if m.registros == nil {
			m.registros = map[string]d.RegistroSalida{}
		}
		m.registros[e.ID] = d.RegistroSalida{EmisionID: e.ID, Adaptador: "manual", Estado: "pendiente", Version: 1}
	}
	m.peticiones[a.UsuarioID+p.Clave] = peticion{e, h}
	m.emisiones = append(m.emisiones, e)
	return nil
}
func (m *memoria) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	previas := len(m.emisiones)
	registroPeticiones := map[string]peticionRegistro{}
	for k, v := range m.registroPeticiones {
		registroPeticiones[k] = v
	}
	registros := map[string]d.RegistroSalida{}
	for k, v := range m.registros {
		registros[k] = v
	}
	eventos := len(m.eventos)
	peticiones := map[string]peticion{}
	for k, v := range m.peticiones {
		peticiones[k] = v
	}
	err := fn(ctx)
	if err == nil && m.fallaCommit {
		err = d.ErrNoDisponible
	}
	if err != nil {
		m.emisiones = m.emisiones[:previas]
		m.eventos = m.eventos[:eventos]
		m.peticiones = peticiones
		m.registros = registros
		m.registroPeticiones = registroPeticiones
	}
	return err
}
func (m *memoria) RegistrarEnTransaccion(_ context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	if m.fallaAuditoria {
		return auditoria.Evento{}, d.ErrNoDisponible
	}
	m.eventos = append(m.eventos, e)
	return auditoria.Evento{}, nil
}
func (m *memoria) Confirmar(_ context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventos = append(m.eventos, e)
	return auditoria.Evento{}, nil
}

func preparar(t *testing.T) (*app.Servicio, app.Dependencias, *memoria, *fuente, d.Actor, d.Solicitud) {
	t.Helper()
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	m := &memoria{plantilla: d.Plantilla{ID: "55555555-5555-4555-8555-555555555555", Tipo: d.ActaAceptacion, Idioma: "es", Version: "1.0.0", Ruta: "plantillas/entrada/aceptacion/v1.0.0.docx", Huella: d.Huella(b), RolFirmante: "aceptacion"}, peticiones: map[string]peticion{}}
	f := &fuente{datos: d.Datos{ExpedienteID: expediente, FechaDocumento: reloj{}.Ahora(), Versiones: map[string]string{"expediente": "1"}, Aceptacion: &d.Aceptacion{Numero: "04-01-26", Equipo: "ETF Valle Lecrín", Centro: "Valle Lecrín", Municipio: "Dúrcal", ServicioDerivante: "Servicios Sociales Comunitarios", Familia: "Ortega Salas", Descripcion: `Lucía < & " ñ 😀`, Madre: d.Persona{Nombre: "Lucía Ortega Salas", Nacimiento: "14/02/1987"}, Menores: []d.Menor{{Persona: d.Persona{Nombre: "Nerea Ortega Salas", Nacimiento: "07/05/2016"}, Sexo: "Femenino"}}, Profesional: "Pablo Ruiz Vílchez"}}}
	archivos := ficheros.Plantillas{Raiz: "../../../idiomas", Paquetes: referencias{}}
	deps := app.Dependencias{Repositorio: m, Datos: f, Firmantes: firmantes{}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: m, Registro: m, RegistroTransaccional: m, Reloj: reloj{}, Identificadores: &ids{}}
	return app.Nuevo(deps), deps, m, f, d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, d.Solicitud{Tipo: d.ActaAceptacion, ExpedienteID: expediente, Idioma: "es", Clave: "aceptacion-1"}
}
func TestEmisionIdempotenciaHuellasYRegeneracion(t *testing.T) {
	s, _, m, f, a, p := preparar(t)
	r, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	repetida, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	if !repetida.Repetida || !bytes.Equal(r.Contenido, repetida.Contenido) || !reflect.DeepEqual(r.Emision, repetida.Emision) || len(m.emisiones) != 1 || d.Huella(r.Contenido) != r.Emision.HuellaResultado || len(r.Emision.HuellaDatos) != 64 {
		t.Fatal("repetición o huellas incorrectas")
	}
	b, _ := json.Marshal(m.emisiones)
	if bytes.Contains(b, []byte("Ortega")) || bytes.Contains(b, []byte("Inés")) || bytes.Contains(b, []byte("PK")) {
		t.Fatal("se han persistido datos o fichero")
	}
	if !reflect.DeepEqual(f.datos.Versiones, map[string]string{"expediente": "1"}) {
		t.Fatal("versión de expediente modificada")
	}
	f.datos.Aceptacion.Descripcion = "Valoración actualizada"
	if r, err := s.Emitir(t.Context(), a, p); !errors.Is(err, d.ErrIdempotencia) || len(r.Contenido) != 0 || len(m.emisiones) != 1 {
		t.Fatal("clave reutilizada", err)
	}
	p.Clave = "aceptacion-2"
	nueva, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	if nueva.Emision.ID == r.Emision.ID || nueva.Emision.Aviso != "documento.regenerado" || nueva.Emision.HuellaDatos == r.Emision.HuellaDatos || nueva.Emision.HuellaResultado == r.Emision.HuellaResultado || len(m.emisiones) != 2 {
		t.Fatal("regeneración sin emisión nueva y aviso")
	}
	if f.datos.FechaDocumento != (reloj{}).Ahora() {
		t.Fatal("fecha cambiada")
	}
}
func TestEmisionConcurrenteUnaSolaEmision(t *testing.T) {
	s, _, m, _, a, p := preparar(t)
	var wg sync.WaitGroup
	errores := make(chan error, 8)
	resultados := make(chan d.Resultado, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := s.Emitir(context.Background(), a, p)
			errores <- err
			resultados <- r
		}()
	}
	wg.Wait()
	close(errores)
	close(resultados)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
	id := ""
	for r := range resultados {
		if id == "" {
			id = r.Emision.ID
		}
		if r.Emision.ID != id {
			t.Fatal("recibos distintos")
		}
	}
	if len(m.emisiones) != 1 {
		t.Fatal(len(m.emisiones))
	}
}
func TestEmisionNoEntregaSinAuditoriaOCommit(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprint(commit), func(t *testing.T) {
			s, _, m, _, a, p := preparar(t)
			m.fallaCommit = commit
			m.fallaAuditoria = !commit
			r, err := s.Emitir(t.Context(), a, p)
			if err == nil || len(r.Contenido) != 0 || len(m.emisiones) != 0 || len(m.peticiones) != 0 {
				t.Fatal("entrega o escritura sin confirmación")
			}
		})
	}
}
func TestEmisionPlantillaInexistenteYPermisos(t *testing.T) {
	s, _, m, f, a, p := preparar(t)
	m.plantilla.ID = ""
	if _, err := s.Emitir(t.Context(), a, p); !errors.Is(err, d.ErrPlantilla) {
		t.Fatal(err)
	}
	for _, perfil := range []string{"administracion", "consulta", "sistema", "equipo"} {
		a.Perfiles = []string{perfil}
		llamadas := f.llamadas
		if _, err := s.Emitir(t.Context(), a, p); !errors.Is(err, d.ErrAcceso) || f.llamadas != llamadas {
			t.Fatal(perfil, err)
		}
	}
}

type alterado struct{ word.Motor }

func (m alterado) Generar(ctx context.Context, b []byte, r d.Relleno) ([]byte, error) {
	contenido, err := m.Motor.Generar(ctx, b, r)
	return append(contenido, 1), err
}
func TestEmisionConflictoReproduccionConRecibo(t *testing.T) {
	s, deps, _, _, a, p := preparar(t)
	r, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	deps.Motor = alterado{}
	repetida, err := app.Nuevo(deps).Emitir(t.Context(), a, p)
	var conflicto *d.ConflictoReproduccion
	if !errors.As(err, &conflicto) || conflicto.Emision.ID != r.Emision.ID || len(repetida.Contenido) != 0 {
		t.Fatal(err)
	}
}
func archivos(t *testing.T, raiz string) map[string]int64 {
	t.Helper()
	m := map[string]int64{}
	err := filepath.Walk(raiz, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !i.IsDir() {
			m[p] = i.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestEmisionNoEscribeFicheros(t *testing.T) {
	s, deps, _, _, a, p := preparar(t)
	raiz := t.TempDir()
	carpeta := filepath.Join(raiz, "es", "plantillas", "entrada", "aceptacion")
	if err := os.MkdirAll(carpeta, 0700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(carpeta, "v1.0.0.docx"), b, 0400); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(carpeta, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(carpeta, 0700) })
	antes := archivos(t, raiz)
	deps.Ficheros = ficheros.Plantillas{Raiz: raiz, Paquetes: referencias{}}
	s = app.Nuevo(deps)
	if _, err := s.Emitir(t.Context(), a, p); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(antes, archivos(t, raiz)) {
		t.Fatal("se escribió un fichero")
	}
}
func TestEmisionLibreOfficeLocal(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_PROBAR_LIBREOFFICE") != "1" {
		t.Skip("ETF_DOCUMENTO_PROBAR_LIBREOFFICE=1")
	}
	s, _, _, _, a, p := preparar(t)
	r, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(r.Contenido), int64(len(r.Contenido)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			archivo, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			xml, err := io.ReadAll(archivo)
			_ = archivo.Close()
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(xml, []byte("{{")) || !bytes.Contains(xml, []byte("Inés Moreno Castillo")) {
				t.Fatal("marcadores o firmante incorrectos")
			}
		}
	}
	raiz := t.TempDir()
	doc := filepath.Join(raiz, "aceptacion.docx")
	if err = os.WriteFile(doc, r.Contenido, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancelar()
	cmd := exec.CommandContext(ctx, "env", "-u", "LD_LIBRARY_PATH", "soffice", "-env:UserInstallation=file://"+filepath.Join(raiz, "perfil"), "--headless", "--convert-to", "pdf", "--outdir", raiz, doc)
	salida, err := cmd.CombinedOutput()
	t.Log(strings.TrimSpace(string(salida)))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(raiz, "aceptacion.pdf"))
	if err != nil || info.Size() == 0 {
		t.Fatal("LibreOffice no abrió el documento", err)
	}
}
