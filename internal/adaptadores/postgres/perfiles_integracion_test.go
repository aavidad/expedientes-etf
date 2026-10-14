package postgres

import (
	"context"
	"crypto/rand"
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

func sqlPerfiles(t *testing.T, s string) string {
	t.Helper()
	c := os.Getenv("ETF_POSTGRES_CONTENEDOR")
	if c == "" {
		t.Skip("ETF_POSTGRES_CONTENEDOR")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", c, "psql", "-X", "-qAt", "-U", "etf_instalador", "-d", "etf", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SQL local: %v %s", err, out)
	}
	return strings.TrimSpace(string(out))
}
func actorPerfiles(u dominio.Usuario) entrada.Acceso {
	return entrada.Acceso{Usuario: u, Estado: dominio.Activo}
}
func TestIntegracionPerfilesAdministradorUnicoRetiradaYRLS(t *testing.T) {
	b, ctx := baseIntegracion(t)
	usuarios := NuevosUsuarios(b)
	var guid dominio.GUID
	_, _ = rand.Read(guid[:])
	cuenta := fmt.Sprintf("marta.%x", guid[:4])
	inicial, err := NuevosUsuariosConAdministrador(b, cuenta)
	if err != nil {
		t.Fatal(err)
	}
	sqlPerfiles(t, `CREATE FUNCTION identidad.fallo_auditoria_prueba() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.resultado='ok' AND NEW.accion IN ('identidad.administrador_inicial','identidad.perfiles_cambiar','identidad.usuarios_listar') THEN RAISE EXCEPTION 'auditoria.prueba_fallo'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER fallo_perfiles BEFORE INSERT ON auditoria.evento FOR EACH ROW EXECUTE FUNCTION identidad.fallo_auditoria_prueba();`)
	limpiarFallo := func() {
		sqlPerfiles(t, `DROP TRIGGER IF EXISTS fallo_perfiles ON auditoria.evento; DROP FUNCTION IF EXISTS identidad.fallo_auditoria_prueba();`)
	}
	defer limpiarFallo()
	if _, fallo := inicial.RegistrarAcceso(ctx, dominio.IdentidadDirectorio{GUID: guid, CuentaAD: cuenta, NombreMostrado: "Marta López Rivas"}, time.Now().UTC()); fallo == nil {
		t.Fatal("alta inicial sin auditoría")
	}
	if sqlPerfiles(t, `SELECT count(*) FROM identidad.alta_administrador_inicial;`) != "0" {
		t.Fatal("marca inicial sin auditoría")
	}
	limpiarFallo()
	identidadInicial := dominio.IdentidadDirectorio{GUID: guid, CuentaAD: cuenta, NombreMostrado: "Marta López Rivas"}
	rechazo := errors.New("prueba.revertir_alta_inicial")
	if fallo := b.EjecutarIdentidad(ctx, func(ctx context.Context) error {
		if _, e := inicial.RegistrarAcceso(ctx, identidadInicial, time.Now().UTC()); e != nil {
			return e
		}
		return rechazo
	}); !errors.Is(fallo, rechazo) {
		t.Fatal("reversión inicial", fallo)
	}
	if sqlPerfiles(t, `SELECT count(*) FROM identidad.alta_administrador_inicial;`) != "0" {
		t.Fatal("marca confirmada sin operación")
	}
	admin, err := inicial.RegistrarAcceso(ctx, dominio.IdentidadDirectorio{GUID: guid, CuentaAD: cuenta, NombreMostrado: "Marta López Rivas"}, time.Now().UTC())
	if err != nil || !admin.Permite(dominio.Administracion) {
		t.Fatal("alta inicial", err, admin)
	}
	b2, fallo := Nueva(ctx, os.Getenv)
	if fallo != nil {
		t.Fatal(fallo)
	}
	defer b2.Cerrar()
	inicial2, fallo := NuevosUsuariosConAdministrador(b2, cuenta)
	if fallo != nil {
		t.Fatal(fallo)
	}
	var concurrencia sync.WaitGroup
	fallos := make(chan error, 8)
	for i := 0; i < 8; i++ {
		concurrencia.Add(1)
		go func(i int) {
			defer concurrencia.Done()
			repo := inicial
			if i%2 == 0 {
				repo = inicial2
			}
			u, e := repo.RegistrarAcceso(ctx, identidadInicial, time.Now().UTC())
			if e == nil && !u.Permite(dominio.Administracion) {
				e = errors.New("prueba.administrador_ausente")
			}
			fallos <- e
		}(i)
	}
	concurrencia.Wait()
	close(fallos)
	for e := range fallos {
		if e != nil {
			t.Fatal("alta concurrente", e)
		}
	}
	s := aplicacion.NuevaAdministracionUsuarios(&Perfiles{}, b, NuevoRegistroAuditoria(b), relojPersistencia{})
	lista, err := s.Listar(ctx, actorPerfiles(admin))
	if err != nil {
		t.Fatal(err)
	}
	version := func(id string) int64 {
		t.Helper()
		l, e := s.Listar(ctx, actorPerfiles(admin))
		if e != nil {
			t.Fatal(e)
		}
		for _, u := range l {
			if u.ID == id {
				return u.Version
			}
		}
		t.Fatal("usuario no listado")
		return 0
	}
	if len(lista) == 0 {
		t.Fatal("lista vacía")
	}
	pendiente := personaIntegracion(t, usuarios, ctx)
	otro := personaIntegracion(t, usuarios, ctx)
	equipos := []string{}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: admin.ID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		e, _ := DesdeContexto(ctx)
		rows, er := e.Query(ctx, `SELECT id::text FROM catalogo.equipo ORDER BY codigo LIMIT 2`)
		if er != nil {
			return er
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if er = rows.Scan(&id); er != nil {
				return er
			}
			equipos = append(equipos, id)
		}
		return rows.Err()
	})
	if err != nil || len(equipos) != 2 {
		t.Fatal("equipos", err)
	}
	ahora := time.Now().UTC()
	hasta := ahora.Add(time.Hour)
	cambio := dominio.CambioPerfiles{Version: version(pendiente.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.ProfesionalETF, Desde: ahora, Hasta: &hasta}, {Perfil: dominio.Consulta, Desde: ahora}}, Equipo: &dominio.Adscripcion{EquipoID: equipos[0], Puesto: "ts", Desde: ahora, Hasta: &hasta}}
	sqlPerfiles(t, `CREATE FUNCTION identidad.fallo_auditoria_prueba() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.resultado='ok' AND NEW.accion IN ('identidad.perfiles_cambiar','identidad.usuarios_listar') THEN RAISE EXCEPTION 'auditoria.prueba_fallo'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER fallo_perfiles BEFORE INSERT ON auditoria.evento FOR EACH ROW EXECUTE FUNCTION identidad.fallo_auditoria_prueba();`)
	if l, fallo := s.Listar(ctx, actorPerfiles(admin)); fallo != dominio.ErrPersistencia || len(l) != 0 {
		t.Fatal("lista entregada sin auditoría", fallo)
	}
	if _, fallo := s.Cambiar(ctx, actorPerfiles(admin), pendiente.ID, cambio); fallo != dominio.ErrPersistencia {
		t.Fatal("perfiles confirmados sin auditoría", fallo)
	}
	if sqlPerfiles(t, fmt.Sprintf(`SELECT estado FROM identidad.usuario WHERE id='%s';`, pendiente.ID)) != "pendiente_alta" {
		t.Fatal("escritura parcial")
	}
	if sqlPerfiles(t, fmt.Sprintf(`SELECT count(*) FROM identidad.usuario_perfil WHERE usuario_id='%s';`, pendiente.ID)) != "0" {
		t.Fatal("perfil persistido sin auditoría")
	}
	limpiarFallo()
	guardado, err := s.Cambiar(ctx, actorPerfiles(admin), pendiente.ID, cambio)
	if err != nil {
		t.Fatal("asignación", err)
	}
	if guardado.Estado != dominio.Activo || guardado.Equipo.EquipoID != equipos[0] {
		t.Fatal(guardado)
	}
	if _, err = s.Cambiar(ctx, actorPerfiles(admin), pendiente.ID, cambio); !errors.Is(err, dominio.ErrConflicto) {
		t.Fatal("versión obsoleta", err)
	}
	// Cambiar equipo cierra la adscripción anterior y no concede dos ETF.
	cambio.Version = guardado.Version
	cambio.Perfiles = guardado.Perfiles
	cambio.Equipo = guardado.Equipo
	cambio.Equipo.EquipoID = equipos[1]
	guardado, err = s.Cambiar(ctx, actorPerfiles(admin), pendiente.ID, cambio)
	if err != nil {
		t.Fatal("traslado", err)
	}
	if sqlPerfiles(t, fmt.Sprintf(`SELECT count(*) FROM identidad.usuario_equipo WHERE usuario_id='%s' AND desde<=clock_timestamp() AND (hasta IS NULL OR hasta>clock_timestamp());`, pendiente.ID)) != "1" {
		t.Fatal("dos equipos activos")
	}
	// FK y restricciones temporales se prueban con privilegios de instalación,
	// separados del rol real de aplicación que carece de INSERT directo.
	comprobacion := fmt.Sprintf(`DO $$ BEGIN
  BEGIN INSERT INTO identidad.usuario_equipo VALUES ('%s','%s','ts',clock_timestamp(),NULL,'%s'); RAISE EXCEPTION 'prueba.solapamiento_admitido'; EXCEPTION WHEN exclusion_violation THEN NULL; END;
  BEGIN INSERT INTO identidad.usuario_equipo VALUES ('%s','ffffffff-ffff-4fff-8fff-ffffffffffff','ts',clock_timestamp()+interval '2 hours',NULL,'%s'); RAISE EXCEPTION 'prueba.fk_admitida'; EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 END $$;`, pendiente.ID, equipos[0], admin.ID, otro.ID, admin.ID)
	sqlPerfiles(t, comprobacion)
	// Tabla sintética: aún no existe el módulo de expedientes. Comprueba contexto
	// RLS real de dos equipos sin introducir tablas de negocio en la migración.
	sqlPerfiles(t, fmt.Sprintf(`CREATE TABLE identidad.prueba_aislamiento(id uuid PRIMARY KEY,equipo_id uuid NOT NULL);
 ALTER TABLE identidad.prueba_aislamiento ENABLE ROW LEVEL SECURITY; ALTER TABLE identidad.prueba_aislamiento FORCE ROW LEVEL SECURITY;
 CREATE POLICY equipo ON identidad.prueba_aislamiento TO etf_app USING
 (current_setting('app.usuario_id',true)<>'' AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK (equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
 GRANT SELECT,UPDATE ON identidad.prueba_aislamiento TO etf_app;
 INSERT INTO identidad.prueba_aislamiento VALUES(gen_random_uuid(),'%s'),(gen_random_uuid(),'%s');`, equipos[0], equipos[1]))
	defer sqlPerfiles(t, `DROP TABLE identidad.prueba_aislamiento;`)
	verificar := func(actor plataforma.Identidad, n int) {
		t.Helper()
		err := b.Ejecutar(ctx, actor, func(ctx context.Context) error {
			e, _ := DesdeContexto(ctx)
			var cuenta int
			if er := e.QueryRow(ctx, `SELECT count(*) FROM identidad.prueba_aislamiento`).Scan(&cuenta); er != nil {
				return er
			}
			if cuenta != n {
				t.Fatalf("RLS: %v recibió %d", actor.Perfiles, cuenta)
			}
			tag, er := e.Exec(ctx, `UPDATE identidad.prueba_aislamiento SET equipo_id=equipo_id WHERE equipo_id=$1`, equipos[0])
			if er != nil {
				return er
			}
			if actor.EquipoID == equipos[1] && tag.RowsAffected() != 0 {
				t.Fatal("escritura horizontal")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	verificar(plataforma.Identidad{}, 0)
	verificar(plataforma.Identidad{UsuarioID: admin.ID, Perfiles: []string{"administracion"}}, 0)
	verificar(plataforma.Identidad{UsuarioID: admin.ID, Perfiles: []string{"consulta"}}, 0)
	verificar(plataforma.Identidad{UsuarioID: pendiente.ID, Perfiles: []string{"profesional_etf"}, EquipoID: equipos[1]}, 1)
	limitador, _ := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 100, MaxClaves: 1000})
	servicio, err := aplicacion.NuevoServicio(directorioPersistencia{pendiente.IdentidadDirectorio}, usuarios, NuevasSesiones(b), relojPersistencia{}, azarPersistencia{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour}, limitador, b)
	if err != nil {
		t.Fatal(err)
	}
	var ip dominio.IP
	ip[15] = 1
	abierta, err := servicio.Entrar(ctx, entrada.Inicio{Cuenta: pendiente.CuentaAD, Contrasena: "clave-ficticia", IP: ip})
	if err != nil {
		t.Fatal(err)
	}
	acceso, err := servicio.Consultar(ctx, abierta.Identificador)
	if err != nil || acceso.Usuario.EquipoID != equipos[1] {
		t.Fatal("sesión sin permisos vigentes", err)
	}
	if _, err = s.Listar(ctx, acceso); !errors.Is(err, dominio.ErrAccesoDenegado) {
		t.Fatal("escalada vertical", err)
	}
	// Cuenta propia y perfil administrativo falsificado no superan la RLS.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: pendiente.ID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error { _, e := (&Perfiles{}).Listar(ctx); return e })
	if !errors.Is(err, dominio.ErrAccesoDenegado) {
		t.Fatal("administración falsificada", err)
	}
	cambio = dominio.CambioPerfiles{Version: version(pendiente.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Consulta, Desde: time.Now().UTC()}}}
	if _, err = s.Cambiar(ctx, actorPerfiles(admin), pendiente.ID, cambio); err != nil {
		t.Fatal(err)
	}
	acceso, err = servicio.Consultar(ctx, abierta.Identificador)
	if err != nil || acceso.Usuario.Permite(dominio.ProfesionalETF) || acceso.Usuario.EquipoID != "" {
		t.Fatal("perfil retirado sigue vigente", err)
	}
	// Las concesiones futuras y caducadas desaparecen del permiso efectivo.
	futuro := personaIntegracion(t, usuarios, ctx)
	desdeFutura := time.Now().UTC().Add(time.Hour)
	if _, err = s.Cambiar(ctx, actorPerfiles(admin), futuro.ID, dominio.CambioPerfiles{Version: version(futuro.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Consulta, Desde: desdeFutura}}}); err != nil {
		t.Fatal(err)
	}
	leerPropio := func(persona dominio.Usuario) dominio.Usuario {
		t.Helper()
		var u dominio.Usuario
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: persona.ID}, func(ctx context.Context) error { var e error; u, e = usuarios.Obtener(ctx, persona.ID); return e })
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	// Editar el mismo inicio futuro conserva historia y devuelve sólo la nueva
	// concesión, aunque la anterior se haya cancelado con intervalo vacío.
	finFutura := desdeFutura.Add(time.Hour)
	futuroEditado, fallo := s.Cambiar(ctx, actorPerfiles(admin), futuro.ID, dominio.CambioPerfiles{Version: version(futuro.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Consulta, Desde: desdeFutura, Hasta: &finFutura}}})
	if fallo != nil || len(futuroEditado.Perfiles) != 1 {
		t.Fatal("edición futura", fallo, futuroEditado)
	}
	profesionalFuturo := personaIntegracion(t, usuarios, ctx)
	asignacionFutura := dominio.CambioPerfiles{Version: version(profesionalFuturo.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.ProfesionalETF, Desde: desdeFutura}}, Equipo: &dominio.Adscripcion{EquipoID: equipos[0], Puesto: "es", Desde: desdeFutura}}
	futuroGuardado, fallo := s.Cambiar(ctx, actorPerfiles(admin), profesionalFuturo.ID, asignacionFutura)
	if fallo != nil {
		t.Fatal(fallo)
	}
	asignacionFutura.Version = futuroGuardado.Version
	asignacionFutura.Equipo.EquipoID = equipos[1]
	futuroGuardado, fallo = s.Cambiar(ctx, actorPerfiles(admin), profesionalFuturo.ID, asignacionFutura)
	if fallo != nil || len(futuroGuardado.Perfiles) != 1 || futuroGuardado.Equipo.EquipoID != equipos[1] {
		t.Fatal("sustitución de equipo futuro", fallo)
	}
	if _, fallo = s.Cambiar(ctx, actorPerfiles(admin), profesionalFuturo.ID, dominio.CambioPerfiles{Version: futuroGuardado.Version, Estado: dominio.Activo}); fallo != nil {
		t.Fatal("retirada futura", fallo)
	}
	l, fallo := s.Listar(ctx, actorPerfiles(admin))
	if fallo != nil {
		t.Fatal(fallo)
	}
	for _, u := range l {
		if u.ID == profesionalFuturo.ID && (len(u.Perfiles) != 0 || u.Equipo != nil || u.Estado != dominio.PendienteAlta) {
			t.Fatal("intervalo vacío listado", u)
		}
	}
	if len(leerPropio(futuro).Perfiles) != 0 {
		t.Fatal("perfil futuro vigente antes de tiempo")
	}
	temporal := personaIntegracion(t, usuarios, ctx)
	ahoraTemporal := time.Now().UTC()
	finTemporal := ahoraTemporal.Add(300 * time.Millisecond)
	if _, err = s.Cambiar(ctx, actorPerfiles(admin), temporal.ID, dominio.CambioPerfiles{Version: version(temporal.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Consulta, Desde: ahoraTemporal, Hasta: &finTemporal}}}); err != nil {
		t.Fatal(err)
	}
	if !leerPropio(temporal).Permite(dominio.Consulta) {
		t.Fatal("concesión vigente ausente")
	}
	time.Sleep(time.Until(finTemporal) + 10*time.Millisecond)
	if leerPropio(temporal).Permite(dominio.Consulta) {
		t.Fatal("concesión caducada sigue vigente")
	}
	// Una concesión ya iniciada que espera el bloqueo debe revalidar el permiso
	// después de una retirada concurrente confirmada.
	victima := personaIntegracion(t, usuarios, ctx)
	v, fallo := s.Cambiar(ctx, actorPerfiles(admin), victima.ID, dominio.CambioPerfiles{Version: version(victima.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Administracion, Desde: time.Now().UTC()}}})
	if fallo != nil {
		t.Fatal(fallo)
	}
	victima = leerPropio(victima)
	versionDestino := version(otro.ID)
	bloqueado := make(chan struct{})
	continuar := make(chan struct{})
	revocado := make(chan error, 1)
	go func() {
		revocado <- b2.Ejecutar(ctx, plataforma.Identidad{UsuarioID: admin.ID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
			e, _ := DesdeContexto(ctx)
			if _, er := e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0))`); er != nil {
				return er
			}
			close(bloqueado)
			<-continuar
			_, er := e.Exec(ctx, `SELECT identidad.cambiar_perfiles($1,$2,'activo','[]'::jsonb,'null'::jsonb)`, victima.ID, v.Version)
			return er
		})
	}()
	<-bloqueado
	intentado := make(chan error, 1)
	go func() {
		_, e := s.Cambiar(ctx, actorPerfiles(victima), otro.ID, dominio.CambioPerfiles{Version: versionDestino, Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Administracion, Desde: time.Now().UTC()}}})
		intentado <- e
	}()
	limite := time.Now().Add(2 * time.Second)
	for sqlPerfiles(t, `SELECT count(*) FROM pg_stat_activity WHERE usename='etf_app' AND wait_event_type='Lock' AND query LIKE 'SELECT identidad.cambiar_perfiles%';`) == "0" {
		if time.Now().After(limite) {
			close(continuar)
			t.Fatal("petición no esperó el bloqueo")
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(continuar)
	if e := <-revocado; e != nil {
		t.Fatal("retirada concurrente", e)
	}
	if e := <-intentado; !errors.Is(e, dominio.ErrAccesoDenegado) {
		t.Fatal("administrador retirado escribió tras esperar", e)
	}
	// Designar segundo administrador antes de retirar al primero permite comprobar
	// que ni entrar de nuevo ni reutilizar su login conceden el alta inicial.
	_, err = s.Cambiar(ctx, actorPerfiles(admin), otro.ID, dominio.CambioPerfiles{Version: version(otro.ID), Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Administracion, Desde: time.Now().UTC()}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Cambiar(ctx, actorPerfiles(admin), admin.ID, dominio.CambioPerfiles{Version: version(admin.ID), Estado: dominio.Activo}); err != nil {
		t.Fatal("retirada inicial", err)
	}
	admin, err = inicial.RegistrarAcceso(ctx, admin.IdentidadDirectorio, time.Now().UTC())
	if err != nil || len(admin.Perfiles) != 0 {
		t.Fatal("administrador reconcedido", err)
	}
	reemplazo := admin.IdentidadDirectorio
	reemplazo.GUID[0] ^= 1
	nuevo, err := inicial.RegistrarAcceso(ctx, reemplazo, time.Now().UTC())
	if err != nil || len(nuevo.Perfiles) != 0 || nuevo.ID == admin.ID {
		t.Fatal("login reutilizado privilegiado", err)
	}
	if sqlPerfiles(t, `SELECT count(*) FROM identidad.alta_administrador_inicial;`) != "1" {
		t.Fatal("alta inicial duplicada")
	}
	if sqlPerfiles(t, `SELECT count(*) FROM auditoria.evento WHERE accion='identidad.administrador_inicial';`) != "1" {
		t.Fatal("auditoría inicial duplicada")
	}
	t.Log("PERFILES-OK: combinación, vigencia, FK, equipo único, RLS dos equipos, denegación horizontal/vertical, retirada y primer administrador sin reconcesión")
}
