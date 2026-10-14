package expediente_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func prepararHitos(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, dominio.Actor, string) {
	t.Helper()
	b, ctx, deps, coordinacion, p := prepararAceptacion(t)
	deps.Hitos = &adaptador.Repositorio{}
	d := subir(t, ctx, deps, coordinacion)
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, coordinacion, d.ID, "aceptar-"+d.ID, conMenores(p, d)); err != nil {
		t.Fatal(err)
	}
	profesional := dominio.Actor{UsuarioID: coordinacion.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	return b, ctx, deps, coordinacion, profesional, d.ID
}

func TestHitosConsentimientoConcurrenteHistoriaYWord(t *testing.T) {
	b, ctx, deps, coordinacion, profesional, id := prepararHitos(t)
	s := aplicacion.Nuevo(deps)
	p := dominio.ConfirmacionHito{Tipo: "consentimiento", Fecha: "2026-02-05", Version: 2}
	rs, errs := make([]dominio.ResultadoHito, 12), make([]error, 12)
	var wg sync.WaitGroup
	for i := range rs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); rs[i], errs[i] = s.ConfirmarHito(ctx, profesional, id, "firma-"+id, p) }(i)
	}
	wg.Wait()
	for i, r := range rs {
		if errs[i] != nil || r.Estado != dominio.EnIntervencion || r.Version != 3 || !reflect.DeepEqual(rs[0], r) {
			t.Fatal(i, r, errs[i])
		}
	}
	ajeno := profesional
	ajeno.EquipoID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := s.ConfirmarHito(ctx, ajeno, id, "firma-"+id, p); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal("recibo entregado fuera de equipo vigente", err)
	}
	if _, err := s.ConfirmarHito(ctx, profesional, id, "otra-"+id, p); err == nil {
		t.Fatal("no detectó versión anterior")
	} else {
		var c dominio.Conflicto
		if !errors.As(err, &c) || c.VersionActual != 3 {
			t.Fatal(err)
		}
	}
	p.Fecha = "2026-02-06"
	if _, err := s.ConfirmarHito(ctx, profesional, id, "firma-"+id, p); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	hitos, err := s.ConsultarHitos(ctx, profesional, id)
	if err != nil || len(hitos) != 2 || hitos[1].Tipo != "consentimiento" || hitos[1].ConfirmadoPor != profesional.UsuarioID {
		t.Fatal(hitos, err)
	}
	d, err := s.Consultar(ctx, profesional, id)
	if err != nil || d.Aceptacion == nil || d.Aceptacion.FechaActa != "2026-02-03" || d.Estado != dominio.EnIntervencion {
		t.Fatal(d, err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coordinacion.UsuarioID, Perfiles: coordinacion.Perfiles}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		for tabla, esperado := range map[string]int{"hito": 2, "estado_cambio": 2, "episodio": 1} {
			var n int
			if err := e.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM expediente.%s WHERE expediente_id=$1", tabla), id).Scan(&n); err != nil {
				return err
			}
			if n != esperado {
				t.Fatalf("%s: %d", tabla, n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: coordinacion.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ev := range exportacion.Eventos {
		if ev.ExpedienteID == id && ev.Accion == "expediente.confirmar_hito" {
			n++
		}
	}
	if n != 1 {
		t.Fatal("auditoría duplicada", n)
	}
	t.Log("12 confirmaciones: un consentimiento, un cambio de estado, un evento; repetición conserva recibo; ficha conserva datos Word tras firma")
}

func TestHitosAnexosFechasRLSYAuditoriaAtomica(t *testing.T) {
	b, ctx, deps, coordinacion, profesional, id := prepararHitos(t)
	s := aplicacion.Nuevo(deps)
	for i, p := range []dominio.ConfirmacionHito{
		{Tipo: "consentimiento", Fecha: "2026-02-02", Version: 2}, {Tipo: "consentimiento", Fecha: "9999-12-31", Version: 2}, {Tipo: "anexo8", Fecha: "2026-02-05", Version: 2},
	} {
		if _, err := s.ConfirmarHito(ctx, profesional, id, fmt.Sprintf("invalido-%d-%s", i, id), p); err == nil {
			t.Fatal(p)
		}
	}
	ajeno := profesional
	ajeno.EquipoID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := s.ConfirmarHito(ctx, ajeno, id, "ajeno-"+id, dominio.ConfirmacionHito{Tipo: "anexo7", Fecha: "2026-02-05", Version: 2}); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal(err)
	}
	if _, err := s.ConsultarHitos(ctx, ajeno, id); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal(err)
	}
	// Los permisos SQL compartidos de aceptación no permiten corregir la fecha.
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: profesional.UsuarioID, Perfiles: profesional.Perfiles, EquipoID: profesional.EquipoID}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		_, err = e.Exec(ctx, `UPDATE expediente.expediente SET fecha_aceptacion='2026-02-04',modificado_por=$2 WHERE id=$1`, id, profesional.UsuarioID)
		return err
	})
	if err == nil {
		t.Fatal("profesional corrigió aceptación por SQL")
	}
	for _, perfil := range []string{"consulta", "administracion"} {
		a := profesional
		a.Perfiles = []string{perfil}
		if _, err := s.ConsultarHitos(ctx, a, id); !errors.Is(err, dominio.ErrAcceso) {
			t.Fatal(perfil, err)
		}
	}
	p := dominio.ConfirmacionHito{Tipo: "anexo7", Fecha: "2026-02-05", Version: 2}
	fallidos := deps
	fallidos.RegistroTransaccional = registroFallido{}
	if _, err := aplicacion.Nuevo(fallidos).ConfirmarHito(ctx, profesional, id, "anexo-"+id, p); !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal(err)
	}
	r, err := s.ConfirmarHito(ctx, profesional, id, "anexo-"+id, p)
	if err != nil || r.Estado != dominio.Aceptada || r.Version != 3 {
		t.Fatal(r, err)
	}
	p.Version = 3
	if _, err := s.ConfirmarHito(ctx, profesional, id, "duplicado-"+id, p); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	if hs, err := aplicacion.Nuevo(fallidos).ConsultarHitos(ctx, profesional, id); hs != nil || !errors.Is(err, dominio.ErrNoDisponible) {
		t.Fatal(hs, err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coordinacion.UsuarioID, Perfiles: coordinacion.Perfiles}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var n int
		if err := e.QueryRow(ctx, `SELECT count(*) FROM expediente.estado_cambio WHERE expediente_id=$1`, id).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			t.Fatal("anexo cambió estado", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("Fecha anterior/futura y subprograma rechazados; equipo ajeno invisible; anexos no cambian estado; fallo de auditoría revierte hito/versión/recibo y no entrega lectura")
}
