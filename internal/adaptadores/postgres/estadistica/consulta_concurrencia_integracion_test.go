package estadistica_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	actividadApp "github.com/aavidad/expedientes-etf/internal/aplicacion/actividad"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	actividadDominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type calculoEnEspera struct {
	motor     salida.CalculoConsulta
	iniciados chan<- struct{}
	continuar <-chan struct{}
}

func (s calculoEnEspera) CalcularConsulta(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.CifrasConsulta, error) {
	if _, err := postgres.DesdeContexto(ctx); err == nil {
		return d.CifrasConsulta{}, errors.New("consulta.transaccion_retenida")
	}
	select {
	case s.iniciados <- struct{}{}:
	case <-ctx.Done():
		return d.CifrasConsulta{}, ctx.Err()
	}
	select {
	case <-s.continuar:
	case <-ctx.Done():
		return d.CifrasConsulta{}, ctx.Err()
	}
	return s.motor.CalcularConsulta(ctx, a, f, ahora)
}

func TestConsultaPostgresLectoresYCierreSinBloqueo(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	tecnico, err := postgres.NuevaCalculoMemoria(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer tecnico.Cerrar()
	persistencia := &repo.Repositorio{}
	reglasMemoria := reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}
	fuente := app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{}, Reglas: reglasMemoria, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: persistencia}
	fuenteTecnica := app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{CalculoInterno: true}, Reglas: reglasMemoria, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{CalculoInterno: true}}
	cierre := app.NuevoCierre(app.DependenciasCierre{Repositorio: persistencia, Fuente: fuente, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
	consulta := actor
	consulta.Perfiles = []string{"consulta"}
	f := d.FiltroMemoria{Anio: 2026, EquipoID: equipo}
	estado, err := cierre.EstadoMemoria(ctx, actor, f)
	if err != nil {
		t.Fatal(err)
	}
	if estado.Estado == "cerrada" {
		if _, err = cierre.ReabrirMemoria(ctx, actor, f, "fx4-reabrir-"+actor.UsuarioID, d.ActoMemoria{Version: estado.Version, Motivo: "Se comprueba el cálculo concurrente de las cifras."}); err != nil {
			t.Fatal(err)
		}
		estado, err = cierre.EstadoMemoria(ctx, actor, f)
		if err != nil {
			t.Fatal(err)
		}
	}
	actividadServicio := actividadApp.Nuevo(&actividad.Repositorio{}, b, postgres.NuevoRegistroAuditoriaTransaccional(b), time.Now)
	profesional := actividadDominio.Actor{UsuarioID: actor.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: equipo}
	if _, err = actividadServicio.Registrar(ctx, profesional, actividadDominio.Comision, actividadDominio.Cambio{Fecha: "2026-02-12", Observaciones: "Se revisaron las actuaciones de coordinación del equipo."}); err != nil {
		t.Fatal(err)
	}
	const lectores = 6 // Supera las cuatro conexiones del pool de la aplicación.
	iniciados := make(chan struct{}, lectores)
	continuar := make(chan struct{})
	defer func() {
		if continuar != nil {
			close(continuar)
		}
	}()
	motor := app.EjecutorConsulta{Transacciones: tecnico, Fuente: fuenteTecnica}
	deps := app.DependenciasConsulta{Reglas: &catalogo.Repositorio{}, Cerradas: persistencia, Calculo: calculoEnEspera{motor, iniciados, continuar}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}
	servicio := app.NuevoConsulta(deps)
	type resultado struct {
		cifras d.CifrasConsulta
		err    error
	}
	resultados := make(chan resultado, lectores)
	for i := 0; i < lectores; i++ {
		go func() { cifras, e := servicio.ConsultarCifras(ctx, consulta, f); resultados <- resultado{cifras, e} }()
	}
	for i := 0; i < lectores; i++ {
		select {
		case <-iniciados:
		case <-time.After(3 * time.Second):
			t.Fatal("los lectores retienen conexiones o se bloquean entre sí")
		}
	}
	plazo, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cerrada, err := cierre.CerrarMemoria(plazo, actor, f, "fx4-cierre-"+actor.UsuarioID, d.ActoMemoria{Version: estado.Version})
	if err != nil {
		t.Fatal("el cierre espera al cálculo de Consulta", err)
	}
	if cerrada.Estado != "cerrada" {
		t.Fatal(cerrada)
	}
	deps.Calculo = motor
	foto, err := app.NuevoConsulta(deps).ConsultarCifras(plazo, consulta, f)
	if err != nil || !foto.Definitivas || len(foto.Apartados) != 33 {
		t.Fatal("foto cerrada incoherente", foto, err)
	}
	comisiones := 0
	for _, apartado := range foto.Apartados {
		if apartado.Codigo == d.ComisionesTecnicas {
			comisiones = apartado.Total.Comisiones
		}
	}
	if comisiones != d.CifraSuprimida {
		t.Fatalf("comisiones: recibido %d, esperado oculto", comisiones)
	}
	close(continuar)
	continuar = nil
	for i := 0; i < lectores; i++ {
		r := <-resultados
		if r.err != nil || r.cifras.Definitivas || !reflect.DeepEqual(r.cifras.Apartados, foto.Apartados) {
			t.Fatal("cifras provisionales incoherentes con el cierre", r.err, r.cifras)
		}
	}
	// Una escritura que retenga el bloqueo de Memoria tampoco impide leer la foto.
	bloqueado := make(chan struct{})
	soltar := make(chan struct{})
	fin := make(chan error, 1)
	go func() {
		fin <- b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
			if e := persistencia.BloquearMemorias(ctx); e != nil {
				return e
			}
			close(bloqueado)
			select {
			case <-soltar:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-bloqueado:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	foto2, err := app.NuevoConsulta(deps).ConsultarCifras(plazo, consulta, f)
	close(soltar)
	if e := <-fin; e != nil {
		t.Fatal(e)
	}
	if err != nil || !reflect.DeepEqual(foto, foto2) {
		t.Fatal("la foto espera un bloqueo asesor", err)
	}
	if _, err = cierre.ReabrirMemoria(ctx, actor, f, "fx4-restaurar-"+actor.UsuarioID, d.ActoMemoria{Version: cerrada.Version, Motivo: "Finaliza la comprobación de concurrencia y se restauran las lecturas abiertas."}); err != nil {
		t.Fatal(err)
	}
	t.Log("CONSULTA-CONCURRENCIA-OK 6 lectores, cierre antes de liberar cálculo, 33 apartados coherentes y foto sin bloqueo asesor")
}
