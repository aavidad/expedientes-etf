package catalogo_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	identidadDominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const actorID = "00000000-0000-4000-8000-000000000012"

func base(t *testing.T) (*postgres.Base, context.Context) {
	t.Helper()
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancelar)
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Cerrar)
	if err := b.Comprobar(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var v int
		if err := tx.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&v); err != nil {
			return err
		}
		if v < 180000 || v >= 190000 {
			t.Fatalf("requiere PostgreSQL 18: %d", v)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return b, ctx
}
func migrador(t *testing.T, ctx context.Context) pgx.Tx {
	t.Helper()
	if os.Getenv("ETF_MIGRADOR_CONTRASENA") == "" {
		t.Fatal("ETF_MIGRADOR_CONTRASENA requerida en ensayo")
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(os.Getenv("ETF_BD_HOST"), os.Getenv("ETF_BD_PUERTO")), Path: "/" + os.Getenv("ETF_BD_NOMBRE"), User: url.UserPassword("etf_migrador", os.Getenv("ETF_MIGRADOR_CONTRASENA")), RawQuery: "sslmode=disable"}
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatal("conexión migrador")
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE etf_propietario; SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='10s'"); err != nil {
		t.Fatal(err)
	}
	return tx
}
func rechaza(t *testing.T, ctx context.Context, tx pgx.Tx, sql, codigo string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(ctx, "SAVEPOINT rechazo"); err != nil {
		t.Fatal(err)
	}
	_, err := tx.Exec(ctx, sql, args...)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != codigo {
		t.Fatalf("rechazo esperado %s: %v", codigo, err)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT rechazo; RELEASE SAVEPOINT rechazo"); err != nil {
		t.Fatal(err)
	}
}
func TestRepositorioExigeTransaccion(t *testing.T) {
	_, err := (&adaptador.Repositorio{}).Vigente(context.Background(), "listas", time.Now())
	if !errors.Is(err, postgres.ErrTransaccion) {
		t.Fatal(err)
	}
}
func TestIntegracionCatalogoCargaYConsulta(t *testing.T) {
	b, ctx := base(t)
	c := aplicacion.NuevaConsulta(&adaptador.Repositorio{}, b)
	actor := dominio.Actor{UsuarioID: actorID, Perfiles: []string{"administracion"}}
	fecha := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	d, err := c.Datos(ctx, actor, "asignacion_municipios", fecha)
	if err != nil {
		t.Fatal(err)
	}
	if d.Version.Numero != 1 || d.Version.Estado != "publicada" {
		t.Fatal(d.Version)
	}
	for _, fuente := range []struct {
		nombre   string
		cantidad int
	}{{"equipos", len(d.Equipos)}, {"centros", len(d.Centros)}, {"municipios", len(d.Municipios)}} {
		contenido, err := os.ReadFile("../../../../datos/catalogos/" + fuente.nombre + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var datos map[string]json.RawMessage
		if err := json.Unmarshal(contenido, &datos); err != nil {
			t.Fatal(err)
		}
		var filas []json.RawMessage
		if err := json.Unmarshal(datos[fuente.nombre], &filas); err != nil {
			t.Fatal(err)
		}
		if fuente.cantidad != len(filas) {
			t.Fatalf("%s: BD %d, JSON %d", fuente.nombre, fuente.cantidad, len(filas))
		}
	}
	equipos := map[string]string{}
	centros := map[string]string{}
	for _, e := range d.Equipos {
		equipos[e.ID] = e.Codigo
	}
	for _, e := range d.Centros {
		centros[e.ID] = e.Codigo
	}
	encontrado, fuera, ela := false, false, false
	for _, m := range d.Municipios {
		if m.INE == "18102" {
			encontrado = true
			if m.EquipoID == nil || m.CentroID == nil || equipos[*m.EquipoID] != "06" || centros[*m.CentroID] != "PINOS_PUENTE_MOCLIN" {
				t.Fatal("Íllora: centro y equipo independientes")
			}
		}
		if m.INE == "18087" {
			fuera = true
			if m.AmbitoDiputacion || m.EquipoID != nil || m.CentroID != nil {
				t.Fatal("ámbito Granada")
			}
		}
		if m.Tipo == "ela" {
			ela = true
			if m.MatrizID == nil {
				t.Fatal("ELA sin matriz")
			}
		}
		if m.ZTS != nil {
			t.Fatal("ZTS todavía sin decisión")
		}
	}
	if !encontrado || !fuera || !ela {
		t.Fatal("faltan municipios")
	}
	opciones, err := c.Datos(ctx, actor, "listas", fecha)
	if err != nil || len(opciones.Opciones) != 25 {
		t.Fatal("opciones iniciales", len(opciones.Opciones), err)
	}
	confirmadas := map[string]bool{}
	for _, o := range opciones.Opciones {
		confirmadas[o.Lista+"."+o.Codigo] = true
	}
	if !confirmadas["subprograma.REUNIFICACION"] || !confirmadas["pronostico.FAVORABLE"] || !confirmadas["fin_seguimiento.CAMBIO_SITUACION"] {
		t.Fatal("códigos iniciales")
	}
	t.Log("LISTAS-OK 25 opciones vigentes, incluido fin de seguimiento; códigos sin etiquetas en BD")
	v, err := c.Memoria(ctx, actor, "asignacion_municipios", 2026)
	if err != nil || v.ID != d.Version.ID {
		t.Fatal(v, err)
	}
	if _, err := c.Vigente(ctx, actor, "listas", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal(err)
	}
	if _, err := c.Vigente(ctx, dominio.Actor{}, "listas", fecha); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal("sin contexto", err)
	}
	if _, err := c.Vigente(ctx, dominio.Actor{UsuarioID: actorID, Perfiles: []string{"desconocido"}}, "listas", fecha); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal("perfil desconocido", err)
	}
	for _, perfil := range []string{"coordinacion", "profesional_etf", "consulta", "administracion", "sistema"} {
		if _, err := c.Vigente(ctx, dominio.Actor{UsuarioID: actorID, Perfiles: []string{perfil}}, "listas", fecha); err != nil {
			t.Fatal(perfil, err)
		}
	}
	t.Logf("CARGA-OK %d equipos, %d centros, %d municipios; Íllora y ELA; RLS por perfil", len(d.Equipos), len(d.Centros), len(d.Municipios))
}
func TestIntegracionCatalogoPermisosMinimos(t *testing.T) {
	b, ctx := base(t)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var inseguras int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='catalogo' AND c.relkind='r' AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity
  OR pg_get_userbyid(c.relowner)<>'etf_propietario' OR has_table_privilege('etf_app',c.oid,'INSERT,UPDATE,DELETE,TRUNCATE'))`).Scan(&inseguras)
		if err != nil {
			return err
		}
		if inseguras != 0 {
			t.Fatal("tablas inseguras", inseguras)
		}
		var crear, publicar bool
		if err := tx.QueryRow(ctx, `SELECT has_schema_privilege('etf_app','catalogo','CREATE'),has_function_privilege('etf_app','catalogo.publicar(uuid,uuid)','EXECUTE')`).Scan(&crear, &publicar); err != nil {
			return err
		}
		if crear || publicar {
			t.Fatal("permisos sobrantes")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// El rechazo efectivo se comprueba en transacciones separadas.
	for _, sql := range []string{`UPDATE catalogo.version SET motivo='catalogo.alterado'`, `INSERT INTO catalogo.equipo(codigo,clave_texto,activo_desde) VALUES ('09','equipo.09','2026-01-01')`, `SELECT catalogo.publicar(gen_random_uuid(),gen_random_uuid())`, `TRUNCATE catalogo.lista_opcion`} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, sql)
			return err
		})
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "42501" {
			t.Fatalf("escritura no denegada: %v", err)
		}
	}
	t.Log("PERMISOS-OK app sólo SELECT, RLS forzada, sin CREATE ni publicación")
}
func TestIntegracionCatalogoPublicacionInmutable(t *testing.T) {
	_, ctx := base(t)
	tx := migrador(t, ctx)
	var anterior string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM catalogo.version WHERE ambito='listas' AND estado='publicada' ORDER BY numero DESC LIMIT 1`).Scan(&anterior); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{`UPDATE catalogo.version SET motivo='catalogo.alterado' WHERE id=$1`, `UPDATE catalogo.version SET vigente_hasta='2026-07-01' WHERE id=$1`, `DELETE FROM catalogo.version WHERE id=$1`} {
		rechaza(t, ctx, tx, sql, "P0001", anterior)
	}
	var nueva string
	err := tx.QueryRow(ctx, `INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES ('listas',3,'2030-07-01','catalogo.revision',$1) RETURNING id::text`, actorID).Scan(&nueva)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO catalogo.lista_opcion(version_id,lista,codigo,orden) VALUES ($1,'pronostico','FAVORABLE',1)`, nueva); err != nil {
		t.Fatal(err)
	}
	rechaza(t, ctx, tx, `INSERT INTO catalogo.municipio_asignacion(version_id,municipio_id) SELECT $1,id FROM catalogo.municipio LIMIT 1`, "P0001", nueva)
	if _, err := tx.Exec(ctx, `SELECT catalogo.publicar($1,$2)`, nueva, actorID); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{`UPDATE catalogo.lista_opcion SET codigo='DESFAVORABLE' WHERE version_id=$1`, `DELETE FROM catalogo.lista_opcion WHERE version_id=$1`, `INSERT INTO catalogo.lista_opcion(version_id,lista,codigo,orden) VALUES ($1,'pronostico','DESFAVORABLE',2)`} {
		rechaza(t, ctx, tx, sql, "P0001", nueva)
	}
	rechaza(t, ctx, tx, `UPDATE catalogo.lista_opcion SET version_id=$1 WHERE version_id=$1`, "P0001", nueva)
	rechaza(t, ctx, tx, `UPDATE catalogo.publicacion SET anterior_hasta='2030-07-01' WHERE version_id=$1`, "P0001", nueva)
	rechaza(t, ctx, tx, `TRUNCATE catalogo.lista_opcion`, "P0001")
	var historia int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM catalogo.publicacion WHERE version_id=$1 AND anterior_id=$2 AND anterior_hasta IS NULL`, nueva, anterior).Scan(&historia); err != nil || historia != 1 {
		t.Fatal("historia", historia, err)
	}
	for _, caso := range []struct {
		fecha string
		id    string
	}{{"2030-06-30", anterior}, {"2030-07-01", nueva}, {"2030-12-31", nueva}} {
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM catalogo.version WHERE ambito='listas' AND estado='publicada' AND daterange(vigente_desde,vigente_hasta,'[)') @> $1::date`, caso.fecha).Scan(&id); err != nil || id != caso.id {
			t.Fatal(caso.fecha, id, err)
		}
	}
	// Un ámbito no admite publicaciones retrospectivas que cambien el orden.
	var retro string
	if err := tx.QueryRow(ctx, `INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES ('listas',4,'2030-04-01','catalogo.revision',$1) RETURNING id::text`, actorID).Scan(&retro); err != nil {
		t.Fatal(err)
	}
	rechaza(t, ctx, tx, `SELECT catalogo.publicar($1,$2)`, "P0001", retro, actorID)
	// Un intervalo vacío se rechaza también en borrador.
	rechaza(t, ctx, tx, `INSERT INTO catalogo.version(ambito,numero,vigente_desde,vigente_hasta,motivo,creada_por) VALUES ('plazos',1,'2026-07-01','2026-07-01','catalogo.revision',$1)`, "23514", actorID)
	// Comprueba la exclusión independientemente de la barrera del disparador.
	if _, err := tx.Exec(ctx, `ALTER TABLE catalogo.version DISABLE TRIGGER proteger_version`); err != nil {
		t.Fatal(err)
	}
	rechaza(t, ctx, tx, `INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por,estado,publicada_por,publicada_en) VALUES ('listas',5,'2030-08-01','catalogo.revision',$1,'publicada',$1,now())`, "23P01", actorID)
	if _, err := tx.Exec(ctx, `ALTER TABLE catalogo.version ENABLE TRIGGER proteger_version`); err != nil {
		t.Fatal(err)
	}
	// Tampoco se trasladan filas publicadas a un borrador para eludir la protección.
	rechaza(t, ctx, tx, `UPDATE catalogo.lista_opcion SET version_id=$1 WHERE version_id=$2`, "P0001", retro, nueva)
	t.Log("VIGENCIAS-OK cierre controlado, historia, frontera 01/07, regla 31/12 e inmutabilidad")
}

func TestIntegracionCatalogoBorradoresSoloAdministracion(t *testing.T) {
	b, ctx := base(t)
	tx := migrador(t, ctx)
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES ('listas',100,'2030-01-01','catalogo.revision',$1) RETURNING id::text`, actorID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO catalogo.lista_opcion(version_id,lista,codigo,orden) VALUES ($1,'pronostico','RESERVADO',1)`, id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		limpieza := migrador(t, context.Background())
		if _, err := limpieza.Exec(context.Background(), `DELETE FROM catalogo.lista_opcion WHERE version_id=$1`, id); err != nil {
			t.Error(err)
		}
		if _, err := limpieza.Exec(context.Background(), `DELETE FROM catalogo.version WHERE id=$1`, id); err != nil {
			t.Error(err)
		}
		if err := limpieza.Commit(context.Background()); err != nil {
			t.Error(err)
		}
	})
	for _, perfil := range []string{"administracion", "profesional_etf", "coordinacion", "consulta", "sistema"} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{perfil}}, func(ctx context.Context) error {
			ejecutor, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			for _, sql := range []string{`SELECT count(*) FROM catalogo.version WHERE id=$1`, `SELECT count(*) FROM catalogo.lista_opcion WHERE version_id=$1`} {
				var n int
				if err := ejecutor.QueryRow(ctx, sql, id).Scan(&n); err != nil {
					return err
				}
				esperado := 0
				if perfil == "administracion" {
					esperado = 1
				}
				if n != esperado {
					t.Fatalf("%s: borrador %d, esperado %d", perfil, n, esperado)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(perfil, err)
		}
	}
	t.Log("BORRADORES-OK sólo Administración ve versión y opciones sin publicar")
}

func TestIntegracionCatalogoHTTP(t *testing.T) {
	b, ctx := base(t)
	consulta := aplicacion.NuevaConsulta(&adaptador.Repositorio{}, b)
	for _, autorizado := range []bool{false, true} {
		cfg := adaptadorHTTP.ConfiguracionSesion{}
		if autorizado {
			cfg = adaptadorHTTP.ConfiguracionSesion{Servicio: sesionCatalogoHTTP{}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
		}
		servidor := httptest.NewServer(adaptadorHTTP.Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), b.Comprobar, adaptadorHTTP.Opciones{Catalogos: adaptadorHTTP.ConfiguracionCatalogos{Consulta: consulta}, Sesion: cfg}))
		t.Cleanup(servidor.Close)
		cliente := servidor.Client()
		cliente.Timeout = 3 * time.Second
		for _, ruta := range []string{"listas/vigente?anio=2026", "asignacion_municipios/datos?fecha=2026-12-31"} {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, servidor.URL+"/api/admin/catalogos/"+ruta, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))})
			resp, err := cliente.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			contenido, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !autorizado {
				if resp.StatusCode != 403 || !strings.Contains(string(contenido), "identidad.acceso_denegado") {
					t.Fatal(resp.StatusCode, string(contenido))
				}
				continue
			}
			numero := `"numero":1`
			if strings.HasPrefix(ruta, "listas/") {
				numero = `"numero":2`
			}
			if resp.StatusCode != 200 || !strings.Contains(string(contenido), numero) {
				t.Fatal(resp.StatusCode, string(contenido))
			}
			if strings.HasPrefix(ruta, "asignacion") && !strings.Contains(string(contenido), `"ine":"18102"`) {
				t.Fatal("Íllora ausente")
			}
		}
	}
	t.Log("HTTP-OK servidor real: 403 por defecto; desarrollo consulta PostgreSQL y regla 31/12")
}

// El módulo de catálogo recibe una sesión validada como puerto de entrada.
// Su prueba aislada no depende de AD ni del esquema de identidad.
type sesionCatalogoHTTP struct{}

func (sesionCatalogoHTTP) AdmitirPeticion(context.Context) error                  { return nil }
func (sesionCatalogoHTTP) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesionCatalogoHTTP) Entrar(context.Context, identidadEntrada.Inicio) (identidadEntrada.SesionAbierta, error) {
	return identidadEntrada.SesionAbierta{}, identidadDominio.ErrCredencialesInvalidas
}
func (sesionCatalogoHTTP) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	return identidadEntrada.Acceso{Estado: identidadDominio.Activo, Usuario: identidadDominio.Usuario{ID: actorID, Estado: identidadDominio.Activo, Perfiles: []identidadDominio.Perfil{identidadDominio.Administracion}}}, nil
}
func (sesionCatalogoHTTP) Salir(context.Context, string) error { return nil }
