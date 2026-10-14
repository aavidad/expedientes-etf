package expediente_test

import (
	"context"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestSPMEstadoFechaRespuestaRepeticionYReversionAuditoria(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	deps.Hitos, deps.SalidasSPM = &adaptador.Repositorio{}, &adaptador.Repositorio{}
	s := app.Nuevo(deps)
	if _, err := s.ConfirmarHito(ctx, a, id, "spm-consentimiento-"+id, d.ConfirmacionHito{Version: 2, Tipo: "consentimiento", Fecha: "2026-02-04"}); err != nil {
		t.Fatal(err)
	}
	comprobar := func(estado string, version, cambios int) {
		t.Helper()
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			e, err := (&adaptador.Repositorio{}).Obtener(ctx, id)
			if err != nil {
				return err
			}
			if e.Estado != estado || e.Version != version {
				t.Fatal("estado y versión", e.Estado, e.Version, estado, version)
			}
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var n int
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.estado_cambio WHERE expediente_id=$1 AND (estado_nuevo='derivada_spm' OR estado_anterior='derivada_spm')`, id).Scan(&n); err != nil {
				return err
			}
			if n != cambios {
				t.Fatal("historia repetida", n, cambios)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.OperarSPM(ctx, a, id, "spm-preparar-"+id, "crear", d.PeticionSPM{Version: 3, Motivo: "Se solicita valoración de protección.", PropuestaMedida: "Valorar una medida protectora."})
	if err != nil {
		t.Fatal(err)
	}
	comprobar("en_intervencion", 3, 0)
	confirmacion := d.PeticionSPM{Version: r.Version, Fecha: "2026-02-05"}
	fallidas := deps
	fallidas.RegistroTransaccional = auditoriaFallidaSPM{}
	if _, err = app.Nuevo(fallidas).OperarSPM(ctx, a, id, "spm-auditoria-"+id, "confirmar", confirmacion); err == nil {
		t.Fatal("fecha sin auditoría")
	}
	comprobar("en_intervencion", 3, 0)
	r, err = s.OperarSPM(ctx, a, id, "spm-confirmar-"+id, "confirmar", confirmacion)
	if err != nil {
		t.Fatal(err)
	}
	comprobar("derivada_spm", 4, 1)
	if _, err = s.OperarSPM(ctx, a, id, "spm-confirmar-"+id, "confirmar", confirmacion); err != nil {
		t.Fatal(err)
	}
	comprobar("derivada_spm", 4, 1)
	r, err = s.OperarSPM(ctx, a, id, "spm-espera-"+id, "responder", d.PeticionSPM{Version: r.Version, Respuesta: d.SPMAlaEspera})
	if err != nil {
		t.Fatal(err)
	}
	comprobar("derivada_spm", 4, 1)
	respuesta := d.PeticionSPM{Version: r.Version, Respuesta: d.SPMMedidaNoNecesaria, Fecha: "2026-02-06"}
	if _, err = app.Nuevo(fallidas).OperarSPM(ctx, a, id, "spm-respuesta-auditoria-"+id, "responder", respuesta); err == nil {
		t.Fatal("respuesta sin auditoría")
	}
	comprobar("derivada_spm", 4, 1)
	if _, err = s.OperarSPM(ctx, a, id, "spm-responder-"+id, "responder", respuesta); err != nil {
		t.Fatal(err)
	}
	comprobar("en_intervencion", 5, 2)
	if _, err = s.OperarSPM(ctx, a, id, "spm-responder-"+id, "responder", respuesta); err != nil {
		t.Fatal(err)
	}
	comprobar("en_intervencion", 5, 2)
	t.Log("SPM: fecha → derivada_spm; espera conserva bloqueo; respuesta → en_intervencion; repetición sin duplicados y auditoría fallida revierte ambos cambios")
}
