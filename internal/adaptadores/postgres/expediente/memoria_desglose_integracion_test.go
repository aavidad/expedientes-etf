package expediente_test

import (
	"context"
	"errors"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	familia "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
)

func TestMemoriaDesgloseFotoRLSYAuditoria(t *testing.T) {
	b, ctx, deps, actor, p := prepararAceptacion(t)
	subida := subir(t, ctx, deps, actor)
	p = conMenores(p, subida)
	aceptada, err := familia.Nuevo(deps).Aceptar(ctx, actor, subida.ID, "memdes-"+subida.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	coordinacion := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: []string{"coordinacion"}}
	equipo := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	f := d.FiltroMemoria{Anio: 2026, EquipoID: p.EquipoID}
	persistencia := &repo.Repositorio{}
	fuente := app.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Actividad: &actividad.Repositorio{}, Equipos: &catalogo.Repositorio{}, Narrativas: persistencia}
	audit := postgres.NuevoRegistroAuditoria(b)
	cierres := app.NuevoCierre(app.DependenciasCierre{Repositorio: persistencia, Fuente: fuente, Transacciones: b, Auditoria: audit, Reloj: relojMemoria{}})
	estado, err := cierres.EstadoMemoria(ctx, coordinacion, f)
	if err != nil {
		t.Fatal(err)
	}
	if estado.Estado == "cerrada" {
		result, e := cierres.ReabrirMemoria(ctx, coordinacion, f, "memdes-reabrir-"+subida.ID, d.ActoMemoria{Version: estado.Version, Motivo: "Se comprueban los aportes por expediente."})
		if e != nil {
			t.Fatal(e)
		}
		estado = result.EstadoMemoria
	}
	fuenteConCierre := app.FuenteConCierre{Repositorio: persistencia, Fuente: fuente}
	svc := app.ServicioNavegacion{Cierres: persistencia, Fuente: fuenteConCierre, Numeros: &exp.LecturaEstadistica{}, Transacciones: b, Auditoria: audit, Reloj: relojMemoria{}}
	s := d.SeleccionDesglose{Celda: -1, Medida: "familias"}
	antes, err := svc.DesglosarMemoria(ctx, equipo, f, d.CasosAtendidos, s)
	if err != nil || antes.Aviso != "" {
		t.Fatal(antes, err)
	}
	encontrada := false
	suma := 0
	for _, fila := range antes.Filas {
		suma += fila.Cantidad
		if fila.ExpedienteID == aceptada.ID {
			encontrada = true
			if !fila.Ficha {
				t.Fatal("ficha propia ausente")
			}
		}
	}
	if !encontrada || suma != antes.Cifra {
		t.Fatal("suma o expediente", antes)
	}
	cierre, err := cierres.CerrarMemoria(ctx, equipo, f, "memdes-cerrar-"+subida.ID, d.ActoMemoria{Version: estado.Version})
	if err != nil {
		t.Fatal(err)
	}
	s.Revision = cierre.RevisionActual
	foto, err := svc.DesglosarMemoria(ctx, equipo, f, d.CasosAtendidos, s)
	if err != nil || !foto.Definitivas || foto.Cifra != antes.Cifra || foto.Aviso != "" {
		t.Fatal(foto, err)
	}
	// Un caso aceptado después del cierre no entra en la revisión elegida.
	nueva := subir(t, ctx, deps, actor)
	pNueva := conMenores(p, nueva)
	_, err = familia.Nuevo(deps).Aceptar(ctx, actor, nueva.ID, "memdes-posterior-"+nueva.ID, pNueva)
	if err != nil {
		t.Fatal(err)
	}
	despues, err := svc.DesglosarMemoria(ctx, equipo, f, d.CasosAtendidos, s)
	if err != nil || despues.Cifra != foto.Cifra || len(despues.Filas) != len(foto.Filas) {
		t.Fatal("foto mutable", despues, err)
	}
	ajeno := equipo
	ajeno.EquipoID = "44444444-4444-4444-8444-444444444444"
	for _, a := range []d.ActorMemoria{ajeno, {UsuarioID: actor.UsuarioID, Perfiles: []string{"consulta"}}} {
		r, err := svc.DesglosarMemoria(ctx, a, f, d.CasosAtendidos, s)
		if !errors.Is(err, d.ErrPermisoMemoria) || len(r.Filas) != 0 {
			t.Fatal("permiso", r, err)
		}
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: ajeno.UsuarioID, Perfiles: ajeno.Perfiles, EquipoID: ajeno.EquipoID}, func(ctx context.Context) error {
		numeros, err := (&exp.LecturaEstadistica{}).NumerosMemoria(ctx, []string{aceptada.ID})
		if err != nil {
			return err
		}
		if len(numeros) != 0 {
			t.Fatal("RLS entrega número ajeno", numeros)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("MEMDES-RLS-OK: equipo propio, ajeno y Consulta denegados; revisión estable tras nueva aceptación")
}
