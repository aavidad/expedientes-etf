package documento_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	manual "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/registro_manual"
	paquetesFicheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
)

func (m *memoria) LeerRegistro(_ context.Context, id string) (d.Emision, d.RegistroSalida, error) {
	for _, e := range m.emisiones {
		if e.ID == id {
			return e, m.registros[id], nil
		}
	}
	return d.Emision{}, d.RegistroSalida{}, nil
}
func (m *memoria) GuardarRegistro(_ context.Context, r d.RegistroSalida, version int) error {
	if m.registros[r.EmisionID].Version != version {
		return d.ErrNoDisponible
	}
	m.registros[r.EmisionID] = r
	return nil
}

type firmantesVariables struct {
	nombre  string
	version int
}

func (f *firmantesVariables) Vigente(_ context.Context, rol string, _ time.Time) (d.Firmante, error) {
	if rol == "jefe_spm" {
		return d.Firmante{ID: "33333333-3333-4333-8333-333333333334", Version: f.version, Nombre: "Clara Serrano Rivas"}, nil
	}
	return d.Firmante{ID: "33333333-3333-4333-8333-333333333333", Version: f.version, Nombre: f.nombre}, nil
}

func textoWord(t *testing.T, b []byte) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			b, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
	}
	t.Fatal("parte ausente")
	return ""
}

func prepararEntrada(t *testing.T, tipo string, n int, idioma string) (*app.Servicio, *memoria, *fuente, d.Actor, d.Solicitud) {
	t.Helper()
	_, deps, m, f, a, p := preparar(t)
	version := "1.0.0"
	if tipo == d.ActaAceptacion {
		version = "1.1.0"
	}
	ruta := "plantillas/" + tipo + "/v" + version + ".docx"
	b, err := os.ReadFile("../../../idiomas/" + idioma + "/" + ruta)
	if err != nil {
		t.Fatal(err)
	}
	m.plantilla = d.Plantilla{ID: "55555555-5555-4555-8555-555555555555", Tipo: tipo, Idioma: idioma, Version: version, Ruta: ruta, Huella: d.Huella(b), RolFirmante: "aceptacion"}
	p.Tipo = tipo
	p.Idioma = idioma
	p.Clave = fmt.Sprintf("entrada-%d-%s", n, idioma)
	textos, err := paquetesFicheros.NuevosPaquetes("../../../idiomas", "../../../temas").Textos(t.Context(), idioma)
	if err != nil {
		t.Fatal(err)
	}
	f.datos.Aceptacion.FechaActa = "13/10/2026"
	f.datos.Aceptacion.Direccion = "Calle del Laurel 18, Dúrcal"
	f.datos.Aceptacion.DestinatarioSSCC = "Valle Lecrín-Ogíjares"
	f.datos.Aceptacion.MarcaSeleccion = textos["documento.marca_seleccion"]
	f.datos.Aceptacion.RotuloMenor = textos["documento.rotulo_menor"]
	f.datos.Aceptacion.Selecciones = map[string]string{}
	for _, campo := range regexp.MustCompile(`\{\{((?:equipo|centro|profesional|origen)_[A-Za-z0-9_]+)\}\}`).FindAllStringSubmatch(textoWord(t, b), -1) {
		f.datos.Aceptacion.Selecciones[campo[1]] = ""
	}
	for _, k := range []string{"equipo_04", "centro_valle_lecrin_ogijares", "profesional_ts", "origen_anexo3"} {
		f.datos.Aceptacion.Selecciones[k] = textos["documento.marca_seleccion"]
	}
	f.datos.Aceptacion.Menores = nil
	nombres := []string{"Nerea", "Álvaro", "Celia", "Mateo", "Irene", "Hugo", "Paula", "Bruno", "Aitana", "Leo", "Claudia", "Mario", "Vera", "Lucas", "Eva"}
	for i := 0; i < n; i++ {
		f.datos.Aceptacion.Menores = append(f.datos.Aceptacion.Menores, d.Menor{Persona: d.Persona{Nombre: nombres[i] + " Ortega Salas", Nacimiento: fmt.Sprintf("%02d/05/2016", i+1)}, Sexo: textos["documento.sexo.m"], CodigoSexo: "M"})
	}
	f.datos.Aceptacion.Adultos = []d.Adulto{{Persona: d.Persona{Nombre: "Rosa Salas Ávila"}, Parentesco: "Abuela"}, {Persona: d.Persona{Nombre: "Daniel Ortega Gil"}, Parentesco: "Tío"}, {Persona: d.Persona{Nombre: "Marta Salas Ríos"}, Parentesco: "Tía"}, {Persona: d.Persona{Nombre: "Pablo Ortega Vega"}, Parentesco: "Tío"}, {Persona: d.Persona{Nombre: "Elena Gil Ruiz"}, Parentesco: "Abuela"}}
	deps.Firmantes = &firmantesVariables{nombre: "Inés Moreno Castillo", version: 1}
	deps.RepositorioRegistro = m
	deps.RegistroSalida = manual.Registro{}
	return app.Nuevo(deps), m, f, a, p
}

func TestEntradaTresEmisionesUnoSeisQuinceMenores(t *testing.T) {
	for _, tipo := range []string{d.ActaAceptacion, d.OficioSSCC, d.OficioSPM} {
		for _, n := range []int{1, 6, 15} {
			for _, idioma := range []string{"es", "en"} {
				t.Run(fmt.Sprintf("%s/%d/%s", tipo, n, idioma), func(t *testing.T) {
					s, m, f, a, p := prepararEntrada(t, tipo, n, idioma)
					r, err := s.Emitir(t.Context(), a, p)
					if err != nil {
						t.Fatal(err)
					}
					xml := textoWord(t, r.Contenido)
					if strings.Contains(xml, "{{") || strings.Contains(xml, "FIRMANTE_PENDIENTE") || strings.Contains(xml, "DESTINATARIO_PENDIENTE") {
						t.Fatal("marcadores o nombres fijos")
					}
					for _, menor := range f.datos.Aceptacion.Menores {
						if strings.Count(xml, menor.Nombre) != 1 {
							t.Fatalf("menor omitido o duplicado: %s", menor.Nombre)
						}
					}
					if !strings.Contains(xml, "Inés Moreno Castillo") {
						t.Fatal("firmante ausente")
					}
					if tipo == d.ActaAceptacion {
						for _, adulto := range f.datos.Aceptacion.Adultos {
							if !strings.Contains(xml, adulto.Nombre) {
								t.Fatal("adulto ausente")
							}
						}
						if !strings.Contains(xml, "Anexo  2L-A") || !strings.Contains(xml, "Anexo 2J") {
							t.Fatal("referencia alterada")
						}
					}
					if tipo == d.OficioSPM && !strings.Contains(xml, "Clara Serrano Rivas") {
						t.Fatal("jefe SPM ausente")
					}
					repetida, err := s.Emitir(t.Context(), a, p)
					if err != nil || !repetida.Repetida || !bytes.Equal(r.Contenido, repetida.Contenido) || len(m.emisiones) != 1 {
						t.Fatal("repetición cambió contenido", err)
					}
					if idioma == "es" {
						comprobarLibreOffice(t, r.Contenido, strings.ReplaceAll(tipo, "/", "_")+fmt.Sprint(n), tipo == d.ActaAceptacion && n == 15)
					}
				})
			}
		}
	}
}

func TestNoAceptacionEsperaEmisionesUnoSeisQuinceMenores(t *testing.T) {
	for _, tipo := range []string{d.ActaNoAceptacion, d.OficioListaEspera} {
		for _, n := range []int{1, 6, 15} {
			for _, idioma := range []string{"es", "en"} {
				t.Run(fmt.Sprintf("%s/%d/%s", tipo, n, idioma), func(t *testing.T) {
					s, m, f, a, p := prepararEntrada(t, tipo, n, idioma)
					for _, perfil := range []string{"equipo", "administracion", "consulta"} {
						actor := a
						actor.Perfiles = []string{perfil}
						if _, err := s.Emitir(t.Context(), actor, p); !errors.Is(err, d.ErrAcceso) {
							t.Fatal("perfil admitido", perfil, err)
						}
					}
					f.datos.EquipoID = ""
					f.datos.Aceptacion.Numero = ""
					f.datos.Aceptacion.Decision = &d.DecisionDocumento{Fecha: "13/10/2026", FechaDerivacion: "06/10/2026", Motivo: "La familia recibe atención especializada de otro recurso.", Subprograma: "RIESGO_SOCIAL"}
					r, err := s.Emitir(t.Context(), a, p)
					if err != nil {
						t.Fatal(err)
					}
					xml := textoWord(t, r.Contenido)
					if strings.Contains(xml, "{{") {
						t.Fatal("marcadores sin rellenar")
					}
					for _, menor := range f.datos.Aceptacion.Menores {
						if strings.Count(xml, menor.Nombre) != 1 {
							t.Fatal("menor omitido o duplicado", menor.Nombre)
						}
					}
					if tipo == d.ActaNoAceptacion && !strings.Contains(xml, f.datos.Aceptacion.Decision.Motivo) {
						t.Fatal("motivo ausente")
					}
					if tipo == d.OficioListaEspera && (!strings.Contains(xml, "06/10/2026") || m.registros[r.Emision.ID].Estado != "pendiente") {
						t.Fatal("fecha o registro ausente")
					}
					repetida, err := s.Emitir(t.Context(), a, p)
					if err != nil || !repetida.Repetida || !bytes.Equal(r.Contenido, repetida.Contenido) {
						t.Fatal("repetición", err)
					}
					if idioma == "es" {
						comprobarLibreOffice(t, r.Contenido, strings.ReplaceAll(tipo, "/", "_")+fmt.Sprint(n), n == 15)
					}
				})
			}
		}
	}
}

func comprobarLibreOffice(t *testing.T, b []byte, nombre string, evidencia bool) {
	t.Helper()
	if os.Getenv("ETF_DOCUMENTO_PROBAR_LIBREOFFICE") != "1" {
		return
	}
	raiz := t.TempDir()
	doc := filepath.Join(raiz, nombre+".docx")
	if err := os.WriteFile(doc, b, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancelar()
	cmd := exec.CommandContext(ctx, "env", "-u", "LD_LIBRARY_PATH", "libreoffice", "-env:UserInstallation=file://"+filepath.Join(raiz, "perfil"), "--headless", "--convert-to", "pdf", "--outdir", raiz, doc)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("LibreOffice: %v %s", err, salida)
	}
	pdf, err := os.ReadFile(filepath.Join(raiz, nombre+".pdf"))
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("PDF inválido", err)
	}
	t.Logf("LibreOffice abrió %s; PDF %d bytes", nombre, len(pdf))
	if evidencia && os.Getenv("ETF_DOCUMENTO_EVIDENCIA") != "" {
		destino, err := filepath.Abs(os.Getenv("ETF_DOCUMENTO_EVIDENCIA"))
		if err != nil {
			t.Fatal(err)
		}
		repo, _ := filepath.Abs("../../..")
		if destino == repo || strings.HasPrefix(destino, repo+string(os.PathSeparator)) {
			t.Fatal("evidencia dentro del repositorio")
		}
		if err := os.MkdirAll(destino, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destino, nombre+".pdf"), pdf, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destino, nombre+".docx"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRegistroSalidaVersionRepeticionPermisoYAuditoria(t *testing.T) {
	s, m, _, a, p := prepararEntrada(t, d.OficioSSCC, 6, "es")
	e, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	peticion := d.SolicitudRegistroSalida{EmisionID: e.Emision.ID, Clave: "registro-1", Version: 1, Numero: "2026/1842", Fecha: "2026-10-13"}
	antes := len(m.eventos)
	r, err := s.AnotarRegistroSalida(t.Context(), a, peticion)
	if err != nil || r.Version != 2 || r.Estado != "anotado" || r.Numero != peticion.Numero || len(m.eventos) != antes+1 {
		t.Fatal(r, err)
	}
	repetida, err := s.AnotarRegistroSalida(t.Context(), a, peticion)
	if err != nil || r != repetida || m.registros[r.EmisionID] != r {
		t.Fatal("anotación duplicada", err)
	}
	peticion.Numero = "2026/1843"
	if _, err := s.AnotarRegistroSalida(t.Context(), a, peticion); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal("clave reutilizada", err)
	}
	peticion.Clave = "registro-2"
	var conflicto *d.ConflictoRegistro
	if _, err := s.AnotarRegistroSalida(t.Context(), a, peticion); !errors.As(err, &conflicto) || conflicto.VersionActual != 2 {
		t.Fatal("sin conflicto", err)
	}
	peticion.Version = 2
	m.fallaAuditoria = true
	if _, err := s.AnotarRegistroSalida(t.Context(), a, peticion); !errors.Is(err, d.ErrNoDisponible) || m.registros[r.EmisionID] != r {
		t.Fatal("escritura sin auditoría", err)
	}
	m.fallaAuditoria = false
	for _, perfil := range []string{"equipo", "administracion", "consulta"} {
		actor := a
		actor.Perfiles = []string{perfil}
		if _, err := s.AnotarRegistroSalida(t.Context(), actor, peticion); !errors.Is(err, d.ErrAcceso) {
			t.Fatal("perfil admitido", perfil, err)
		}
	}
	s, _, _, a, p = prepararEntrada(t, d.ActaAceptacion, 1, "es")
	e, err = s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	peticion.EmisionID = e.Emision.ID
	if _, err = s.AnotarRegistroSalida(t.Context(), a, peticion); !errors.Is(err, d.ErrRegistroOficio) {
		t.Fatal("registro de acta", err)
	}
}

func TestFirmanteCambiadoGeneraDocumentoNuevo(t *testing.T) {
	_, deps, m, f, a, p := preparar(t)
	vars := &firmantesVariables{nombre: "Inés Moreno Castillo", version: 1}
	deps.Firmantes = vars
	s := app.Nuevo(deps)
	inicial, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	vars.nombre = "Elena Núñez Romero"
	vars.version = 2
	if _, err := s.Emitir(t.Context(), a, p); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal("reproducción con firmante nuevo", err)
	}
	p.Clave = "firmante-nuevo"
	nueva, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textoWord(t, nueva.Contenido), vars.nombre) || inicial.Emision.HuellaDatos == nueva.Emision.HuellaDatos || len(m.emisiones) != 2 || nueva.Emision.Aviso != "documento.regenerado" || f.datos.Versiones["expediente"] != "1" {
		t.Fatal("cambio de catálogo no reflejado")
	}
}

func (m *memoria) PeticionRegistro(_ context.Context, a d.Actor, p d.SolicitudRegistroSalida, h string) (d.RegistroSalida, bool, error) {
	anterior, ok := m.registroPeticiones[a.UsuarioID+":"+p.Clave]
	if !ok {
		return d.RegistroSalida{}, false, nil
	}
	if anterior.huella != h {
		return d.RegistroSalida{}, false, d.ErrIdempotencia
	}
	return anterior.registro, true, nil
}
func (m *memoria) RecordarRegistro(_ context.Context, a d.Actor, p d.SolicitudRegistroSalida, h string, r d.RegistroSalida) error {
	if m.registroPeticiones == nil {
		m.registroPeticiones = map[string]peticionRegistro{}
	}
	m.registroPeticiones[a.UsuarioID+":"+p.Clave] = peticionRegistro{h, r}
	return nil
}

type peticionRegistro struct {
	huella   string
	registro d.RegistroSalida
}
