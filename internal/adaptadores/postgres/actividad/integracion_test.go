package actividad_test

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogoRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/actividad"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"testing"
	"time"
)

type auditoriaObservada struct {
	registro interface {
		Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error)
	}
	ultimo auditoria.Datos
}

func (a *auditoriaObservada) Confirmar(ctx context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	a.ultimo = d
	return a.registro.Confirmar(ctx, d)
}

type auditoriaFallida struct{}

func (auditoriaFallida) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.no_disponible")
}

func TestIntegracionActividadRLSRecuentosAuditoria(t *testing.T) {
	if os.Getenv("ETF_ACTIVIDAD_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_ACTIVIDAD_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	audit := &auditoriaObservada{registro: postgres.NuevoRegistroAuditoriaTransaccional(b)}
	s := app.Nuevo(&repo.Repositorio{}, b, audit, time.Now, app.Opciones{Equipos: repo.Equipos{Catalogo: &catalogoRepo.Repositorio{}, Ahora: time.Now}})
	coordinacion := dominio.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", Perfiles: []string{"coordinacion"}}
	var equipos []string
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coordinacion.UsuarioID, Perfiles: coordinacion.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		filas, err := tx.Query(ctx, `SELECT id::text FROM catalogo.equipo ORDER BY codigo LIMIT 2`)
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var id string
			if err := filas.Scan(&id); err != nil {
				return err
			}
			equipos = append(equipos, id)
		}
		return filas.Err()
	})
	if err != nil || len(equipos) != 2 {
		t.Fatalf("equipos: %v %v", equipos, err)
	}
	a := dominio.Actor{UsuarioID: coordinacion.UsuarioID, EquipoID: equipos[0], Perfiles: []string{"profesional_etf"}}
	bActor := dominio.Actor{UsuarioID: "22222222-2222-4222-8222-222222222222", EquipoID: equipos[1], Perfiles: []string{"profesional_etf"}}
	for _, actor := range []dominio.Actor{a, bActor, coordinacion} {
		lista, err := s.Equipos(ctx, actor)
		if err != nil {
			t.Fatal(err)
		}
		esperado := 1
		if actor.EquipoID == "" {
			esperado = 8
		}
		if len(lista) != esperado {
			t.Fatalf("equipos fuera del ámbito: %v", lista)
		}
		for _, e := range lista {
			if e.ClaveTexto == "" || (actor.EquipoID != "" && e.ID != actor.EquipoID) {
				t.Fatal("etiqueta o ámbito incorrecto", e)
			}
		}
		if audit.ultimo.Accion != "actividad.consultar_equipos" {
			t.Fatal("lectura de equipos sin auditar")
		}
	}
	if _, err := s.Equipos(ctx, dominio.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}); !errors.Is(err, dominio.ErrPermiso) {
		t.Fatal("Consulta accedió al catálogo", err)
	}
	equiposSinAuditoria := app.Nuevo(&repo.Repositorio{}, b, auditoriaFallida{}, time.Now, app.Opciones{Equipos: repo.Equipos{Catalogo: &catalogoRepo.Repositorio{}, Ahora: time.Now}})
	if lista, err := equiposSinAuditoria.Equipos(ctx, coordinacion); err == nil || lista != nil {
		t.Fatal("lectura entregada sin auditoría")
	}

	f := dominio.Filtro{Anio: 2026}
	crear := func(actor dominio.Actor, tipo dominio.Tipo, c dominio.Cambio) dominio.Registro {
		t.Helper()
		fila, err := s.Registrar(ctx, actor, tipo, c)
		if err != nil {
			t.Fatal(err)
		}
		return fila
	}
	one := crear(a, dominio.Comision, dominio.Cambio{Fecha: "2026-01-01", Observaciones: "Coordinación con el centro sanitario"})
	crear(bActor, dominio.Comision, dominio.Cambio{Fecha: "2026-12-31"})
	crear(a, dominio.Comision, dominio.Cambio{Fecha: "2025-12-31"})
	crear(a, dominio.Comision, dominio.Cambio{Fecha: "2027-01-01"})
	inf := crear(a, dominio.Informe, dominio.Cambio{Fecha: "2026-05-20", Organismo: "JUZGADOS"})
	crear(a, dominio.Informe, dominio.Cambio{Fecha: "2026-06-15", Organismo: "OTROS", OrganismoOtros: "Unidad de atención familiar de Loja"})
	crear(bActor, dominio.Informe, dominio.Cambio{Fecha: "2026-04-01", Organismo: "FISCALIA"})
	for _, actor := range []dominio.Actor{a, bActor} {
		filas, err := s.Consultar(ctx, actor, dominio.Comision, f)
		if err != nil || len(filas) != 1 || filas[0].EquipoID != actor.EquipoID {
			t.Fatalf("listado propio: %+v %v", filas, err)
		}
	}
	cifras, err := s.Recontar(ctx, coordinacion, f)
	if err != nil || cifras.Comisiones != 2 || cifras.Informes != 3 || cifras.PorOrganismo["OTROS"] != 1 || cifras.PorOrganismo["FISCALIA"] != 1 {
		t.Fatalf("provincial: %+v %v", cifras, err)
	}
	cifras, err = s.Recontar(ctx, a, f)
	if err != nil || cifras.Comisiones != 1 || cifras.Informes != 2 {
		t.Fatalf("equipo: %+v %v", cifras, err)
	}
	if _, err := s.Recontar(ctx, a, dominio.Filtro{Anio: 2026, EquipoID: bActor.EquipoID}); !errors.Is(err, dominio.ErrPermiso) {
		t.Fatal("filtro ajeno", err)
	}
	if _, err := s.Corregir(ctx, bActor, dominio.Comision, one.ID, dominio.Cambio{Fecha: "2026-01-02", VersionEsperada: 1}); !errors.Is(err, dominio.ErrNoEncontrado) {
		t.Fatal("corregir ajeno", err)
	}
	if _, err := s.Retirar(ctx, bActor, dominio.Informe, inf.ID, 1); !errors.Is(err, dominio.ErrNoEncontrado) {
		t.Fatal("retirar ajeno", err)
	}
	actual, err := s.Corregir(ctx, a, dominio.Comision, one.ID, dominio.Cambio{Fecha: "2026-01-02", VersionEsperada: 1})
	if err != nil || actual.Version != 2 {
		t.Fatal(actual, err)
	}
	_, err = s.Retirar(ctx, a, dominio.Comision, one.ID, 1)
	var conflicto *dominio.Conflicto
	if !errors.As(err, &conflicto) || conflicto.Actual != 2 {
		t.Fatal("conflicto", err)
	}
	if audit.ultimo.EntidadID != one.ID || audit.ultimo.Resultado != "error" {
		t.Fatalf("objetivo de fallo: %+v", audit.ultimo)
	}
	if _, err := s.Retirar(ctx, a, dominio.Comision, one.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Retirar(ctx, a, dominio.Informe, inf.ID, 1); err != nil {
		t.Fatal(err)
	}
	cifras, err = s.Recontar(ctx, coordinacion, f)
	if err != nil || cifras.Comisiones != 1 || cifras.Informes != 2 || cifras.PorOrganismo["JUZGADOS"] != 0 {
		t.Fatal(cifras, err)
	}
	for _, codigo := range []string{"OTROS", "NO_CATALOGADO"} {
		_, err := s.Registrar(ctx, a, dominio.Informe, dominio.Cambio{Fecha: "2026-09-01", Organismo: codigo})
		if codigo == "OTROS" && !errors.Is(err, dominio.ErrOtros) || codigo != "OTROS" && !errors.Is(err, dominio.ErrDatos) {
			t.Fatal(codigo, err)
		}
	}
	if _, err := s.Registrar(ctx, coordinacion, dominio.Comision, dominio.Cambio{Fecha: "2026-10-01"}); !errors.Is(err, dominio.ErrPermiso) {
		t.Fatal("coordinación escribe", err)
	}
	for _, perfil := range []string{"consulta", "administracion"} {
		actor := dominio.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{perfil}}
		if _, err := s.Recontar(ctx, actor, f); !errors.Is(err, dominio.ErrPermiso) {
			t.Fatal(perfil, err)
		}
	}
	// El fallo de auditoría revierte la escritura y nunca devuelve datos de lectura.
	fallida := app.Nuevo(&repo.Repositorio{}, b, auditoriaFallida{}, time.Now)
	if r, err := fallida.Registrar(ctx, a, dominio.Comision, dominio.Cambio{Fecha: "2026-11-01"}); err == nil || r.ID != "" {
		t.Fatal("escritura sin auditoría", r, err)
	}
	if r, err := fallida.Consultar(ctx, a, dominio.Informe, f); err == nil || r != nil {
		t.Fatal("lectura sin auditoría", r, err)
	}
	cifras, err = s.Recontar(ctx, coordinacion, f)
	if err != nil || cifras.Comisiones != 1 {
		t.Fatal("escritura revertida", cifras, err)
	}
	// Prueba directa como etf_app: segunda barrera sin pasar por la aplicación.
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		for _, tabla := range []string{"comision_tecnica", "informe_tecnico"} {
			var total int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM actividad."+tabla+" WHERE equipo_id=$1", bActor.EquipoID).Scan(&total); err != nil {
				return err
			}
			if total != 0 {
				t.Fatal("RLS ajena", tabla, total)
			}
		}
		tag, err := tx.Exec(ctx, `UPDATE actividad.informe_tecnico SET fecha=DATE '2026-08-01',version=version+1 WHERE equipo_id=$1`, bActor.EquipoID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			t.Fatal("RLS escritura ajena")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rechazo := func(identidad plataforma.Identidad, sql, codigo string, args ...any) {
		t.Helper()
		err := b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, sql, args...)
			return err
		})
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != codigo {
			t.Fatalf("rechazo %s: %v", codigo, err)
		}
	}
	rechazo(identidad, `INSERT INTO actividad.comision_tecnica(equipo_id,fecha) VALUES($1,DATE '2026-01-01')`, "42501", bActor.EquipoID)
	rechazo(identidad, `INSERT INTO actividad.informe_tecnico(equipo_id,fecha,organismo,organismo_otros) VALUES($1,DATE '2026-01-01','OTROS','  ')`, "23514", a.EquipoID)
	rechazo(identidad, `UPDATE actividad.informe_tecnico SET equipo_id=$1,version=version+1 WHERE equipo_id=$2 AND NOT retirada`, "23514", bActor.EquipoID, a.EquipoID)
	rechazo(identidad, `UPDATE actividad.informe_tecnico SET version=version+1,retirada=false WHERE id=$1`, "23514", inf.ID)
	rechazo(identidad, `DELETE FROM actividad.informe_tecnico`, "42501")
	rechazo(identidad, `UPDATE auditoria.evento SET accion='actividad.alterada'`, "42501")
	for _, id := range []plataforma.Identidad{{}, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}} {
		if err := b.Ejecutar(ctx, id, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM actividad.informe_tecnico`).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Fatal("RLS perfil", id.Perfiles, n)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("ACTIVIDAD-OK RLS dos equipos, años limítrofes, recuentos 15/16, Otros, conflictos, retirada y reversión de auditoría")
}
