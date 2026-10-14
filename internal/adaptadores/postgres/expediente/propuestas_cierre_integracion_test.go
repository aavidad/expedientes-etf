package expediente_test

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"sync"
	"testing"
)

func TestPropuestaCierrePostgres(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	defer b.Cerrar()
	deps.Propuestas = &repo.Repositorio{}
	deps.MotivosCierre = &catalogo.Repositorio{}
	s := app.Nuevo(deps)
	ficha := subir(t, ctx, deps, coord)
	p = conMenores(p, ficha)
	r, err := s.Aceptar(ctx, coord, ficha.ID, "aceptar-cierre", p)
	if err != nil {
		t.Fatal(err)
	}
	actor := d.Actor{UsuarioID: coord.UsuarioID, EquipoID: p.EquipoID, Perfiles: []string{"profesional_etf"}}
	f := d.Finalizacion{Version: r.Version, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true, ObjetivosPendientes: "No se ha iniciado la intervención por falta de consentimiento familiar."}
	if _, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "guardar-cierre", f); err != nil {
		t.Fatal(err)
	}
	g, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "guardar-cierre", f)
	if err != nil || g.Version != 3 {
		t.Fatal(g, err)
	}
	if _, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "guardar-diferente", f); err == nil {
		t.Fatal("falta conflicto")
	} else {
		var c d.Conflicto
		if !errors.As(err, &c) || c.VersionActual != 3 {
			t.Fatal(err)
		}
	}
	f.Motivo = "OBJ_TOTAL"
	if _, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "guardar-cierre", f); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal(err)
	}
	f.Version = 3
	if _, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "motivo-no-permitido", f); !errors.Is(err, d.ErrConsentimientoCierre) {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	rs := make([]d.PropuestaCierre, n)
	es := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rs[i], es[i] = s.ProponerCierre(ctx, actor, ficha.ID, "propuesta-unica", d.VersionCierre{Version: 3})
		}(i)
	}
	wg.Wait()
	for i := range n {
		if es[i] != nil || rs[i] != rs[0] {
			t.Fatal(i, rs[i], es[i])
		}
	}
	if rs[0].Version != 4 || rs[0].Estado != "pendiente" {
		t.Fatal(rs[0])
	}
	if _, err := s.GuardarFinalizacion(ctx, actor, ficha.ID, "editar-pendiente", d.Finalizacion{Version: 4, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true}); !errors.Is(err, d.ErrEstado) {
		t.Fatal(err)
	}
	for _, perfil := range []string{"consulta", "administracion", "coordinacion", ""} {
		a := actor
		a.Perfiles = nil
		if perfil != "" {
			a.Perfiles = []string{perfil}
		}
		if _, err := s.ProponerCierre(ctx, a, ficha.ID, "sin-permiso-"+perfil, d.VersionCierre{Version: 4}); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(perfil, err)
		}
	}
	ajeno := actor
	ajeno.EquipoID = "88888888-8888-4888-8888-888888888888"
	if _, err := s.ProponerCierre(ctx, ajeno, ficha.ID, "otro-equipo", d.VersionCierre{Version: 4}); !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal(err)
	}
	for _, a := range []d.Actor{ajeno, {UsuarioID: actor.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: actor.UsuarioID, Perfiles: []string{"administracion"}}} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"finalizacion", "propuesta_cierre"} {
				var total int
				if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla+" WHERE expediente_id=$1", ficha.ID).Scan(&total); err != nil {
					return err
				}
				if total != 0 {
					t.Fatal("RLS", tabla, total)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	lista, err := s.ListarPropuestasCierre(ctx, coord, 200, 0)
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	for _, item := range lista {
		if item.ID == rs[0].ID {
			encontrada = true
		}
	}
	if !encontrada {
		t.Fatal("bandeja vacía")
	}
	actual, err := s.Consultar(ctx, coord, ficha.ID)
	if err != nil || actual.Estado != d.Aceptada || actual.Version != 4 {
		t.Fatal(actual, err)
	}
	comprobarHistoriaDecision(t, b, ctx, coord, ficha.ID, 1, 1, 1)
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err := app.Nuevo(fallo).RetirarPropuestaCierre(ctx, actor, ficha.ID, "retirar", d.VersionCierre{Version: 4}); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	retirada, err := s.RetirarPropuestaCierre(ctx, actor, ficha.ID, "retirar", d.VersionCierre{Version: 4})
	if err != nil || retirada.Version != 5 || retirada.Estado != "retirada" {
		t.Fatal(retirada, err)
	}
	t.Log("ACEPTADA sin intervención; 8 POST idempotentes, un recibo; versión 409; perfiles y equipo ajeno denegados; sin hitos nuevos; retirada revierte si falla auditoría")
}
