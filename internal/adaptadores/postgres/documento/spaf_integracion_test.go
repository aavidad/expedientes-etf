package documento_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	entradaIdentidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// La ficha persistente es F2.01. Este productor sintético prueba el cálculo
// real F3.04 y exige la transacción PostgreSQL recibida por el puerto.
type lecturaSPAFWord struct {
	version  string
	cantidad int
}

func (f *lecturaSPAFWord) ConsultarSPAF(ctx context.Context, filtro est.FiltroSPAF, corte, conocido time.Time) ([]est.ExpedienteSPAF, est.Historia[est.ReglasSPAF], error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return nil, nil, err
	}
	var actor string
	if err := tx.QueryRow(ctx, `SELECT current_setting('app.usuario_id')`).Scan(&actor); err != nil || actor != usuario {
		return nil, nil, d.ErrAcceso
	}
	fecha := time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)
	filas := []est.ExpedienteSPAF{}
	for i := 1; i <= f.cantidad; i++ {
		datos := est.DatosSPAF{Menores: est.Dato[[]est.MenorSPAF]{Conocido: true, Valor: []est.MenorSPAF{{ID: fmt.Sprint(i), Nombre: "Lucía", Apellidos: "Serrano Ruiz", Nacimiento: est.Dato[time.Time]{Conocido: true, Valor: time.Date(2016, 6, 10, 0, 0, 0, 0, time.UTC)}}}}, Municipio: est.Dato[string]{Conocido: true, Valor: "18013"}}
		filas = append(filas, est.ExpedienteSPAF{ID: fmt.Sprintf("77777777-7777-4777-8777-%012d", i), EquipoID: filtro.EquipoID, Numero: fmt.Sprintf("06-%02d-26", i), Episodios: []est.Episodio{{ID: fmt.Sprint(i), Aceptacion: fecha, Apertura: fecha}}, Ficha: est.Historia[est.DatosSPAF]{{ID: f.version, Vigencia: est.Vigencia{Desde: fecha}, RegistradoEn: fecha, Dato: est.Dato[est.DatosSPAF]{Conocido: true, Valor: datos}}}})
	}
	reglas := est.Historia[est.ReglasSPAF]{{ID: "regla-1", Vigencia: est.Vigencia{Desde: fecha}, RegistradoEn: fecha, Dato: est.Dato[est.ReglasSPAF]{Conocido: true, Valor: est.ReglasSPAF{MotivosCierre: map[est.Motivo]string{}}}}}
	return filas, reglas, nil
}

type relojSPAFWord struct{}

func (relojSPAFWord) Ahora() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }

type sesionesSPAFWord struct {
	sesiones
	perfil identidad.Perfil
	equipo string
}

func (s sesionesSPAFWord) Consultar(context.Context, string) (entradaIdentidad.Acceso, error) {
	return entradaIdentidad.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{s.perfil}, EquipoID: s.equipo}}, nil
}

type motorSPAFVersionNueva struct{ word.MotorSPAF }

func (motorSPAFVersionNueva) Version() string { return "docx-spaf-2.0.0" }

type auditoriaSPAFRota struct{ *postgres.RegistroAuditoria }

func (auditoriaSPAFRota) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, d.ErrNoDisponible
}

func TestSPAFWordPostgresHTTP(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	// Verifica las políticas corregidas con el rol real de aplicación.
	for _, perfil := range []string{"profesional_etf", "", "equipo"} {
		perfiles := []string{perfil}
		if perfil == "" {
			perfiles = nil
		}
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: perfiles}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"equipo", "centro_ss", "municipio", "version", "firmante"} {
				var total int
				if err := tx.QueryRow(ctx, "SELECT count(*) FROM catalogo."+tabla).Scan(&total); err != nil {
					return err
				}
				if (perfil == "profesional_etf" && total == 0) || (perfil != "profesional_etf" && total != 0) {
					t.Fatalf("catálogo %s perfil %q: %d", tabla, perfil, total)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM catalogo.equipo`).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Fatal("catálogo sin contexto", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r := postgres.NuevoRegistroAuditoria(b)
	var equipo, ajeno string
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		if err := tx.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo='06'`).Scan(&equipo); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo='07'`).Scan(&ajeno)
	})
	if err != nil {
		t.Fatal(err)
	}
	fuente := &lecturaSPAFWord{version: "ficha-1", cantidad: 16}
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	calculo := estadistica.NuevoSPAF(estadistica.DependenciasSPAF{Lectura: fuente, Transacciones: b, Auditoria: r, Reloj: relojSPAFWord{}})
	deps := app.DependenciasEmisionSPAF{Calculo: calculo, Repositorio: repo.Repositorio{}, Catalogos: &catalogo.Repositorio{}, Referencias: archivos, Presentacion: archivos, Idiomas: archivos, Ficheros: archivos, Motor: word.MotorSPAF{}, Transacciones: b, Auditoria: r, Reloj: relojSPAFWord{}, Identificadores: ids{}}
	s := app.NuevoEmisionSPAF(deps)
	actor := d.Actor{UsuarioID: usuario, Perfiles: []string{"profesional_etf"}, EquipoID: equipo}
	p := d.SolicitudSPAF{EquipoID: equipo, Anio: 2026, Mes: 1, Idioma: "es", Clave: "spaf-inicial", Versiones: d.VersionesSPAF{Regla: "regla-1", Fichas: map[string]string{}}}
	for i := 1; i <= fuente.cantidad; i++ {
		p.Versiones.Fichas[fmt.Sprintf("77777777-7777-4777-8777-%012d", i)] = fuente.version
	}
	h := func(perfil identidad.Perfil, propio string) http.Handler {
		return adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesionesSPAFWord{perfil: perfil, equipo: propio}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: adaptadorHTTP.ConfiguracionDocumentos{SPAF: s}})
	}
	cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"}
	token := func(handler http.Handler) string {
		req := httptest.NewRequest("GET", "https://etf.interno/api/sesion", nil)
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var v struct {
			CSRF string `json:"csrf"`
		}
		if json.Unmarshal(w.Body.Bytes(), &v) != nil || v.CSRF == "" {
			t.Fatal(w.Code, w.Body.String())
		}
		return v.CSRF
	}
	handler := h(identidad.ProfesionalETF, equipo)
	csrf := token(handler)
	post := func(handler http.Handler, p d.SolicitudSPAF, metodo, csrf string, anonimo bool) *httptest.ResponseRecorder {
		cuerpo, _ := json.Marshal(p)
		req := httptest.NewRequest(metodo, fmt.Sprintf("https://etf.interno/api/equipos/%s/tabla-spaf/%d/%d.docx", p.EquipoID, p.Anio, p.Mes), bytes.NewReader(cuerpo))
		if !anonimo {
			req.AddCookie(cookie)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://etf.interno")
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Idempotency-Key", p.Clave)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	primera := post(handler, p, "POST", csrf, false)
	if primera.Code != 201 {
		t.Fatal(primera.Code, primera.Body.String())
	}
	xml := xmlDocumento(t, primera.Body.Bytes())
	if strings.Contains(xml, "{{") || strings.Count(xml, "Serrano Ruiz, Lucía") != 16 || !strings.Contains(xml, "Alhama de Granada") || !strings.Contains(xml, "10/06/2016") || primera.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("Word sin completar o datos omitidos")
	}
	if os.Getenv("ETF_SPAF_WORD_SALIDA") != "" {
		if err := os.WriteFile(os.Getenv("ETF_SPAF_WORD_SALIDA"), primera.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repetida := post(handler, p, "POST", csrf, false)
	if repetida.Code != 200 || !bytes.Equal(primera.Body.Bytes(), repetida.Body.Bytes()) || primera.Header().Get("X-Emision-ID") != repetida.Header().Get("X-Emision-ID") {
		t.Fatal("idempotencia", repetida.Code, repetida.Body.String())
	}
	for _, caso := range []struct {
		nombre               string
		perfil               identidad.Perfil
		equipo, metodo, csrf string
		anonimo              bool
		estado               int
	}{
		{"equipo ajeno", identidad.ProfesionalETF, ajeno, "POST", csrf, false, 403},
		{"consulta", identidad.Consulta, equipo, "POST", csrf, false, 403},
		{"administración", identidad.Administracion, equipo, "POST", csrf, false, 403},
		{"anónimo", identidad.ProfesionalETF, equipo, "POST", csrf, true, 401},
		{"GET", identidad.ProfesionalETF, equipo, "GET", csrf, false, 405},
		{"sin CSRF", identidad.ProfesionalETF, equipo, "POST", "", false, 403},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := post(h(caso.perfil, caso.equipo), p, caso.metodo, caso.csrf, caso.anonimo)
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	p.Mes = 2
	if w := post(handler, p, "POST", csrf, false); w.Code != 422 {
		t.Fatal("clave con otro mes", w.Code, w.Body.String())
	}
	p.Mes = 1
	depsMotor := deps
	depsMotor.Motor = motorSPAFVersionNueva{}
	if resultado, err := app.NuevoEmisionSPAF(depsMotor).EmitirSPAF(ctx, actor, p); err == nil || len(resultado.Contenido) != 0 {
		t.Fatal("motor distinto reproduce bajo misma clave", err)
	} else {
		var conflicto *d.ConflictoReproduccion
		if !errors.As(err, &conflicto) || conflicto.Emision.ID != primera.Header().Get("X-Emision-ID") {
			t.Fatal("reproducción imposible sin recibo", err)
		}
	}
	p.Clave = "spaf-en"
	p.Idioma = "en"
	if w := post(h(identidad.Coordinacion, ""), p, "POST", csrf, false); w.Code != 201 || strings.Contains(xmlDocumento(t, w.Body.Bytes()), "{{") {
		t.Fatal("inglés/coordinación", w.Code, w.Body.String())
	}
	p.Idioma = "es"
	p.Clave = "spaf-inicial"
	fuente.version = "ficha-2"
	if w := post(handler, p, "POST", csrf, false); w.Code != 409 || !strings.Contains(w.Body.String(), "emision") {
		t.Fatal("versión", w.Code, w.Body.String())
	}
	if w := post(handler, p, "POST", csrf, false); w.Code != 409 || !strings.Contains(w.Body.String(), primera.Header().Get("X-Emision-ID")) {
		t.Fatal("ficha cambiada sin recibo original", w.Code, w.Body.String())
	}
	p.Clave = "spaf-version-obsoleta"
	if w := post(handler, p, "POST", csrf, false); w.Code != 409 || !strings.Contains(w.Body.String(), "version_actual") {
		t.Fatal("vista previa obsoleta", w.Code, w.Body.String())
	}
	for id := range p.Versiones.Fichas {
		p.Versiones.Fichas[id] = fuente.version
	}
	p.Clave = "spaf-nueva-version"
	nueva := post(handler, p, "POST", csrf, false)
	if nueva.Code != 201 || nueva.Header().Get("X-Aviso-Clave") != "documento.regenerado" {
		t.Fatal("regeneración", nueva.Code, nueva.Body.String())
	}
	contar := func(perfiles []string, propio string, esperadas int) {
		t.Helper()
		if len(perfiles) == 1 && perfiles[0] == "" {
			perfiles = nil
		}
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: perfiles, EquipoID: propio}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var n int
			err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.emision WHERE tipo_documento='spaf'`).Scan(&n)
			if err == nil && n != esperadas {
				t.Fatalf("RLS %v: %d", perfiles, n)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// El adaptador se llama directamente para comprobar la segunda barrera RLS.
	var emisionValida d.Emision
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: actor.Perfiles, EquipoID: equipo}, func(ctx context.Context) error {
		var err error
		emisionValida, _, err = (repo.Repositorio{}).PeticionSPAF(ctx, actor, "spaf-inicial")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"profesional_etf", "consulta", "administracion"} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}, EquipoID: equipo}, func(ctx context.Context) error {
			copia := emisionValida
			copia.ID, _ = (ids{}).Nuevo()
			if perfil == "profesional_etf" {
				copia.EquipoID = ajeno
			}
			return (repo.Repositorio{}).GuardarSPAF(ctx, actor, p, strings.Repeat("a", 64), copia)
		})
		if err == nil {
			t.Fatal("escritura RLS permitió alcance ajeno", perfil)
		}
	}
	contar([]string{"profesional_etf"}, equipo, 3)
	contar([]string{"profesional_etf"}, ajeno, 0)
	contar([]string{"coordinacion"}, "", 3)
	for _, perfil := range []string{"consulta", "administracion", ""} {
		contar([]string{perfil}, equipo, 0)
	}
	deps.Auditoria = auditoriaSPAFRota{r}
	fallida := app.NuevoEmisionSPAF(deps)
	p.Clave = "spaf-sin-auditoria"
	if res, err := fallida.EmitirSPAF(ctx, actor, p); err == nil || len(res.Contenido) != 0 {
		t.Fatal("emisión sin auditoría", err)
	}
	contar([]string{"profesional_etf"}, equipo, 3)
	p.Clave = "spaf-concurrente"
	var wg sync.WaitGroup
	resultados := make(chan d.Resultado, 4)
	fallos := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); res, err := s.EmitirSPAF(ctx, actor, p); resultados <- res; fallos <- err }()
	}
	wg.Wait()
	close(resultados)
	close(fallos)
	for err := range fallos {
		if err != nil {
			t.Fatal(err)
		}
	}
	id := ""
	for res := range resultados {
		if id == "" {
			id = res.Emision.ID
		}
		if id != res.Emision.ID {
			t.Fatal("emisión duplicada")
		}
	}
	contar([]string{"profesional_etf"}, equipo, 4)
	marca, err := r.Confirmar(ctx, auditoria.Datos{OcurridoEn: relojSPAFWord{}.Ahora(), UsuarioID: usuario, Perfiles: []string{"coordinacion"}, Accion: "documento.comprobar_spaf", Entidad: "documento", Resultado: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := r.Exportar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"administracion"}}, 1, marca.ID)
	if err != nil {
		t.Fatal(err)
	}
	encontrados := 0
	for _, evento := range exportacion.Eventos {
		if evento.Accion == "documento.emitir_spaf" && evento.Resultado == "ok" {
			encontrados++
		}
	}
	if encontrados < 7 {
		t.Fatal("emisiones sin eventos", encontrados)
	}
	if _, err := calculo.CalcularEnTransaccion(ctx, est.ActorSPAF{UsuarioID: usuario, Perfiles: actor.Perfiles, EquipoID: equipo}, est.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: equipo}, relojSPAFWord{}.Ahora()); !errors.Is(err, est.ErrDisponibleSPAF) {
		t.Fatal("lector fuera de transacción", err)
	}
	t.Log("SPAF-OK: Word es/en, 16 filas, permisos, CSRF, idempotencia concurrente, versión 409, RLS y reversión con auditoría fallida")
}
