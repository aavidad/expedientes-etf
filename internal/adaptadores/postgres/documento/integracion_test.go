package documento_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	entradaIdentidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

const usuario = "11111111-1111-4111-8111-111111111111"
const expediente = "22222222-2222-4222-8222-222222222222"

type fuente struct{ descripcion string }

func (f *fuente) Obtener(context.Context, d.Actor, d.Solicitud) (d.Datos, error) {
	selecciones := map[string]string{}
	for i := 1; i <= 8; i++ {
		selecciones[fmt.Sprintf("equipo_%02d", i)] = ""
	}
	for _, c := range []string{"vegas_altas", "alfacar", "vega_temple", "santa_fe", "rio_fardes", "valle_lecrin_ogijares", "sierra_nevada", "montes_orientales", "pinos_puente_moclin", "huetor_tajar_montefrio", "alhama", "costa", "alpujarra", "norte"} {
		selecciones["centro_"+c] = ""
	}
	for _, p := range []string{"ts", "es", "ps"} {
		selecciones["profesional_"+p] = ""
	}
	for _, o := range []string{"anexo3", "anexo4", "etf"} {
		selecciones["origen_"+o] = ""
	}
	return d.Datos{ExpedienteID: expediente, FechaDocumento: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), Versiones: map[string]string{"expediente": "1"}, Aceptacion: &d.Aceptacion{Selecciones: selecciones, Numero: "04-01-26", Equipo: "ETF Valle Lecrín", Centro: "Valle Lecrín", Municipio: "Dúrcal", Familia: "Ortega Salas", Descripcion: f.descripcion, Profesional: "Pablo Ruiz Vílchez", Menores: []d.Menor{{Persona: d.Persona{Nombre: "Nerea Ortega Salas", Nacimiento: "07/05/2016"}, Sexo: "Femenino"}}}}, nil
}

type reloj struct{}

func (reloj) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type ids struct{}

func (ids) Nuevo() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), err
}

type registroFallido struct{}

func (registroFallido) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, d.ErrNoDisponible
}

type sesiones struct{}

func (sesiones) AdmitirPeticion(context.Context) error { return nil }
func (sesiones) Entrar(context.Context, entradaIdentidad.Inicio) (entradaIdentidad.SesionAbierta, error) {
	return entradaIdentidad.SesionAbierta{}, identidad.ErrSesionNoValida
}
func (sesiones) Consultar(context.Context, string) (entradaIdentidad.Acceso, error) {
	return entradaIdentidad.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{identidad.Coordinacion}}}, nil
}
func (sesiones) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesiones) Salir(context.Context, string) error                    { return nil }

func TestIntegracionEmisionConcurrenteRLSAuditoriaYRegeneracion(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	r := postgres.NuevoRegistroAuditoria(b)
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	f := &fuente{descripcion: `Valoración < & ñ`}
	deps := app.Dependencias{Repositorio: repo.Repositorio{}, Datos: f, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: reloj{}, Identificadores: ids{}}
	s := app.Nuevo(deps)
	a := d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	p := d.Solicitud{Tipo: d.ActaAceptacion, ExpedienteID: expediente, Idioma: "es", Clave: "integracion-concurrente"}
	i := plataforma.Identidad{UsuarioID: usuario, Perfiles: a.Perfiles}
	var estadoAntes string
	var versionAntes int
	var fechaAntes time.Time
	if err = b.Ejecutar(ctx, i, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT estado,version,fecha_recepcion FROM expediente.expediente WHERE id=$1`, expediente).Scan(&estadoAntes, &versionAntes, &fechaAntes)
	}); err != nil {
		t.Fatal(err)
	}
	resultados := make(chan d.Resultado, 6)
	fallos := make(chan error, 6)
	var wg sync.WaitGroup
	for j := 0; j < 6; j++ {
		wg.Add(1)
		go func() { defer wg.Done(); e, err := s.Emitir(ctx, a, p); resultados <- e; fallos <- err }()
	}
	wg.Wait()
	close(resultados)
	close(fallos)
	for err := range fallos {
		if err != nil {
			t.Fatal(err)
		}
	}
	var primera d.Resultado
	for e := range resultados {
		if primera.Emision.ID == "" {
			primera = e
		}
		if e.Emision.ID != primera.Emision.ID || !bytes.Equal(e.Contenido, primera.Contenido) {
			t.Fatal("emisiones concurrentes distintas")
		}
	}
	if primera.Emision.HuellaResultado != d.Huella(primera.Contenido) {
		t.Fatal("huella incorrecta")
	}
	contar := func(identidad plataforma.Identidad, esperadas int) {
		t.Helper()
		err := b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.emision WHERE expediente_id=$1`, expediente).Scan(&n); err != nil {
				return err
			}
			if n != esperadas {
				t.Fatalf("RLS: %v vio %d", identidad.Perfiles, n)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	contar(i, 1)
	for _, perfil := range []string{"administracion", "consulta", "sistema", "equipo"} {
		contar(plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}}, 0)
	}
	contar(plataforma.Identidad{}, 0)
	if err = b.Ejecutar(ctx, i, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var estado string
		var version int
		var fecha time.Time
		err := tx.QueryRow(ctx, `SELECT estado,version,fecha_recepcion FROM expediente.expediente WHERE id=$1`, expediente).Scan(&estado, &version, &fecha)
		if estado != estadoAntes || version != versionAntes || fecha != fechaAntes {
			t.Fatal("la emisión cambió el expediente")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	deps.RegistroTransaccional = registroFallido{}
	p.Clave = "auditoria-fallida"
	if e, err := app.Nuevo(deps).Emitir(ctx, a, p); err == nil || len(e.Contenido) != 0 {
		t.Fatal("descarga sin auditoría")
	}
	contar(i, 1)
	f.descripcion = "Valoración actualizada"
	p.Clave = "integracion-concurrente"
	if _, err = s.Emitir(ctx, a, p); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal(err)
	}
	p.Clave = "integracion-regeneracion"
	nueva, err := s.Emitir(ctx, a, p)
	if err != nil {
		t.Fatal(err)
	}
	if nueva.Emision.Aviso != "documento.regenerado" || nueva.Emision.ID == primera.Emision.ID {
		t.Fatal("sin aviso o emisión nueva")
	}
	contar(i, 2)
	if err = b.Ejecutar(ctx, i, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.peticion_idempotente WHERE clave='auditoria-fallida'`).Scan(&n)
		if n != 0 {
			t.Fatal("la petición sobrevivió al fallo de auditoría")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// El disparador comprueba pertenencia incluso si el actor tiene todos los
	// equipos. La base de F1.01 aún conserva equipo nulo en este expediente.
	err = b.Ejecutar(ctx, i, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `INSERT INTO documento.emision SELECT gen_random_uuid(),tipo_documento,expediente_id,(SELECT id FROM catalogo.equipo ORDER BY id LIMIT 1),plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_sha256,huella_datos,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave FROM documento.emision WHERE expediente_id='22222222-2222-4222-8222-222222222222' LIMIT 1`)
		return err
	})
	if err == nil {
		t.Fatal("pertenencia falsificada")
	}
	err = b.Ejecutar(ctx, i, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `UPDATE documento.emision SET version_motor='alterada'`)
		return err
	})
	if err == nil {
		t.Fatal("emisión mutable")
	}
	h := adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesiones{}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: adaptadorHTTP.ConfiguracionDocumentos{Servicio: s}})
	cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))}
	consulta := httptest.NewRequest("GET", "https://etf.interno/api/sesion", nil)
	consulta.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, consulta)
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &acceso); err != nil || w.Code != 200 || acceso.CSRF == "" {
		t.Fatal("sesión", w.Code, err)
	}
	var descarga []byte
	for j := 0; j < 2; j++ {
		peticion := httptest.NewRequest("POST", "https://etf.interno/api/documentos/emisiones", bytes.NewBufferString(`{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`))
		peticion.AddCookie(cookie)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Origin", "https://etf.interno")
		peticion.Header.Set("X-CSRF-Token", acceso.CSRF)
		peticion.Header.Set("Idempotency-Key", "integracion-http")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticion)
		esperado := 201
		if j == 1 {
			esperado = 200
		}
		if w.Code != esperado || w.Header().Get("X-Documento-SHA256") != d.Huella(w.Body.Bytes()) || w.Header().Get("X-Emision-ID") == "" || w.Header().Get("X-Plantilla-Version") != "1.1.1" {
			t.Fatal("POST", w.Code, w.Body.String())
		}
		if j == 0 {
			descarga = append([]byte(nil), w.Body.Bytes()...)
		} else if !bytes.Equal(descarga, w.Body.Bytes()) {
			t.Fatal("POST repetido distinto")
		}
	}
	t.Log("POST 201 y repetición 200: DOCX idéntico; 6 peticiones concurrentes; RLS; auditoría atómica; expediente intacto; regeneración con aviso")
}
