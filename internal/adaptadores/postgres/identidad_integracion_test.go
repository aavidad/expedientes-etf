package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func personaIntegracion(t *testing.T, u *Usuarios, ctx context.Context) dominio.Usuario {
	t.Helper()
	var guid dominio.GUID
	_, _ = rand.Read(guid[:])
	cuenta := fmt.Sprintf("ines.%x", guid[:4])
	persona, err := u.RegistrarAcceso(ctx, dominio.IdentidadDirectorio{GUID: guid, CuentaAD: cuenta, NombreMostrado: "Inés Moreno Castillo"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return persona
}
func TestIntegracionIdentidadGUIDRenombradoReutilizacionYRLS(t *testing.T) {
	b, ctx := baseIntegracion(t)
	u := NuevosUsuarios(b)
	persona := personaIntegracion(t, u, ctx)
	renombrada := persona.IdentidadDirectorio
	renombrada.CuentaAD += ".nueva"
	nueva, err := u.RegistrarAcceso(ctx, renombrada, time.Now().UTC())
	if err != nil || nueva.ID != persona.ID || nueva.Estado != dominio.PendienteAlta {
		t.Fatalf("renombrado: %v", err)
	}
	renombrada.GUID[0] ^= 1
	reutilizada, err := u.RegistrarAcceso(ctx, renombrada, time.Now().UTC())
	if err != nil || reutilizada.ID == persona.ID || reutilizada.GUID != renombrada.GUID || reutilizada.Estado != dominio.PendienteAlta || len(reutilizada.Perfiles) != 0 {
		t.Fatal("login reutilizado heredó identidad o no pudo entrar", err)
	}
	if _, err := u.Obtener(ctx, persona.ID); !errors.Is(err, dominio.ErrUsuarioNoDisponible) {
		t.Fatal("lectura anónima permitida")
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: persona.ID}, func(ctx context.Context) error {
		propio, err := u.Obtener(ctx, persona.ID)
		if err != nil || propio.ID != persona.ID || propio.GUID != persona.GUID || propio.CuentaAD != nueva.CuentaAD {
			t.Fatal("lectura propia", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var filtrado string
	if err := b.pool.QueryRow(ctx, `SELECT coalesce(current_setting('app.usuario_id',true),'')`).Scan(&filtrado); err != nil || filtrado != "" {
		t.Fatal("identidad filtrada")
	}
}
func TestIntegracionSesionesHuellaRollbackYCierre(t *testing.T) {
	b, ctx := baseIntegracion(t)
	u := NuevosUsuarios(b)
	s := NuevasSesiones(b)
	persona := personaIntegracion(t, u, ctx)
	var huella dominio.Huella
	_, _ = rand.Read(huella[:])
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ses, err := dominio.NuevaSesion(huella, persona.ID, ahora, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Crear(ctx, ses); err != nil {
		t.Fatal(err)
	}
	if err := s.Crear(ctx, ses); err == nil {
		t.Fatal("sesión sobrescrita")
	}
	if _, err := b.pool.Exec(ctx, `SELECT * FROM identidad.sesion`); err == nil {
		t.Fatal("sesiones accesibles directamente")
	}
	rechazo := errors.New("operacion.rechazada")
	if err := s.Modificar(ctx, huella, func(ctx context.Context, actual *dominio.Sesion) error {
		actual.UltimaActividadEn = ahora.Add(time.Minute)
		return rechazo
	}); !errors.Is(err, rechazo) {
		t.Fatal(err)
	}
	if err := s.Modificar(ctx, huella, func(ctx context.Context, actual *dominio.Sesion) error {
		if !actual.UltimaActividadEn.Equal(ahora) {
			t.Fatal("cambio no revertido")
		}
		obtenido, err := u.Obtener(ctx, actual.UsuarioID)
		if err != nil || obtenido.ID != persona.ID {
			t.Fatal("transacción compartida", err)
		}
		actual.UltimaActividadEn = ahora.Add(time.Minute)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Eliminar(ctx, huella); err != nil {
		t.Fatal(err)
	}
	if err := s.Eliminar(ctx, huella); err != nil {
		t.Fatal("cierre no idempotente", err)
	}
	if err := s.Modificar(ctx, huella, func(context.Context, *dominio.Sesion) error { t.Fatal("sesión resucitada"); return nil }); !errors.Is(err, dominio.ErrSesionNoValida) {
		t.Fatal(err)
	}
}
func TestIntegracionSesionesCierreConcurrenteNoResucita(t *testing.T) {
	b, ctx := baseIntegracion(t)
	otra, err := Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer otra.Cerrar()
	u := NuevosUsuarios(b)
	s := NuevasSesiones(b)
	persona := personaIntegracion(t, u, ctx)
	var huella dominio.Huella
	_, _ = rand.Read(huella[:])
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ses, _ := dominio.NuevaSesion(huella, persona.ID, ahora, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour})
	if err := s.Crear(ctx, ses); err != nil {
		t.Fatal(err)
	}
	inicio := make(chan struct{})
	liberar := make(chan struct{})
	fallos := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		fallos <- s.Modificar(ctx, huella, func(ctx context.Context, actual *dominio.Sesion) error {
			close(inicio)
			<-liberar
			actual.UltimaActividadEn = ahora.Add(time.Minute)
			return nil
		})
	}()
	<-inicio
	go func() { defer wg.Done(); fallos <- NuevasSesiones(otra).Eliminar(ctx, huella) }()
	close(liberar)
	wg.Wait()
	close(fallos)
	for err := range fallos {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Modificar(ctx, huella, func(context.Context, *dominio.Sesion) error { return nil }); !errors.Is(err, dominio.ErrSesionNoValida) {
		t.Fatal("sesión resucitada", err)
	}
}

type directorioPersistencia struct{ persona dominio.IdentidadDirectorio }

func (d directorioPersistencia) Autenticar(context.Context, string, string) (dominio.IdentidadDirectorio, error) {
	return d.persona, nil
}

type relojPersistencia struct{}

func (relojPersistencia) Ahora() time.Time { return time.Now().UTC() }

type azarPersistencia struct{}

func (azarPersistencia) Llenar(b []byte) error { _, err := rand.Read(b); return err }

type usuariosObservados struct {
	*Usuarios
	id string
}

func (u *usuariosObservados) RegistrarAcceso(ctx context.Context, i dominio.IdentidadDirectorio, ahora time.Time) (dominio.Usuario, error) {
	persona, err := u.Usuarios.RegistrarAcceso(ctx, i, ahora)
	u.id = persona.ID
	return persona, err
}

type sesionesRechazadas struct{ *Sesiones }

func (s sesionesRechazadas) Crear(context.Context, dominio.Sesion) error {
	return dominio.ErrPersistencia
}
func TestIntegracionEntradaRevierteUsuarioSiSesionFalla(t *testing.T) {
	b, ctx := baseIntegracion(t)
	existente := personaIntegracion(t, NuevosUsuarios(b), ctx)
	var guid dominio.GUID
	_, _ = rand.Read(guid[:])
	persona := dominio.IdentidadDirectorio{GUID: guid, CuentaAD: existente.CuentaAD, NombreMostrado: "Inés Moreno Castillo"}
	usuarios := &usuariosObservados{Usuarios: NuevosUsuarios(b)}
	limitador, _ := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 5, PorIP: 30, MaxClaves: 100})
	servicio, err := aplicacion.NuevoServicio(directorioPersistencia{persona}, usuarios, sesionesRechazadas{NuevasSesiones(b)}, relojPersistencia{}, azarPersistencia{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour}, limitador, b)
	if err != nil {
		t.Fatal(err)
	}
	var ip dominio.IP
	ip[15] = 1
	if _, err := servicio.Entrar(ctx, entrada.Inicio{Cuenta: persona.CuentaAD, Contrasena: "clave-ficticia", IP: ip}); !errors.Is(err, dominio.ErrPersistencia) {
		t.Fatal("entrada confirmada", err)
	}
	if usuarios.id == "" {
		t.Fatal("prueba no llegó al alta")
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuarios.id}, func(ctx context.Context) error {
		_, err := usuarios.Obtener(ctx, usuarios.id)
		if !errors.Is(err, dominio.ErrUsuarioNoDisponible) {
			t.Fatal("usuario no revertido", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: existente.ID}, func(ctx context.Context) error {
		e, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var cuenta string
		if err := e.QueryRow(ctx, `SELECT cuenta_ad FROM identidad.usuario WHERE id=$1`, existente.ID).Scan(&cuenta); err != nil || cuenta != existente.CuentaAD {
			t.Fatal("login anterior no restaurado al revertir", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

}

// La alteración de la función afecta exclusivamente al contenedor desechable
// creado por ensayar-identidad.sh; se restaura antes de salir de la prueba.
func TestIntegracionAuditoriaFallidaBloqueaDatosYCambios(t *testing.T) {
	b, ctx := baseIntegracion(t)
	contenedor := os.Getenv("ETF_POSTGRES_CONTENEDOR")
	if contenedor == "" {
		t.Skip("ETF_POSTGRES_CONTENEDOR para inyección de fallo local")
	}
	if !strings.HasPrefix(contenedor, "etf-f009b-") {
		t.Fatal("contenedor de prueba no autorizado")
	}
	u := NuevosUsuarios(b)
	s := NuevasSesiones(b)
	persona := personaIntegracion(t, u, ctx)
	var bytes [32]byte
	_, _ = rand.Read(bytes[:])
	identificador := base64.RawURLEncoding.EncodeToString(bytes[:])
	huella, _ := dominio.HuellaIdentificador(identificador)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ses, _ := dominio.NuevaSesion(huella, persona.ID, ahora, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour})
	if err := s.Crear(ctx, ses); err != nil {
		t.Fatal(err)
	}
	limitador, _ := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 5, PorIP: 30, MaxClaves: 100})
	servicio, err := aplicacion.NuevoServicio(directorioPersistencia{persona.IdentidadDirectorio}, u, s, relojPersistencia{}, azarPersistencia{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour}, limitador, b)
	if err != nil {
		t.Fatal(err)
	}
	fuente, err := exec.Command("git", "show", os.Getenv("ETF_AUDITORIA_REF")+":migraciones/auditoria/0001_cadena.sql").Output()
	if err != nil {
		t.Fatal(err)
	}
	texto := string(fuente)
	inicio := strings.Index(texto, "CREATE FUNCTION auditoria.registrar(")
	fin := strings.Index(texto, "CREATE FUNCTION auditoria.exigir_administracion(")
	if inicio < 0 || fin < inicio {
		t.Fatal("función de auditoría no localizada")
	}
	original := strings.Replace(texto[inicio:fin], "CREATE FUNCTION auditoria.registrar(", "CREATE OR REPLACE FUNCTION auditoria.registrar(", 1)
	sql := func(guion string) {
		t.Helper()
		limite, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		cmd := exec.CommandContext(limite, "docker", "exec", "-i", contenedor, "psql", "-X", "-U", "etf_instalador", "-d", "etf", "-v", "ON_ERROR_STOP=1")
		cmd.Stdin = strings.NewReader(guion)
		if _, err := cmd.CombinedOutput(); err != nil {
			t.Fatal("inyección de fallo local", err)
		}
	}
	sql(`CREATE OR REPLACE FUNCTION auditoria.registrar(p_ocurrido_en timestamptz,p_accion text,p_entidad text,p_entidad_id uuid,p_expediente_id uuid,p_resultado text)
 RETURNS auditoria.evento LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,auditoria AS $$ BEGIN RAISE EXCEPTION 'auditoria.prueba_fallo'; END $$;`)
	defer sql(original)
	if acceso, err := servicio.Consultar(ctx, identificador); !errors.Is(err, dominio.ErrPersistencia) || acceso.Usuario.ID != "" || acceso.Estado != "" {
		t.Fatal("datos entregados sin auditoría", err)
	}
	if err := servicio.Salir(ctx, identificador); !errors.Is(err, dominio.ErrPersistencia) {
		t.Fatal("cierre confirmado sin auditoría", err)
	}
	var ip dominio.IP
	ip[15] = 1
	if abierta, err := servicio.Entrar(ctx, entrada.Inicio{Cuenta: persona.CuentaAD, Contrasena: "clave-ficticia", IP: ip}); !errors.Is(err, dominio.ErrPersistencia) || abierta.Identificador != "" {
		t.Fatal("entrada confirmada sin auditoría", err)
	}
	sql(original)
	if err := s.Modificar(ctx, huella, func(_ context.Context, actual *dominio.Sesion) error {
		if !actual.UltimaActividadEn.Equal(ahora) {
			t.Fatal("actividad no revertida")
		}
		return nil
	}); err != nil {
		t.Fatal("sesión eliminada a pesar del fallo", err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: persona.ID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		e, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var integra bool
		if err := e.QueryRow(ctx, `SELECT auditoria.verificar(1,9223372036854775807) IS NULL`).Scan(&integra); err != nil {
			return err
		}
		if !integra {
			t.Fatal("cadena de auditoría alterada")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
