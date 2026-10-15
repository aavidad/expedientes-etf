package expediente_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"sync"
	"testing"
)

func TestSEG3TransaccionOrdinaria(t *testing.T) {
	b, ctx, _, a, _ := prepararAceptacion(t)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var id string
		if err = tx.QueryRow(ctx, `INSERT INTO expediente.expediente(id,estado,fecha_recepcion) VALUES(gen_random_uuid(),'derivacion_recibida',now()) RETURNING id::text`).Scan(&id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido) VALUES($1,'1','{}')`, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSEG3PlazoDuplicadosAnulacionYRLS(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	admin := a
	admin.Perfiles = []string{"administracion"}
	plazo, err := s.ConsultarPlazoCarga(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		actual, e := s.ConsultarPlazoCarga(ctx, admin)
		if e == nil {
			actual.Fecha = "2026-12-31"
			_, e = s.FijarPlazoCarga(ctx, admin, actual)
		}
		if e != nil {
			t.Error(e)
		}
	})
	copia := plazo
	copia.Fecha = "2026-01-01"
	if _, err = s.FijarPlazoCarga(ctx, a, copia); !errors.Is(err, d.ErrAcceso) {
		t.Fatal("Coordinación cambia plazo", err)
	}
	cerrada, err := s.FijarPlazoCarga(ctx, admin, copia)
	if err != nil {
		t.Fatal(err)
	}
	c := casoHistorico(p, "")
	c.Familia = nil
	c.Menores = nil
	// Identidad inventada distinta de las demás pruebas del mismo ensayo.
	c.Datos.Extraccion.Campos = []d.Campo{{Codigo: "menores.persona.nombre", Valor: "Mateo Cárdenas Robles"}}
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{c}}
	if _, err = s.ValidarCargaHistorica(ctx, a, carga); !errors.Is(err, d.ErrCargaCerrada) {
		t.Fatal("carga tras cierre", err)
	}
	cerrada.Fecha = "2026-12-31"
	if _, err = s.FijarPlazoCarga(ctx, admin, cerrada); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FijarPlazoCarga(ctx, admin, cerrada); err == nil {
		t.Fatal("versión antigua aceptada")
	}
	v, err := s.ValidarCargaHistorica(ctx, a, carga)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CargarHistoricos(ctx, a, v.LoteID, "seg3-"+v.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := s.ValidarCargaHistorica(ctx, a, carga)
	if err != nil || len(v2.Avisos) == 0 {
		t.Fatal("duplicado sin aviso", v2, err)
	}
	if _, err = s.CargarHistoricos(ctx, a, v2.LoteID, "seg3-"+v2.LoteID, carga); !errors.Is(err, d.ErrDuplicadosSinConfirmar) {
		t.Fatal("duplicado sin confirmación", err)
	}
	carga.ConfirmarDuplicados = true
	abierta := c
	abierta.Estado = d.EnIntervencion
	abierta.Episodios = []d.EpisodioHistorico{{Apertura: p.FechaActa}}
	carga.Casos = []d.CasoHistorico{abierta}
	r2, err := s.CargarHistoricos(ctx, a, v2.LoteID, "seg3-"+v2.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AnularLoteCarga(ctx, admin, r.LoteID); !errors.Is(err, d.ErrAcceso) {
		t.Fatal("Administración anula", err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var actividad bool
		if e = tx.QueryRow(ctx, `SELECT actividad_posterior FROM expediente.carga_inicial WHERE id=$1`, r.LoteID).Scan(&actividad); e != nil {
			return e
		}
		if actividad {
			t.Fatal("actividad marcada durante carga")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	fallos := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); fallos <- s.AnularLoteCarga(ctx, a, r.LoteID) }()
	}
	wg.Wait()
	close(fallos)
	for e := range fallos {
		if e != nil {
			t.Fatal("anulación concurrente", e)
		}
	}
	if err = s.AnularLoteCarga(ctx, a, r.LoteID); err != nil {
		t.Fatal("repetir anulación", err)
	}
	lotes, err := s.ConsultarLotesCarga(ctx, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	for _, l := range lotes {
		if l.ID == r.LoteID {
			encontrada = l.Anulado
		}
	}
	if !encontrada {
		t.Fatal("recibo anulado ausente")
	}
	if _, err = s.Consultar(ctx, a, r.Expedientes[0]); err == nil {
		t.Fatal("expediente anulado sigue disponible")
	}
	if _, err = s.CargarHistoricos(ctx, a, r.LoteID, "seg3-"+r.LoteID, d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{c}}); err == nil {
		t.Fatal("recibo de carga anulado recrea casos")
	}
	if _, err = s.ConsultarLotesCarga(ctx, admin, 0); !errors.Is(err, d.ErrAcceso) {
		t.Fatal("Administración lee lotes", err)
	}
	// Un cambio posterior que se revierte a su valor previo sigue siendo actividad.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE expediente.carga_inicial SET actividad_posterior=false WHERE id=$1`, r2.LoteID)
		return e
	})
	if err == nil {
		t.Fatal("app puede borrar marcador")
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE expediente.expediente SET version=version+1,modificado_por=$2,modificado_en=now() WHERE id=$1`, r2.Expedientes[0], a.UsuarioID)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = s.AnularLoteCarga(ctx, a, r2.LoteID); !errors.Is(err, d.ErrLoteConActividad) {
		t.Fatal("lote con actividad anulado", err)
	}
}

func TestSEG3CargaReservaNumeracionOrdinaria(t *testing.T) {
	_, ctx, deps, a, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	c := casoHistorico(p, "")
	c.Familia = nil
	c.Menores = nil
	carga := d.CargaHistorica{EquipoID: p.EquipoID}
	for i := 1; i <= 99; i++ {
		v := c
		v.Numero = fmt.Sprintf("08-%02d-26", i)
		carga.Casos = append(carga.Casos, v)
	}
	if _, err := s.ValidarCargaHistorica(ctx, a, carga); !errors.Is(err, d.ErrCargaAgotaNumeracion) {
		t.Fatal("agotamiento permitido", err)
	}
}

func TestSEG3EvaluacionPosteriorImpideAnularTodoElLote(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	c := casoHistorico(p, "")
	c.Estado = d.EnIntervencion
	c.Episodios = []d.EpisodioHistorico{{Apertura: p.FechaActa}}
	c.Familia = nil
	c.Menores = nil
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{c, c}, ConfirmarDuplicados: true}
	v, err := s.ValidarCargaHistorica(ctx, a, carga)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CargarHistoricos(ctx, a, v.LoteID, "seg3-evaluacion-"+v.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `WITH nueva AS(SELECT gen_random_uuid() AS id)
 INSERT INTO evaluacion.instrumento_aplicado(id,expediente_id,equipo_id,datos)
 SELECT id,$1,$2,jsonb_build_object('id',id,'expediente_id',$1::uuid,'equipo_id',$2::uuid) FROM nueva`, r.Expedientes[1], p.EquipoID)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AnularLoteCarga(ctx, a, r.LoteID); !errors.Is(err, d.ErrLoteConActividad) {
		t.Fatal("evaluación eliminada con lote", err)
	}
	for _, id := range r.Expedientes {
		if _, err = s.Consultar(ctx, a, id); err != nil {
			t.Fatal("rechazo retiró parte del lote", err)
		}
	}
}

func TestSEG3EmisionConcurrenteConAnulacion(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	c := casoHistorico(p, "")
	c.Estado = d.EnIntervencion
	c.Episodios = []d.EpisodioHistorico{{Apertura: p.FechaActa}}
	c.Familia = nil
	c.Menores = nil
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{c}, ConfirmarDuplicados: true}
	v, err := s.ValidarCargaHistorica(ctx, a, carga)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CargarHistoricos(ctx, a, v.LoteID, "seg3-emision-"+v.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	empezar := make(chan struct{})
	resultados := make(chan error, 2)
	go func() {
		<-empezar
		resultados <- b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
			tx, e := postgres.DesdeContexto(ctx)
			if e != nil {
				return e
			}
			etiqueta, e := tx.Exec(ctx, `INSERT INTO documento.emision(id,tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_sha256,huella_datos,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave)
 SELECT gen_random_uuid(),tipo_documento,$1,$2,id,idioma,DATE '2026-10-01',$3,now(),decode(repeat('ab',32),'hex'),decode(repeat('cd',32),'hex'),'{}','1','idiomas/es','1',''
 FROM documento.plantilla WHERE idioma='es' ORDER BY id LIMIT 1`, r.Expedientes[0], p.EquipoID, a.UsuarioID)
			if e == nil && etiqueta.RowsAffected() != 1 {
				return errors.New("fixture.emision_ausente")
			}
			return e
		})
	}()
	go func() { <-empezar; resultados <- s.AnularLoteCarga(ctx, a, r.LoteID) }()
	close(empezar)
	uno, dos := <-resultados, <-resultados
	if uno == nil && dos == nil {
		t.Fatal("emisión y anulación incompatibles confirmadas")
	}
	if uno != nil && dos != nil {
		t.Fatal("ambas operaciones fallaron", uno, dos)
	}
}

func TestMemoriaConsultaSEG3RolTecnicoConservaProyeccion(t *testing.T) {
	if os.Getenv("ETF_CALCULO_MEMORIA_CONTRASENA") == "" {
		t.Skip("fixture.rol_calculo_ausente")
	}
	_, ctx, _, a, p := prepararAceptacion(t)
	tecnico, err := postgres.NuevaCalculoMemoria(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer tecnico.Cerrar()
	actor := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}, EquipoID: p.EquipoID}
	err = tecnico.Ejecutar(ctx, actor, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var cantidad int
		return tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM estadistica.calculo_expedientes)+(SELECT count(*) FROM estadistica.calculo_episodios)`).Scan(&cantidad)
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Fatalf("%s: %s", pg.Code, pg.Message)
		}
		t.Fatal(err)
	}
	err = tecnico.Ejecutar(ctx, actor, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `SELECT id FROM expediente.expediente`)
		return e
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "42501" {
		t.Fatal("rol técnico accede a ficha", err)
	}
	err = tecnico.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var cantidad int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.calculo_expedientes`).Scan(&cantidad); e != nil {
			return e
		}
		if cantidad != 0 {
			return errors.New("fixture.proyeccion_sin_contexto")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}
