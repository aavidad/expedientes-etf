package estadistica_test

import (
	"context"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type relojBloqueo struct{ ahora time.Time }

func (r *relojBloqueo) Ahora() time.Time { return r.ahora }

type txCierre struct{}

func (txCierre) Ejecutar(ctx context.Context, _ plataforma.Identidad, f func(context.Context) error) error {
	return f(ctx)
}

type auditCierre struct{ evento auditoria.Datos }

func (r *auditCierre) Confirmar(_ context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	r.evento = e
	return auditoria.Evento{}, nil
}
func (r *auditCierre) RegistrarEnTransaccion(ctx context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	return r.Confirmar(ctx, e)
}

type fuenteCierre struct {
	entrada.FuenteCierreMemoria
	conocido time.Time
}

func (f *fuenteCierre) MemoriaEnTransaccion(_ context.Context, _ d.ActorMemoria, filtro d.FiltroMemoria, conocido time.Time) (d.InformeMemoria, error) {
	f.conocido = conocido
	return d.InformeMemoria{FiltroMemoria: filtro, ConocidoEn: conocido}, nil
}

type repoCierre struct {
	salida.Cierres
	reloj   *relojBloqueo
	despues time.Time
}

func (r repoCierre) BloquearMemorias(context.Context) error { r.reloj.ahora = r.despues; return nil }
func (repoCierre) EstadoMemoria(_ context.Context, f d.FiltroMemoria, _ bool, _ string, _ time.Time) (d.EstadoMemoria, error) {
	return d.EstadoMemoria{FiltroMemoria: f, ID: "11111111-1111-4111-8111-111111111111", Version: 1, Estado: "borrador"}, nil
}
func (repoCierre) RepetirActo(context.Context, string, string, string, string, time.Time) (d.ResultadoActoMemoria, bool, error) {
	return d.ResultadoActoMemoria{}, false, nil
}
func (repoCierre) GuardarActo(_ context.Context, e d.EstadoMemoria, _ d.ActoMemoria, _ *d.RevisionMemoria, _ []d.InformeMemoria, _, _, _, _ string, _ time.Time) (d.ResultadoActoMemoria, error) {
	return d.ResultadoActoMemoria{EstadoMemoria: e}, nil
}
func TestCierreUsaInstantePosteriorAlBloqueo(t *testing.T) {
	antes := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	despues := antes.Add(time.Minute)
	reloj := &relojBloqueo{ahora: antes}
	fuente := &fuenteCierre{}
	audit := &auditCierre{}
	svc := app.NuevoCierre(app.DependenciasCierre{Repositorio: repoCierre{reloj: reloj, despues: despues}, Fuente: fuente, Transacciones: txCierre{}, Auditoria: audit, Reloj: reloj})
	_, err := svc.CerrarMemoria(context.Background(), d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"coordinacion"}}, d.FiltroMemoria{Anio: 2026, EquipoID: "33333333-3333-4333-8333-333333333333"}, "cierre-espera", d.ActoMemoria{Version: 1})
	if err != nil || !fuente.conocido.Equal(despues) || !audit.evento.OcurridoEn.Equal(despues) || audit.evento.EntidadID != "11111111-1111-4111-8111-111111111111" {
		t.Fatal("cierre calculó antes del bloqueo o perdió vínculo auditado", err, fuente.conocido, audit.evento)
	}
}
