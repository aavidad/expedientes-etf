package expediente_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	evRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/evaluacion"
	expRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	evApp "github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion"
	expApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	aud "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	exp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
)

// Las reglas siguientes son una fixture técnica; no son baremos del Programa.
type auditoriaEvaluacionFallida struct{}

func (auditoriaEvaluacionFallida) Confirmar(context.Context, aud.Datos) (aud.Evento, error) {
	return aud.Evento{}, errors.New("auditoria.no_disponible")
}

type catalogoEvaluacion struct{}

func (catalogoEvaluacion) Instrumento(context.Context, string) (bool, error) { return true, nil }
func (catalogoEvaluacion) Escala(context.Context, string) (d.Escala, error) {
	return d.Escala{ID: "11111111-1111-4111-8111-111111111111", Instrumento: "PRUEBA", Version: 1, Items: []d.Item{{Codigo: "item01", Tipo: "numerico", Minimo: 0, Maximo: 2, Paso: 1}}}, nil
}
func TestEvaluacionPersistenciaHTTPRLSAuditoria(t *testing.T) {
	b, ctx, deps, actor, id := prepararFamilia(t)
	adultos := []exp.Adulto{{Orden: 1, Nombre: "Inés", Apellidos: "Moreno Castillo", Papel: "MADRE", Sexo: "M"}}
	if _, e := expApp.Nuevo(deps).EditarFamilia(ctx, actor, id, "adultos-evaluacion", exp.CambioFamilia{Version: 2, Desde: "2026-02-03", Adultos: &adultos}); e != nil {
		t.Fatal(e)
	}
	ficha, e := expApp.Nuevo(deps).ConsultarFamilia(ctx, actor, id)
	if e != nil {
		t.Fatal(e)
	}
	s := &evApp.Servicio{Repo: &evRepo.Repositorio{}, Catalogo: catalogoEvaluacion{}, Familias: &expRepo.Repositorio{}, Adultos: expRepo.ConsultaAdultos{}, Tx: b, Auditoria: postgres.NuevoRegistroAuditoriaTransaccional(b), Ahora: time.Now}
	a := d.Actor{UsuarioID: actor.UsuarioID, EquipoID: actor.EquipoID, Perfiles: actor.Perfiles}
	fecha := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	valor := 1.0
	ap := d.Aplicacion{ID: "22222222-2222-4222-8222-222222222222", ExpedienteID: id, EquipoID: a.EquipoID, Instrumento: "PRUEBA", Momento: "inicial", Sujeto: d.Sujeto{Tipo: "figura_parental", AdultoID: ficha.Adultos[0].ID}, Fecha: &fecha, VersionEscalaID: "11111111-1111-4111-8111-111111111111", Puntuaciones: []d.Puntuacion{{Item: "item01", ValorNumerico: &valor}}}
	c := d.Cambio{Aplicacion: &ap, Clave: "crear"}
	r, e := s.Guardar(ctx, a, id, c)
	if e != nil || r.Aplicacion.Version != 1 {
		t.Fatal(r, e)
	}
	if _, e = s.Guardar(ctx, a, id, c); e != nil {
		t.Fatal(e)
	}
	c.Clave = "conflicto"
	if _, e = s.Guardar(ctx, a, id, c); e == nil {
		t.Fatal("falta conflicto")
	} else {
		var conflicto d.Conflicto
		if !errors.As(e, &conflicto) || conflicto.VersionActual != 1 {
			t.Fatal(e)
		}
	}
	c.Clave = "corregir"
	c.VersionEsperada = 1
	ap.DescripcionCualitativa = "Revisada con la familia."
	if _, e = s.Guardar(ctx, a, id, c); e != nil {
		t.Fatal(e)
	}
	datos, e := s.Leer(ctx, a, id)
	if e != nil || len(datos.Escalas) != 1 || datos.Escalas[0].Version != 2 || len(datos.Escalas[0].Puntuaciones) != 1 || *datos.Escalas[0].Puntuaciones[0].ValorNumerico != 1 {
		t.Fatal(datos, e)
	}
	c.Clave = "corregir"
	ap.DescripcionCualitativa = "Otro cambio."
	if _, e = s.Guardar(ctx, a, id, c); !errors.Is(e, d.ErrIdempotencia) {
		t.Fatal(e)
	}
	insAdulto := d.InstrumentoAplicado{ID: "88888888-8888-4888-8888-888888888888", ExpedienteID: id, EquipoID: a.EquipoID, Codigo: "PRUEBA", Fecha: fecha, Sujeto: &d.Sujeto{Tipo: "figura_parental", AdultoID: ficha.Adultos[0].ID}, Momento: "inicial"}
	instrumentoAdulto := d.Cambio{Instrumento: &insAdulto, Clave: "instrumento-adulto"}
	if _, err := s.Guardar(ctx, a, id, instrumentoAdulto); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Guardar(ctx, a, id, instrumentoAdulto); err != nil {
		t.Fatal(err)
	}
	leido, err := s.Leer(ctx, a, id)
	if err != nil || len(leido.Instrumentos) != 1 || leido.Instrumentos[0].Sujeto.AdultoID != ficha.Adultos[0].ID || leido.Instrumentos[0].Momento != "inicial" {
		t.Fatal(leido, err)
	}
	insAdulto.ID = "44444444-4444-4444-8444-444444444444"
	insAdulto.Sujeto.AdultoID = "55555555-5555-4555-8555-555555555555"
	instrumentoAdulto.Clave = "instrumento-ajeno"
	if _, err := s.Guardar(ctx, a, id, instrumentoAdulto); !errors.Is(err, d.ErrSujeto) {
		t.Fatal(err)
	}
	insAdulto.Sujeto.AdultoID = ficha.Adultos[0].ID
	insAdulto.Momento = "derivacion"
	instrumentoAdulto.Clave = "instrumento-derivacion"
	if _, err := s.Guardar(ctx, a, id, instrumentoAdulto); !errors.Is(err, d.ErrSujeto) {
		t.Fatal(err)
	}
	c.Clave = "sin-auditoria"
	c.VersionEsperada = 2
	s.Auditoria = auditoriaEvaluacionFallida{}
	if _, e = s.Guardar(ctx, a, id, c); e == nil {
		t.Fatal("guardó sin auditoría")
	}
	if r, e := s.Leer(ctx, a, id); e == nil || len(r.Escalas) != 0 {
		t.Fatal("entregó sin auditoría", r, e)
	}
	s.Auditoria = postgres.NuevoRegistroAuditoriaTransaccional(b)
	for _, perfil := range []string{"consulta", "administracion"} {
		ajeno := a
		ajeno.Perfiles = []string{perfil}
		if _, e = s.Leer(ctx, ajeno, id); !errors.Is(e, d.ErrAcceso) {
			t.Fatal(perfil, e)
		}
	}
	otro := a
	otro.EquipoID = "99999999-9999-4999-8999-999999999999"
	if _, e = s.Leer(ctx, otro, id); !errors.Is(e, d.ErrNoEncontrada) {
		t.Fatal(e)
	}
	if e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: otro.EquipoID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var n int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM evaluacion.respuesta`).Scan(&n); e != nil {
			return e
		}
		if n != 0 {
			t.Fatal("RLS permitió leer equipo ajeno")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	ap.Sujeto.AdultoID = "99999999-9999-4999-8999-999999999999"
	c.Clave = "adulto-ajeno"
	if _, e = s.Guardar(ctx, a, id, c); !errors.Is(e, d.ErrSujeto) {
		t.Fatal(e)
	}
	ap.Sujeto = d.Sujeto{Tipo: "menor", MenorID: "99999999-9999-4999-8999-999999999999"}
	c.Clave = "menor"
	if _, e = s.Guardar(ctx, a, id, c); !errors.Is(e, d.ErrNoDisponible) {
		t.Fatal(e)
	}

	apBloqueada := datos.Escalas[0]
	apBloqueada.Momento = "derivacion"
	if _, e = s.Guardar(ctx, a, id, d.Cambio{Aplicacion: &apBloqueada, VersionEsperada: 2, Clave: "derivacion-bloqueada"}); !errors.Is(e, d.ErrAcceso) {
		t.Fatal(e)
	}
	apBloqueada.Momento = "final"
	apBloqueada.Sujeto = d.Sujeto{Tipo: "profesional", ProfesionalID: a.UsuarioID}
	if _, e = s.Guardar(ctx, a, id, d.Cambio{Aplicacion: &apBloqueada, Clave: "profesional-sin-historia"}); !errors.Is(e, d.ErrNoDisponible) {
		t.Fatal(e)
	}
	// Pruebas directas de restricciones: la aplicación no sustituye estas barreras.
	identidadSQL := plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}
	exigirSQL := func(codigo string, operacion func(context.Context) error) {
		t.Helper()
		e := b.Ejecutar(ctx, identidadSQL, operacion)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != codigo {
			t.Fatalf("restricción %s: %v", codigo, e)
		}
	}
	exigirSQL("23514", func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.instrumento_aplicado(id,expediente_id,equipo_id,datos) VALUES($1,$2,$3,'{}')`, "44444444-4444-4444-8444-444444444444", id, a.EquipoID)
		return e
	})
	sujetoSinTipo := map[string]string{"profesional_id": a.UsuarioID}
	meta := map[string]any{"id": "44444444-4444-4444-8444-444444444444", "expediente_id": id, "equipo_id": a.EquipoID, "instrumento": "PRUEBA", "momento": "final", "version": 1, "sujeto": sujetoSinTipo}
	exigirSQL("23514", func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.aplicacion_escala(id,expediente_id,equipo_id,instrumento,momento,sujeto,datos,version) VALUES($1,$2,$3,'PRUEBA','final',$4,$5,1)`, meta["id"], id, a.EquipoID, sujetoSinTipo, meta)
		return e
	})
	_, ctx2, deps2, actor2, id2 := prepararFamilia(t)
	adultos2 := []exp.Adulto{{Orden: 1, Nombre: "Manuel", Apellidos: "Moreno Ruiz", Papel: "PADRE", Sexo: "H"}}
	if _, e = expApp.Nuevo(deps2).EditarFamilia(ctx2, actor2, id2, "adultos-segunda-familia", exp.CambioFamilia{Version: 2, Desde: "2026-02-03", Adultos: &adultos2}); e != nil {
		t.Fatal(e)
	}
	ficha2, e := expApp.Nuevo(deps2).ConsultarFamilia(ctx2, actor2, id2)
	if e != nil {
		t.Fatal(e)
	}
	apCruzada := datos.Escalas[0]
	apCruzada.ID = "55555555-5555-4555-8555-555555555555"
	apCruzada.Sujeto.AdultoID = ficha2.Adultos[0].ID
	apCruzada.Version = 1
	apCruzada.Momento = "final"
	exigirSQL("23503", func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.aplicacion_escala(id,expediente_id,equipo_id,adulto_id,instrumento,momento,sujeto,datos,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,1)`, apCruzada.ID, id, a.EquipoID, apCruzada.Sujeto.AdultoID, apCruzada.Instrumento, apCruzada.Momento, apCruzada.Sujeto, apCruzada)
		return e
	})
	exigirSQL("23503", func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO evaluacion.respuesta(aplicacion_id,expediente_id,equipo_id,item,valor_codigo) VALUES($1,$2,$3,'cruzada','valor')`, ap.ID, id2, a.EquipoID)
		return e
	})
	// HTTP real con sesión, CSRF y servicio PostgreSQL.
	ss := &sesionFamilia{sesionAceptacion: sesionAceptacion{actor.UsuarioID}, equipo: a.EquipoID, perfil: identidad.ProfesionalETF}
	h := httpETF.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Modulos: []httpETF.Modulo{httpETF.ModuloEvaluacion{Servicio: s}}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta, csrf string, cuerpo []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	w := pedir("GET", "/api/sesion", "", nil)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &acceso) != nil {
		t.Fatal(w.Code)
	}
	ruta := "/api/expedientes/" + id + "/evaluaciones"
	ins := d.InstrumentoAplicado{ID: "33333333-3333-4333-8333-333333333333", ExpedienteID: id, EquipoID: a.EquipoID, Codigo: "SCORE15", Fecha: fecha}
	cuerpo, _ := json.Marshal(d.Cambio{Instrumento: &ins, Clave: "instrumento"})
	if w = pedir("POST", ruta, "", cuerpo); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = pedir("POST", ruta, acceso.CSRF, cuerpo); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = pedir("GET", ruta, "", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = pedir("DELETE", ruta, acceso.CSRF, nil); w.Code != 405 {
		t.Fatal(w.Code)
	}
	c = d.Cambio{Aplicacion: &datos.Escalas[0], VersionEsperada: 1, Clave: "http-conflicto"}
	cuerpo, _ = json.Marshal(c)
	if w = pedir("POST", ruta, acceso.CSRF, cuerpo); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	t.Log("PostgreSQL y HTTP: respuestas por ítem, adulto vigente, versión 409, idempotencia, RLS, CSRF y reversión por auditoría comprobados")
}
