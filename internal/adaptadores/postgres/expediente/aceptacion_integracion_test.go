package expediente_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func prepararAceptacion(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, dominio.Aceptacion) {
	t.Helper()
	b, ctx, d, a := preparar(t)
	d.Aceptaciones = &adaptador.Repositorio{}
	d.Numeracion = &catalogo.Repositorio{}
	p := dominio.Aceptacion{Version: 1, FechaActa: "2026-02-03", Subprograma: "RIESGO_SOCIAL", ProfesionalPuesto: "TS", Descripcion: "La familia necesita apoyo para organizar los cuidados y mantener la asistencia escolar."}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return e.QueryRow(ctx, `SELECT m.id::text,c.id::text,q.id::text FROM catalogo.municipio m,catalogo.centro_ss c,catalogo.equipo q
   WHERE m.ine='18001' AND c.codigo='ALHAMA' AND q.codigo='08'`).Scan(&p.MunicipioID, &p.CentroID, &p.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var existe bool
		if e = tx.QueryRow(ctx, `SELECT to_regclass('expediente.plazo_carga') IS NOT NULL`).Scan(&existe); e != nil {
			return e
		}
		if existe {
			_, e = tx.Exec(ctx, `UPDATE expediente.plazo_carga SET fecha_cierre='2026-12-31',version=version+1 WHERE id AND fecha_cierre IS NULL`)
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	return b, ctx, d, a, p
}
func conMenores(p dominio.Aceptacion, d dominio.Derivacion) dominio.Aceptacion {
	p.Menores = nil
	for _, c := range d.Datos.Extraccion.Campos {
		if c.Codigo == "menores.persona.nombre" {
			p.Menores = append(p.Menores, dominio.SexoMenor{Orden: len(p.Menores) + 1, Sexo: "M"})
		}
	}
	return p
}
func contador(t *testing.T, b *postgres.Base, ctx context.Context, a dominio.Actor, p dominio.Aceptacion) int {
	t.Helper()
	n := 0
	fecha, _ := p.Fecha()
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return e.QueryRow(ctx, `SELECT coalesce((SELECT ultimo FROM expediente.contador_expediente WHERE equipo_id=$1 AND anio=$2),0)`, p.EquipoID, fecha.Year()).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func TestAceptacion20ConcurrentesSinHuecosNiRepetidos(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	antes := contador(t, b, ctx, a, p)
	ds := make([]dominio.Derivacion, 20)
	for i := range ds {
		ds[i] = subir(t, ctx, deps, a)
	}
	resultados := make([]dominio.ResultadoAceptacion, 20)
	fallos := make([]error, 20)
	var wg sync.WaitGroup
	salida := make(chan struct{})
	for i := range ds {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-salida
			resultados[i], fallos[i] = aplicacion.Nuevo(deps).Aceptar(ctx, a, ds[i].ID, fmt.Sprintf("lote-%s", ds[i].ID), conMenores(p, ds[i]))
		}(i)
	}
	close(salida)
	wg.Wait()
	numeros := map[string]bool{}
	correlativos := []int{}
	for i, r := range resultados {
		if fallos[i] != nil {
			t.Fatal(i, fallos[i])
		}
		if numeros[r.Numero] {
			t.Fatal("repetido", r.Numero)
		}
		numeros[r.Numero] = true
		correlativos = append(correlativos, r.Correlativo)
		if r.Numero != fmt.Sprintf("08-%02d-26", r.Correlativo) || r.Estado != "aceptada" || r.Version != 2 || r.FechaActa != "2026-02-03" {
			t.Fatal(r)
		}
	}
	sort.Ints(correlativos)
	for i, n := range correlativos {
		if n != antes+i+1 {
			t.Fatal("hueco", correlativos)
		}
	}
	if contador(t, b, ctx, a, p) != antes+20 {
		t.Fatal("contador distinto")
	}
	t.Log("20 aceptaciones simultáneas: 20 números únicos, correlativos consecutivos y año del Acta")
}
func TestAceptacionIdempotenciaConcurrenteYCambioPeticion(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	antes := contador(t, b, ctx, a, p)
	resultados := make([]dominio.ResultadoAceptacion, 20)
	fallos := make([]error, 20)
	var wg sync.WaitGroup
	salida := make(chan struct{})
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-salida
			resultados[i], fallos[i] = aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "aceptar-unica", p)
		}(i)
	}
	close(salida)
	wg.Wait()
	for i, r := range resultados {
		if fallos[i] != nil || r != resultados[0] {
			t.Fatal(i, fallos[i], r)
		}
	}
	otra := subir(t, ctx, deps, a)
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, otra.ID, "aceptar-unica", conMenores(p, otra)); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal("clave reutilizada en otra ruta", err)
	}
	p.Descripcion += " Se acuerda revisar los horarios."
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "aceptar-unica", p); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "otra", p); err == nil {
		t.Fatal("segunda aceptación")
	} else {
		var c dominio.Conflicto
		if !errors.As(err, &c) || c.VersionActual != 2 {
			t.Fatal(err)
		}
	}
	p.Version = 2
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "estado", p); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	if contador(t, b, ctx, a, p) != antes+1 {
		t.Fatal("idempotencia consume números")
	}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		for _, tabla := range []string{"episodio", "hito", "estado_cambio"} {
			var n int
			if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla+" WHERE expediente_id=$1", d.ID).Scan(&n); err != nil {
				return err
			}
			if n != 1 {
				t.Fatal(tabla, n)
			}
		}
		var fecha time.Time
		if err := e.QueryRow(ctx, `SELECT fecha FROM expediente.hito WHERE expediente_id=$1`, d.ID).Scan(&fecha); err != nil {
			return err
		}
		if fecha.Format("2006-01-02") != "2026-02-03" {
			t.Fatal(fecha)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r := postgres.NuevoRegistroAuditoria(b)
	exportacion, err := r.Exportar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ev := range exportacion.Eventos {
		if ev.ExpedienteID == d.ID && ev.Accion == "expediente.aceptar_derivacion" {
			n++
		}
	}
	if n != 1 {
		t.Fatal("auditoría duplicada", n)
	}
	t.Log("20 repeticiones simultáneas: un número, un hito, un episodio y un evento; reutilización distinta rechazada")
}
func TestAceptacionObligatoriosYAuditoriaRevierteContador(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	antes := contador(t, b, ctx, a, p)
	casos := []struct {
		nombre  string
		cambiar func(*dominio.Aceptacion)
		err     error
	}{
		{"puesto", func(p *dominio.Aceptacion) { p.ProfesionalPuesto = "" }, dominio.ErrProfesional},
		{"sexo", func(p *dominio.Aceptacion) { p.Menores[0].Sexo = "" }, dominio.ErrSexo},
		{"falta menor", func(p *dominio.Aceptacion) { p.Menores = p.Menores[1:] }, dominio.ErrSexo},
		{"menor repetido", func(p *dominio.Aceptacion) { p.Menores[1].Orden = 1 }, dominio.ErrSexo},
		{"fecha", func(p *dominio.Aceptacion) { p.FechaActa = "" }, dominio.ErrObligatorios},
		{"descripcion", func(p *dominio.Aceptacion) { p.Descripcion = " " }, dominio.ErrObligatorios},
		{"persona ajena", func(p *dominio.Aceptacion) { id := a.UsuarioID; p.ProfesionalID = &id }, dominio.ErrProfesional},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := p
			q.Menores = append([]dominio.SexoMenor(nil), p.Menores...)
			c.cambiar(&q)
			_, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "datos-incompletos", q)
			if !errors.Is(err, c.err) {
				t.Fatal(err)
			}
		})
	}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err := aplicacion.Nuevo(fallo).Aceptar(ctx, a, d.ID, "fallo-auditoria", p); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal(err)
	}
	if contador(t, b, ctx, a, p) != antes {
		t.Fatal("contador consumido por fallo")
	}
	leida, err := aplicacion.Nuevo(deps).Consultar(ctx, a, d.ID)
	if err != nil || !reflect.DeepEqual(leida, d) {
		t.Fatal("escritura parcial", err)
	}
	r, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "fallo-auditoria", p)
	if err != nil || r.Correlativo != antes+1 {
		t.Fatal(r, err)
	}
	t.Log("campos obligatorios y fallo de auditoría: ninguna aceptación parcial ni hueco")
}
func TestAceptacionRLSPorPerfilEquipoYLecturaAuditada(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "rls", p); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		perfil, equipo string
		visible        bool
	}{
		{"", "", false}, {"administracion", "", false}, {"consulta", "", false}, {"profesional_etf", "", false},
		{"profesional_etf", p.EquipoID, true}, {"profesional_etf", "00000000-0000-4000-8000-000000000012", false},
	} {
		i := plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: caso.equipo}
		if caso.perfil != "" {
			i.Perfiles = []string{caso.perfil}
		}
		err := b.Ejecutar(ctx, i, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"expediente", "lectura_anexo", "derivacion_datos", "episodio", "hito", "estado_cambio", "menor", "dato_estadistico_vigencia"} {
				columna := "expediente_id"
				if tabla == "expediente" {
					columna = "id"
				}
				var n int
				if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla+" WHERE "+columna+"=$1", d.ID).Scan(&n); err != nil {
					return err
				}
				if (n > 0) != caso.visible {
					t.Fatalf("perfil %s tabla %s visible %v", caso.perfil, tabla, n > 0)
				}
			}
			if caso.perfil != "" && caso.perfil != "coordinacion" {
				// La propuesta permite mover la versión técnica, pero el disparador
				// sigue denegando cambios de ficha. También es válido no alcanzar filas por RLS.
				if _, err := e.Exec(ctx, "SAVEPOINT escritura_denegada"); err != nil {
					return err
				}
				tag, err := e.Exec(ctx, `UPDATE expediente.expediente SET descripcion_situacion='Cambio no autorizado' WHERE id=$1`, d.ID)
				if err != nil {
					var denegada *pgconn.PgError
					if !errors.As(err, &denegada) || denegada.Code != "42501" {
						return err
					}
					if _, err := e.Exec(ctx, "ROLLBACK TO SAVEPOINT escritura_denegada"); err != nil {
						return err
					}
				} else if tag.RowsAffected() != 0 {
					t.Fatal("escritura ajena")
				}
				if _, err := e.Exec(ctx, "RELEASE SAVEPOINT escritura_denegada"); err != nil {
					return err
				}
			}
			return nil
		})
		if caso.perfil == "profesional_etf" && caso.visible {
			var falloSQL *pgconn.PgError
			if !errors.As(err, &falloSQL) || falloSQL.Message != "error.sin_permiso" {
				t.Fatalf("corrección de aceptación sin permiso: %v", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	profesional := dominio.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	leida, err := aplicacion.Nuevo(deps).Consultar(ctx, profesional, d.ID)
	if err != nil || leida.Aceptacion == nil || len(leida.Aceptacion.Menores) != len(p.Menores) || leida.Aceptacion.Descripcion != p.Descripcion {
		t.Fatal("ficha", err)
	}
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, profesional, d.ID, "sin-permiso", p); !errors.Is(err, dominio.ErrAcceso) {
		t.Fatal(err)
	}
	deps.RegistroTransaccional = registroFallido{}
	if _, err := aplicacion.Nuevo(deps).Consultar(ctx, profesional, d.ID); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal("lectura sin auditoría", err)
	}
	t.Log("RLS: sólo Coordinación y equipo propio leen; profesionales no aceptan; lectura falla sin auditoría")
}

func TestNumeracionVigenteRealInmutableYFormatoConfigurado(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	cfg, err := pgx.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Host = os.Getenv("ETF_BD_HOST")
	cfg.Port = 5432
	cfg.Database = os.Getenv("ETF_BD_NOMBRE")
	cfg.User = "etf_migrador"
	cfg.Password = os.Getenv("ETF_MIGRADOR_CONTRASENA")
	cfg.TLSConfig = nil
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal("migrador local no disponible")
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE etf_propietario`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
 WITH v AS (INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES('numeracion',2,'2027-01-01','numeracion.d1',$1) RETURNING id)
 INSERT INTO catalogo.regla_numeracion SELECT id,ARRAY['anio','equipo','correlativo'],2,3,4,'/' FROM v`, a.UsuarioID)
	if err != nil {
		t.Fatal("publicación de numeración", err)
	}
	if _, err = tx.Exec(ctx, `SELECT catalogo.publicar(id,$1) FROM catalogo.version WHERE ambito='numeracion' AND numero=2`, a.UsuarioID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, `SET ROLE etf_propietario; UPDATE catalogo.regla_numeracion SET separador='.'`); err == nil {
		t.Fatal("regla publicada modificable")
	}
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	p.FechaActa = "2027-01-01"
	r, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "regla-2027", p)
	if err != nil || r.Numero != "2027/08/001" {
		t.Fatal(r, err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		c := &catalogo.Repositorio{}
		vieja, err := c.ReglaVigente(ctx, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			return err
		}
		n, err := vieja.Formatear("08", 1, 2026)
		if err != nil || n != "08-01-26" {
			t.Fatal(n, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("vigencia [desde,hasta), versión publicada inmutable y formato 2027/08/001 leído del catálogo")
}

func TestAceptacionContadorAgotadoYEstadoConcurrente(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		if err := e.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo='07'`).Scan(&p.EquipoID); err != nil {
			return err
		}
		_, err := e.Exec(ctx, `INSERT INTO expediente.contador_expediente(equipo_id,anio,ultimo) VALUES($1,2026,99) ON CONFLICT(equipo_id,anio) DO UPDATE SET ultimo=99`, p.EquipoID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, "agotado", p); !errors.Is(err, dominio.ErrNumeracion) {
		t.Fatal(err)
	}
	if n := contador(t, b, ctx, a, p); n != 99 {
		t.Fatal(n)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		return e.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo='08'`).Scan(&p.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	antes := contador(t, b, ctx, a, p)
	var wg sync.WaitGroup
	salida := make(chan struct{})
	fallos := make([]error, 2)
	for i := range fallos {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-salida
			_, fallos[i] = aplicacion.Nuevo(deps).Aceptar(ctx, a, d.ID, fmt.Sprintf("competicion-%d", i), p)
		}(i)
	}
	close(salida)
	wg.Wait()
	exitos := 0
	conflictos := 0
	for _, err := range fallos {
		if err == nil {
			exitos++
		} else {
			var c dominio.Conflicto
			if !errors.As(err, &c) || c.VersionActual != 2 {
				t.Fatal(err)
			}
			conflictos++
		}
	}
	if exitos != 1 || conflictos != 1 || contador(t, b, ctx, a, p) != antes+1 {
		t.Fatal(exitos, conflictos)
	}
	t.Log("límite 99 sin consumo; dos claves sobre una derivación: una aceptación y un conflicto con versión 2")
}
