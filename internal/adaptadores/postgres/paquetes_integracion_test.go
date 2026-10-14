package postgres_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5"
)

type antivirusPaquetes struct{}

func (antivirusPaquetes) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusLimpio, nil
}

func idiomaZIP(t *testing.T, codigo, version, titulo string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for nombre, contenido := range map[string]any{
		"idioma.json":       dominio.Idioma{Codigo: codigo, Nombre: "Français", Version: version},
		"textos/comun.json": map[string]string{"comun.titulo": titulo, "comun.error_carga": "Carga interrumpida", "comun.reintentar": "Reintentar"},
	} {
		f, err := z.Create(nombre)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.NewEncoder(f).Encode(contenido); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestIntegracionInstalacionPaquetes(t *testing.T) {
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, func(k string) string {
		if k == "ETF_BD_MAX_CONEXIONES" {
			return "8"
		}
		return os.Getenv(k)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	if err = b.Comprobar(ctx); err != nil {
		t.Fatal(err)
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(os.Getenv("ETF_BD_HOST"), os.Getenv("ETF_BD_PUERTO")), Path: "/" + os.Getenv("ETF_BD_NOMBRE"), User: url.UserPassword("etf_migrador", os.Getenv("ETF_MIGRADOR_CONTRASENA")), RawQuery: "sslmode=disable"}
	propietario, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer propietario.Close(context.Background())
	if _, err = propietario.Exec(ctx, "SET ROLE etf_propietario"); err != nil {
		t.Fatal(err)
	}
	var pgVersion int
	if err = propietario.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&pgVersion); err != nil || pgVersion < 180000 || pgVersion >= 190000 {
		t.Fatal(pgVersion, err)
	}

	actor := dominio.Actor{UsuarioID: "00000000-0000-4000-8000-000000000012", Perfiles: []string{"administracion"}}
	base := t.TempDir()
	t.Cleanup(func() {
		if err := filepath.WalkDir(base, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return os.Chmod(ruta, 0750)
			}
			return nil
		}); err != nil {
			t.Error(err)
		}
	})
	idiomas, temas := filepath.Join(base, "idiomas"), filepath.Join(base, "temas")
	if err = os.MkdirAll(idiomas, 0750); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(temas, 0750); err != nil {
		t.Fatal(err)
	}
	disco := ficheros.NuevosPaquetes(idiomas, temas)
	almacen := ficheros.NuevoAlmacenVersiones(idiomas, temas)
	pub := postgres.NuevaPublicacionPaquetes(b)
	instalador := paquete.NuevoInstalador(disco, antivirusPaquetes{}, almacen, pub, postgres.NuevoRegistroAuditoria(b), time.Now)
	activos := ficheros.NuevosActivos(disco, pub)
	var nonce [3]byte
	_, _ = rand.Read(nonce[:])
	codigo := "fr-" + hex.EncodeToString(nonce[:])
	preparar := func(version, titulo string) dominio.Preparacion {
		p, err := instalador.Preparar(ctx, actor, "idioma", codigo, idiomaZIP(t, codigo, version, titulo))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	contar := func(consulta string, args ...any) int {
		var n int
		if err := propietario.QueryRow(ctx, consulta, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	comprobarDisco := func(cantidad int) {
		vs, err := almacen.Huerfanas(ctx, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, v := range vs {
			if v.Codigo == codigo {
				n++
			}
		}
		if n != cantidad {
			t.Fatalf("rutas: %d; esperadas %d", n, cantidad)
		}
	}

	t.Run("ConcurrenteYServido", func(t *testing.T) {
		uno, dos := preparar("1.0.0", "Titre A"), preparar("2.0.0", "Titre B")
		var wg sync.WaitGroup
		salidas := make(chan error, 2)
		inicio := make(chan struct{})
		for _, p := range []dominio.Preparacion{uno, dos} {
			wg.Add(1)
			go func(ticket string) {
				defer wg.Done()
				<-inicio
				_, err := instalador.Activar(ctx, actor, ticket, 0)
				salidas <- err
			}(p.Ticket)
		}
		close(inicio)
		wg.Wait()
		close(salidas)
		ok, conflictos := 0, 0
		for err := range salidas {
			if err == nil {
				ok++
			} else if errors.Is(err, dominio.ErrConflicto) {
				conflictos++
			} else {
				t.Fatal(err)
			}
		}
		if ok != 1 || conflictos != 1 {
			t.Fatal(ok, conflictos)
		}
		r, err := instalador.Actual(ctx, actor, "idioma", codigo)
		if err != nil || r.Revision != 1 {
			t.Fatal(r, err)
		}
		if contar("SELECT count(*) FROM paquete.idioma_instalado WHERE codigo=$1", codigo) != 1 {
			t.Fatal("registro parcial")
		}
		if contar("SELECT count(*) FROM auditoria.evento e JOIN paquete.idioma_instalado i ON e.entidad_id=i.id WHERE i.codigo=$1 AND e.accion='paquete.activar' AND e.resultado='ok'", codigo) != 1 {
			t.Fatal("auditoría")
		}
		textos, err := activos.Textos(ctx, codigo)
		if err != nil || (textos["comun.titulo"] != "Titre A" && textos["comun.titulo"] != "Titre B") {
			t.Fatal(textos, err)
		}
		comprobarDisco(1)
		consulta := paquete.NuevaConsulta(activos, activos)
		// Lectura pública directa del paquete, sin depender del castellano de reserva.
		servidor := adaptadorHTTP.Nuevo(&lecturaPaquete{Paquetes: consulta, textos: activos}, nil, "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil,
			adaptadorHTTP.Opciones{Instalacion: adaptadorHTTP.ConfiguracionInstalacion{Instalacion: instalador}})
		w := httptest.NewRecorder()
		servidor.ServeHTTP(w, httptest.NewRequest("GET", "/api/idiomas/"+codigo+"/textos", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Titre") {
			t.Fatal(w.Code, w.Body.String())
		}
		t.Log("CONCURRENCIA-OK una activación, un conflicto; una versión y un evento; huérfano limpiado; contenido servido por HTTP")
	})

	t.Run("FalloAuditoriaRevierteYLimpia", func(t *testing.T) {
		p := preparar("3.0.0", "Titre C")
		antes, err := instalador.Actual(ctx, actor, "idioma", codigo)
		if err != nil {
			t.Fatal(err)
		}
		_, err = propietario.Exec(ctx, `CREATE FUNCTION paquete.fallar_ensayo() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.accion='paquete.activar' THEN RAISE EXCEPTION 'ensayo.fallo'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER fallar_ensayo BEFORE INSERT ON auditoria.evento FOR EACH ROW EXECUTE FUNCTION paquete.fallar_ensayo()`)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := propietario.Exec(context.Background(), "DROP TRIGGER fallar_ensayo ON auditoria.evento; DROP FUNCTION paquete.fallar_ensayo()"); err != nil {
				t.Error(err)
			}
		}()
		if _, err = instalador.Activar(ctx, actor, p.Ticket, antes.Revision); err == nil {
			t.Fatal("activación sin auditoría")
		}
		actual, err := instalador.Actual(ctx, actor, "idioma", codigo)
		if err != nil || actual.Revision != antes.Revision || actual.Version != antes.Version {
			t.Fatal(actual, err)
		}
		if contar("SELECT count(*) FROM paquete.idioma_instalado WHERE codigo=$1 AND version='3.0.0'", codigo) != 0 {
			t.Fatal("versión huérfana registrada")
		}
		comprobarDisco(1)
		t.Log("FALLO-MITAD-OK registro y referencia revertidos; versión anterior disponible; huérfano eliminado")
	})

	t.Run("PermisosRLSYInmutabilidad", func(t *testing.T) {
		for _, perfil := range []string{"equipo", "consulta", "coordinacion"} {
			_, err := instalador.Preparar(ctx, dominio.Actor{UsuarioID: actor.UsuarioID, Perfiles: []string{perfil}}, "idioma", codigo, nil)
			if !errors.Is(err, dominio.ErrPermiso) {
				t.Fatal(perfil, err)
			}
		}
		if err := b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var n int
			err = tx.QueryRow(ctx, "SELECT count(*) FROM paquete.idioma_instalado").Scan(&n)
			if err == nil && n != 0 {
				t.Fatal("RLS sin actor")
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		for _, sql := range []string{"DELETE FROM paquete.idioma_instalado", "UPDATE paquete.referencia_activa SET version=100", "SELECT paquete.activar('idioma','fr','1.0.0','a','a',0)"} {
			if err := b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
				tx, _ := postgres.DesdeContexto(ctx)
				_, err := tx.Exec(ctx, sql)
				return err
			}); err == nil {
				t.Fatal("permiso directo", sql)
			}
		}
		if _, err := propietario.Exec(ctx, "UPDATE paquete.idioma_instalado SET version='9.0.0' WHERE codigo=$1", codigo); err == nil {
			t.Fatal("versión mutable")
		}
		var completo bool
		if err := propietario.QueryRow(ctx, "SELECT bool_and(relrowsecurity AND relforcerowsecurity) FROM pg_class WHERE relnamespace='paquete'::regnamespace AND relkind='r'").Scan(&completo); err != nil || !completo {
			t.Fatal(completo, err)
		}
		t.Log("RLS-OK sin contexto cero filas; escritura sólo mediante activación autorizada; versiones inmutables")
	})

	t.Run("ConfirmacionInciertaEsperaYConserva", func(t *testing.T) {
		c, err := disco.Validar(ctx, "idioma", idiomaZIP(t, codigo, "4.0.0", "Titre D"))
		if err != nil {
			t.Fatal(err)
		}
		v, err := almacen.Guardar(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := instalador.Actual(ctx, actor, "idioma", codigo)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := propietario.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		if _, err = tx.Exec(ctx, "SELECT set_config('app.usuario_id',$1,true),set_config('app.perfiles','administracion',true)", actor.UsuarioID); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, "SELECT paquete.activar($1,$2,$3,$4,$5,$6)", v.Tipo, v.Codigo, v.Version, v.Huella, v.Ruta, ref.Revision); err != nil {
			t.Fatal(err)
		}
		limite, cancelar := context.WithTimeout(ctx, 100*time.Millisecond)
		if _, err = pub.Registrada(limite, v); err == nil {
			t.Fatal("limpieza leyó mientras activación no confirmada")
		}
		cancelar()
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		registrada, err := pub.Registrada(ctx, v)
		if err != nil || !registrada {
			t.Fatal(registrada, err)
		}
		// La limpieza reconoce tanto la versión actual como la anterior.
		c, err = disco.Validar(ctx, "idioma", idiomaZIP(t, codigo, "5.0.0", "Titre E"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = almacen.Guardar(ctx, c); err != nil {
			t.Fatal(err)
		}
		limpiador := paquete.NuevoInstalador(disco, antivirusPaquetes{}, almacen, pub, postgres.NuevoRegistroAuditoria(b), func() time.Time { return time.Now().Add(2 * time.Hour) })
		if err = limpiador.Limpiar(ctx); err != nil {
			t.Fatal(err)
		}
		comprobarDisco(2)
		t.Log("COMMIT-INCIERTO-OK espera bloqueo y conserva versión registrada")
	})
}

type lecturaPaquete struct {
	entrada.Paquetes
	textos *ficheros.Activos
}

func (l *lecturaPaquete) Textos(ctx context.Context, codigo string) (map[string]string, error) {
	return l.textos.Textos(ctx, codigo)
}
