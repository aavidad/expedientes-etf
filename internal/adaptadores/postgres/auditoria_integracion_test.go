package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// Requiere Compose desechable y ETF_POSTGRES_INTEGRACION=1, ETF_BD_*.
// El migrador se usa sólo en la prueba de permisos del propietario y fixtures.
// No se omiten fallos de conexión cuando la integración está activada.
func TestIntegracionAuditoria(t *testing.T) {
	_, ctx := baseIntegracion(t)
	base, err := Nueva(ctx, func(clave string) string {
		if clave == "ETF_BD_MAX_CONEXIONES" {
			return "8"
		}
		return os.Getenv(clave)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(base.Cerrar)
	r := NuevoRegistroAuditoria(base)
	admin := administracionAuditoria()

	t.Run("ConcurrenciaYCodificacionCompatible", func(t *testing.T) {
		const cantidad = 32
		var grupo sync.WaitGroup
		eventos := make(chan auditoria.Evento, cantidad)
		fallos := make(chan error, cantidad)
		inicio := make(chan struct{})
		for i := 0; i < cantidad; i++ {
			grupo.Add(1)
			go func(i int) {
				defer grupo.Done()
				<-inicio
				d := datosAuditoria()
				// La cadena se ordena por ID, incluso con fechas decrecientes.
				d.OcurridoEn = d.OcurridoEn.Add(-time.Duration(i) * time.Second)
				if i%2 == 0 {
					d.UsuarioID, d.Perfiles = "sistema", []string{}
				}
				e, err := r.Confirmar(ctx, d)
				if err != nil {
					fallos <- err
					return
				}
				eventos <- e
			}(i)
		}
		close(inicio)
		grupo.Wait()
		close(eventos)
		close(fallos)
		for err := range fallos {
			t.Error(err)
		}
		if t.Failed() {
			return
		}
		var cadena []auditoria.Evento
		for e := range eventos {
			cadena = append(cadena, e)
		}
		sort.Slice(cadena, func(i, j int) bool { return cadena[i].ID < cadena[j].ID })
		exportacion, err := r.Exportar(ctx, admin, cadena[0].ID, cadena[len(cadena)-1].ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(exportacion.Eventos) != cantidad || exportacion.Verificar() != nil {
			t.Fatal("cadena concurrente incompleta o rota")
		}
		for i, e := range cadena {
			if exportacion.Eventos[i].Huella != e.Huella {
				t.Fatal("confirmación distinta de la exportación persistida")
			}
		}
		t.Log("32 inserciones concurrentes: cadena SQL comprobada por el dominio Go")
	})

	t.Run("PermisosMinimosYPerfiles", func(t *testing.T) {
		for _, sql := range []string{
			"UPDATE auditoria.evento SET resultado = 'error'",
			"DELETE FROM auditoria.evento", "TRUNCATE auditoria.evento",
			"INSERT INTO auditoria.evento DEFAULT VALUES", "SELECT * FROM auditoria.evento",
			"SELECT * FROM auditoria.sello_diario", "UPDATE auditoria.sello_diario SET dia = dia",
			"SELECT nextval('auditoria.evento_id_seq')", "CREATE TABLE auditoria.intrusion(id int)",
			"SELECT auditoria.texto_binario('intrusion')",
		} {
			err := base.Ejecutar(ctx, admin, func(ctx context.Context) error {
				ejecutor, _ := DesdeContexto(ctx)
				_, err := ejecutor.Exec(ctx, sql)
				return err
			})
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "42501" {
				t.Fatalf("permiso permitido o error distinto: %s: %v", sql, err)
			}
		}
		for _, perfiles := range [][]string{nil, {"equipo"}, {"consulta"}, {"coordinacion"}} {
			i := plataforma.Identidad{UsuarioID: admin.UsuarioID, Perfiles: perfiles}
			if _, err := r.Exportar(ctx, i, 1, math.MaxInt64); err == nil || err.Error() != "auditoria.permiso_denegado" {
				t.Fatal("exportación sin Administración", err)
			}
			if _, err := r.Verificar(ctx, i, 1, math.MaxInt64); err == nil || err.Error() != "auditoria.permiso_denegado" {
				t.Fatal("verificación sin permiso", err)
			}
		}
		if _, err := r.Exportar(ctx, plataforma.Identidad{}, 1, math.MaxInt64); err == nil || err.Error() != "auditoria.permiso_denegado" {
			t.Fatal("sistema exportó datos", err)
		}
		t.Log("etf_app: INSERT/SELECT/UPDATE/DELETE/TRUNCATE y funciones privadas rechazados; exportación sólo Administración")
	})

	t.Run("TransaccionConjuntaYConfirmacion", func(t *testing.T) {
		rechazo := errors.New("operacion.rechazada")
		var revertido auditoria.Evento
		err := base.Ejecutar(ctx, admin, func(ctx context.Context) error {
			d := datosAuditoria()
			d.Accion = "auditoria.operacion_revertida"
			var err error
			revertido, err = r.RegistrarEnTransaccion(ctx, d)
			if err != nil {
				return err
			}
			if _, err := r.Confirmar(ctx, d); !errors.Is(err, ErrAnidada) {
				return errors.New("auditoria.confirmacion_invalida")
			}
			return rechazo
		})
		if !errors.Is(err, rechazo) {
			t.Fatal(err)
		}
		d := datosAuditoria()
		d.Accion = "auditoria.operacion_confirmada"
		var confirmado auditoria.Evento
		err = base.Ejecutar(ctx, admin, func(ctx context.Context) error {
			var err error
			confirmado, err = r.RegistrarEnTransaccion(ctx, d)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		exportacion, err := r.Exportar(ctx, admin, revertido.ID, confirmado.ID)
		if err != nil || len(exportacion.Eventos) != 1 || exportacion.Eventos[0].ID != confirmado.ID {
			t.Fatal("evento revertido persistido o hueco inválido", err)
		}
		exportacion, err = r.Exportar(ctx, admin, 1, math.MaxInt64)
		if err != nil || exportacion.Eventos[len(exportacion.Eventos)-1].Accion != "auditoria.exportar" {
			t.Fatal("exportación entregada sin apunte confirmado", err)
		}
		// Una identidad distinta produce error y revierte el apunte.
		err = base.Ejecutar(ctx, admin, func(ctx context.Context) error {
			d := datosAuditoria()
			d.UsuarioID = "e4274f2f-f454-4970-86ce-5e10641f796c"
			_, err := r.RegistrarEnTransaccion(ctx, d)
			return err
		})
		if err == nil || err.Error() != "auditoria.confirmacion_invalida" {
			t.Fatal("actor suplantado", err)
		}
		roto, err := r.Verificar(ctx, plataforma.Identidad{}, 1, math.MaxInt64)
		if err != nil || roto != nil {
			t.Fatal("cadena inválida después de rollback", err)
		}
		t.Log("rollback elimina evento; commit conjunto conserva evento; exportación auditada antes de entregar")
	})

	t.Run("ComprobadorSobreExportacionReal", func(t *testing.T) {
		exportacion, err := r.Exportar(ctx, admin, 1, math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		carpeta := t.TempDir()
		binario := filepath.Join(carpeta, "comprobar-auditoria")
		if instalado := os.Getenv("ETF_AUDITORIA_COMPROBADOR"); instalado != "" {
			binario = instalado
		} else if salida, err := exec.CommandContext(ctx, "go", "build", "-o", binario, "../../../cmd/comprobar-auditoria").CombinedOutput(); err != nil {
			t.Fatalf("compilar comprobador: %v: %s", err, salida)
		}
		ruta := filepath.Join(carpeta, "auditoria.json")
		contenido, err := exportacion.Codificar()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, contenido, 0600); err != nil {
			t.Fatal(err)
		}
		comprobar := func(codigo int, clave string, args ...string) {
			t.Helper()
			salida, err := exec.CommandContext(ctx, binario, append([]string{ruta}, args...)...).CombinedOutput()
			obtenido := 0
			if err != nil {
				var proceso *exec.ExitError
				if !errors.As(err, &proceso) {
					t.Fatal(err)
				}
				obtenido = proceso.ExitCode()
			}
			var resultado struct {
				Clave string `json:"clave"`
			}
			if json.Unmarshal(salida, &resultado) != nil || obtenido != codigo || resultado.Clave != clave {
				t.Fatalf("comprobador: código %d, salida %s", obtenido, salida)
			}
			t.Log(string(salida))
		}
		comprobar(0, "auditoria.sello_externo_coincide", strconv.FormatUint(exportacion.Final.ID, 10), exportacion.Final.Huella)
		exportacion.Eventos[0].Resultado = "denegado"
		contenido, _ = json.Marshal(exportacion)
		if err := os.WriteFile(ruta, contenido, 0600); err != nil {
			t.Fatal(err)
		}
		comprobar(1, "auditoria.huella_invalida")
	})

	t.Run("PropietarioSellosYAlteracionSQL", func(t *testing.T) {
		if os.Getenv("ETF_MIGRADOR_CONTRASENA") == "" {
			t.Skip("ETF_MIGRADOR_CONTRASENA para pruebas de propietario")
		}
		cfg := base.pool.Config().ConnConfig.Copy()
		cfg.User, cfg.Password = "etf_migrador", os.Getenv("ETF_MIGRADOR_CONTRASENA")
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			t.Fatal("conexión de ensayo del propietario fallida")
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "SET ROLE etf_propietario"); err != nil {
			t.Fatal(err)
		}
		for _, tabla := range []string{"evento", "sello_diario"} {
			for _, sql := range []string{"UPDATE auditoria." + tabla + " SET " + map[string]string{"evento": "resultado = resultado", "sello_diario": "dia = dia"}[tabla],
				"DELETE FROM auditoria." + tabla, "TRUNCATE auditoria." + tabla} {
				_, err := conn.Exec(ctx, sql)
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Message != "auditoria.solo_insercion" {
					t.Fatal("propietario pudo mutar la auditoría", err)
				}
			}
		}
		// Fixture de un evento ya confirmado ayer. La app nunca tiene INSERT.
		var previo auditoria.Punto
		if err := conn.QueryRow(ctx, "SELECT id, encode(huella, 'hex') FROM auditoria.evento ORDER BY id DESC LIMIT 1").Scan(&previo.ID, &previo.Huella); err != nil {
			t.Fatal(err)
		}
		var id uint64
		if err := conn.QueryRow(ctx, "SELECT nextval('auditoria.evento_id_seq')").Scan(&id); err != nil {
			t.Fatal(err)
		}
		d := datosAuditoria()
		d.UsuarioID, d.Perfiles = "sistema", []string{}
		e, err := auditoria.Encadenar(id, d, previo)
		if err != nil {
			t.Fatal(err)
		}
		_, err = conn.Exec(ctx, `INSERT INTO auditoria.evento(id, ocurrido_en, registrado_en, usuario_id, actor_sistema,
			perfiles, accion, entidad, entidad_id, expediente_id, resultado, huella_anterior, huella)
			OVERRIDING SYSTEM VALUE VALUES ($1, $2, (clock_timestamp() AT TIME ZONE 'UTC')::date - 1,
			NULL, true, '{}', $3, $4, $5::uuid, NULL, $6, decode($7, 'hex'), decode($8, 'hex'))`,
			e.ID, e.OcurridoEn, e.Accion, e.Entidad, e.EntidadID, e.Resultado, e.HuellaAnterior, e.Huella)
		if err != nil {
			t.Fatal(err)
		}
		dia := time.Now().UTC().AddDate(0, 0, -1)
		punto, err := r.SellarDiario(ctx, plataforma.Identidad{}, dia)
		if err != nil || punto.ID == 0 {
			t.Fatal("sello sin evento persistido", err)
		}
		repetido, err := r.SellarDiario(ctx, plataforma.Identidad{}, dia)
		if err != nil || repetido != punto {
			t.Fatal("sello diario mutable", err)
		}
		exportacion, err := r.Exportar(ctx, admin, 1, punto.ID)
		if err != nil || exportacion.Final != punto {
			t.Fatal("sello no corresponde a la exportación", err)
		}
		if _, err := r.SellarDiario(ctx, admin, time.Now().UTC()); err == nil || err.Error() != "auditoria.dia_invalido" {
			t.Fatal("día abierto sellado", err)
		}
		// El propietario puede retirar disparadores. El verificador debe detectar
		// la alteración; toda esta simulación se revierte antes de cerrar la prueba.
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, "ALTER TABLE auditoria.evento DISABLE TRIGGER evento_inmutable"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "UPDATE auditoria.evento SET resultado = 'error' WHERE id = $1", punto.ID); err != nil {
			t.Fatal(err)
		}
		var roto *int64
		if err := tx.QueryRow(ctx, "SELECT auditoria.verificar(1, 9223372036854775807)").Scan(&roto); err != nil || roto == nil || uint64(*roto) != punto.ID {
			t.Fatal("alteración SQL no detectada", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		tx, err = conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, "ALTER TABLE auditoria.evento DISABLE TRIGGER evento_inmutable"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "UPDATE auditoria.evento SET registrado_en = clock_timestamp() WHERE id = $1", punto.ID); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, "SELECT auditoria.verificar(1, 9223372036854775807)").Scan(&roto); err != nil || roto == nil || uint64(*roto) != punto.ID {
			t.Fatal("frontera diaria alterada no detectada", err)
		}
		t.Log("propietario: UPDATE/DELETE/TRUNCATE rechazados; sello repetible; alteración SQL y frontera diaria detectadas")
	})
}
