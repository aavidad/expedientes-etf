package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// Ejecutar con ETF_POSTGRES_INTEGRACION=1 y ETF_BD_* contra el PostgreSQL 18
// desechable del Compose. Sin esa variable se omite; al activarla, cualquier
// problema de configuración/conexión falla, nunca se oculta como omisión.
func baseIntegracion(t *testing.T) (*Base, context.Context) {
	t.Helper()
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancelar)
	base, err := Nueva(ctx, func(clave string) string {
		if clave == "ETF_BD_MAX_CONEXIONES" {
			return "1"
		}
		return os.Getenv(clave)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(base.Cerrar)
	if err := base.Comprobar(ctx); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := base.pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < 180000 || version >= 190000 {
		t.Fatalf("requiere PostgreSQL 18, recibido %d", version)
	}
	return base, ctx
}

func TestIntegracionContextoNoSeFiltraEntreTransacciones(t *testing.T) {
	base, ctx := baseIntegracion(t)
	identidades := []plataforma.Identidad{
		{UsuarioID: "a66cfd72-7f02-4b1e-b934-5da9d245bc33", Perfiles: []string{"coordinacion", "administracion"}, EquipoID: "4a41bbdb-2ccd-48dc-b35a-08b8243127dd"},
		{UsuarioID: "a1e657df-4f31-486b-a975-fbce58ec99bb", Perfiles: []string{"equipo"}, EquipoID: "296d0524-541e-4dd3-a382-338b2d1296b7"},
		{},
	}
	var pidInicial int
	for n, identidad := range identidades {
		err := base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			ejecutor, err := DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var usuario, perfiles, equipos, rol string
			var pid int
			err = ejecutor.QueryRow(ctx, `SELECT current_setting('app.usuario_id', true),
				current_setting('app.perfiles', true), current_setting('app.equipos', true), current_user, pg_backend_pid()`).Scan(&usuario, &perfiles, &equipos, &rol, &pid)
			if err != nil {
				return err
			}
			esperadoUsuario := []string{identidades[0].UsuarioID, identidades[1].UsuarioID, ""}[n]
			esperadoPerfil := []string{"coordinacion,administracion", "equipo", ""}[n]
			esperadoEquipo := []string{"{4a41bbdb-2ccd-48dc-b35a-08b8243127dd}", "{296d0524-541e-4dd3-a382-338b2d1296b7}", ""}[n]
			if usuario != esperadoUsuario || perfiles != esperadoPerfil || equipos != esperadoEquipo || rol != "etf_app" {
				t.Fatal("contexto RLS incorrecto")
			}
			if n == 0 {
				pidInicial = pid
			} else if pid != pidInicial {
				t.Fatal("no se ha reutilizado la misma conexión")
			}
			if err := base.Ejecutar(ctx, identidad, func(context.Context) error { t.Fatal("operación anidada ejecutada"); return nil }); !errors.Is(err, ErrAnidada) {
				t.Fatal("anidamiento permitido")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		comprobarSinContexto(t, base, ctx, pidInicial)
	}
}

func comprobarSinContexto(t *testing.T, base *Base, ctx context.Context, pidEsperado int) {
	t.Helper()
	var usuario, perfiles, equipos string
	var pid int
	err := base.pool.QueryRow(ctx, `SELECT coalesce(current_setting('app.usuario_id', true), ''),
		coalesce(current_setting('app.perfiles', true), ''), coalesce(current_setting('app.equipos', true), ''), pg_backend_pid()`).Scan(&usuario, &perfiles, &equipos, &pid)
	if err != nil {
		t.Fatal(err)
	}
	if usuario != "" || perfiles != "" || equipos != "" || pid != pidEsperado {
		t.Fatal("contexto filtrado o conexión distinta")
	}
}

func TestIntegracionConfirmaORevierteLaOperacion(t *testing.T) {
	base, ctx := baseIntegracion(t)
	rechazo := errors.New("operacion.rechazada")
	for _, caso := range []struct {
		nombre, estado string
		fallo          bool
	}{
		{"confirmar", "committed", false}, {"revertir", "aborted", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var id string
			var pid int
			err := base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: "a66cfd72-7f02-4b1e-b934-5da9d245bc33", Perfiles: []string{"equipo"}}, func(ctx context.Context) error {
				ejecutor, err := DesdeContexto(ctx)
				if err != nil {
					return err
				}
				if err := ejecutor.QueryRow(ctx, "SELECT pg_current_xact_id()::text, pg_backend_pid()").Scan(&id, &pid); err != nil {
					return err
				}
				if caso.fallo {
					return rechazo
				}
				return nil
			})
			if (caso.fallo && !errors.Is(err, rechazo)) || (!caso.fallo && err != nil) {
				t.Fatal(err)
			}
			var estado string
			if err := base.pool.QueryRow(ctx, "SELECT pg_xact_status($1::xid8)", id).Scan(&estado); err != nil {
				t.Fatal(err)
			}
			if estado != caso.estado {
				t.Fatalf("estado: %s, esperado: %s", estado, caso.estado)
			}
			comprobarSinContexto(t, base, ctx, pid)
		})
	}
}

func TestIntegracionPanicoRevierteYLimpiaContexto(t *testing.T) {
	base, ctx := baseIntegracion(t)
	var id string
	var pid int
	func() {
		defer func() {
			if recover() != "operacion.panico" {
				t.Fatal("pánico no propagado")
			}
		}()
		_ = base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: "a66cfd72-7f02-4b1e-b934-5da9d245bc33", Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
			ejecutor, err := DesdeContexto(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := ejecutor.QueryRow(ctx, "SELECT pg_current_xact_id()::text, pg_backend_pid()").Scan(&id, &pid); err != nil {
				t.Fatal(err)
			}
			panic("operacion.panico")
		})
	}()
	var estado string
	if err := base.pool.QueryRow(ctx, "SELECT pg_xact_status($1::xid8)", id).Scan(&estado); err != nil {
		t.Fatal(err)
	}
	if estado != "aborted" {
		t.Fatalf("estado: %s", estado)
	}
	comprobarSinContexto(t, base, ctx, pid)
}

func TestIntegracionErrorSQLImpideConfirmar(t *testing.T) {
	base, ctx := baseIntegracion(t)
	var id string
	var pid int
	err := base.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		ejecutor, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		if err := ejecutor.QueryRow(ctx, "SELECT pg_current_xact_id()::text, pg_backend_pid()").Scan(&id, &pid); err != nil {
			return err
		}
		// Aunque el repositorio no propague su error, la transacción abortada
		// no debe presentarse como confirmada.
		if _, err := ejecutor.Exec(ctx, "SELECT 1 / 0"); err == nil {
			t.Fatal("consulta inválida aceptada")
		}
		return nil
	})
	if !errors.Is(err, ErrNoDisponible) {
		t.Fatalf("confirmación: %v", err)
	}
	var estado string
	if err := base.pool.QueryRow(ctx, "SELECT pg_xact_status($1::xid8)", id).Scan(&estado); err != nil {
		t.Fatal(err)
	}
	if estado != "aborted" {
		t.Fatalf("estado: %s", estado)
	}
	comprobarSinContexto(t, base, ctx, pid)
}

func TestIntegracionCancelacionRevierteLaOperacion(t *testing.T) {
	base, ctx := baseIntegracion(t)
	operacion, cancelar := context.WithCancel(ctx)
	defer cancelar()
	var id string
	err := base.Ejecutar(operacion, plataforma.Identidad{UsuarioID: "a66cfd72-7f02-4b1e-b934-5da9d245bc33"}, func(ctx context.Context) error {
		ejecutor, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		if err := ejecutor.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&id); err != nil {
			return err
		}
		cancelar()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
	var estado string
	if err := base.pool.QueryRow(ctx, "SELECT pg_xact_status($1::xid8)", id).Scan(&estado); err != nil {
		t.Fatal(err)
	}
	if estado != "aborted" {
		t.Fatalf("estado: %s", estado)
	}
	if err := base.Comprobar(ctx); err != nil {
		t.Fatal(err)
	}
}
