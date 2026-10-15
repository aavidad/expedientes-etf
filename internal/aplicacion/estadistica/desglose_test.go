package estadistica_test

import (
	"context"
	"errors"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"testing"
	"time"
)

type fuenteDesglose struct {
	informe  d.InformeMemoria
	llamadas int
}

func (s *fuenteDesglose) MemoriaEnTransaccion(context.Context, d.ActorMemoria, d.FiltroMemoria, time.Time) (d.InformeMemoria, error) {
	s.llamadas++
	return s.informe, nil
}

type numerosDesglose struct{ visibles map[string]string }

func (s numerosDesglose) NumerosMemoria(context.Context, []string) (map[string]string, error) {
	return s.visibles, nil
}

type fotoDesglose struct {
	salida.Cierres
	informe  d.InformeMemoria
	revision int
}

func (s *fotoDesglose) LeerRevision(_ context.Context, _ d.FiltroMemoria, n int) (d.RevisionMemoria, error) {
	s.revision = n
	return d.RevisionMemoria{Informe: s.informe}, nil
}
func TestDesglosePermisosAuditoriaYFotoDeRevision(t *testing.T) {
	f := d.FiltroMemoria{Anio: 2026, EquipoID: "33333333-3333-4333-8333-333333333333"}
	informe := d.InformeMemoria{FiltroMemoria: f, ConocidoEn: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), VersionDesglose: 1, Apartados: []d.ApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.Recuento{Familias: 1}}}, Desglose: []d.AporteMemoria{{Apartado: d.CasosAtendidos, ExpedienteID: "44444444-4444-4444-8444-444444444444", EquipoID: f.EquipoID, Numero: "02-03-26", Total: d.Recuento{Familias: 1}}}}
	fuente := &fuenteDesglose{informe: informe}
	foto := &fotoDesglose{informe: informe}
	audit := &auditCierre{}
	svc := app.ServicioNavegacion{Fuente: fuente, Cierres: foto, Numeros: numerosDesglose{map[string]string{"44444444-4444-4444-8444-444444444444": "02-99-26"}}, Transacciones: txConsulta{}, Auditoria: audit, Reloj: &relojBloqueo{ahora: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)}}
	actor := d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"profesional_etf"}, EquipoID: f.EquipoID}
	s := d.SeleccionDesglose{Celda: -1, Medida: "familias", Revision: 3}
	r, err := svc.DesglosarMemoria(context.Background(), actor, f, d.CasosAtendidos, s)
	if err != nil || !r.Definitivas || !r.Fecha.Equal(informe.ConocidoEn) || r.Filas[0].Numero != "02-03-26" || !r.Filas[0].Ficha || fuente.llamadas != 0 || foto.revision != 3 {
		t.Fatal(r, err)
	}
	if audit.evento.Accion != "estadistica.memoria_desglose" || audit.evento.Resultado != "ok" {
		t.Fatal(audit.evento)
	}
	svc.Numeros = numerosDesglose{}
	r, err = svc.DesglosarMemoria(context.Background(), actor, f, d.CasosAtendidos, s)
	if err != nil || len(r.Filas) != 1 || r.Filas[0].Ficha {
		t.Fatal("foto sin acceso actual", r, err)
	}
	for _, perfil := range []string{"consulta", "administracion", "profesional_etf"} {
		actor.Perfiles = []string{perfil}
		actor.EquipoID = "55555555-5555-4555-8555-555555555555"
		r, err = svc.DesglosarMemoria(context.Background(), actor, f, d.CasosAtendidos, s)
		if !errors.Is(err, d.ErrPermisoMemoria) || len(r.Filas) != 0 || audit.evento.Resultado != "denegado" {
			t.Fatal(perfil, r, err)
		}
	}
	actor.Perfiles = []string{"coordinacion"}
	for _, fallo := range []string{"auditoria", "commit"} {
		dep := svc
		if fallo == "auditoria" {
			dep.Auditoria = &auditConsultaFallida{}
		} else {
			dep.Transacciones = txConsulta{fallar: true}
		}
		r, err := dep.DesglosarMemoria(context.Background(), actor, f, d.CasosAtendidos, s)
		if err == nil || len(r.Filas) != 0 || r.Cifra != 0 {
			t.Fatal("respuesta sin auditoría confirmada", fallo, r, err)
		}
	}
}

// Un cierre confirmado mientras la lectura espera no convierte un cálculo
// anterior al bloqueo en la foto definitiva.
type cierreConcurrenteDesglose struct {
	salida.Cierres
	reloj     *relojBloqueo
	cerradoEn time.Time
	foto      d.InformeMemoria
}

func (s cierreConcurrenteDesglose) BloquearMemorias(context.Context) error {
	s.reloj.ahora = s.cerradoEn.Add(time.Second)
	return nil
}
func (s cierreConcurrenteDesglose) EstadoMemoria(context.Context, d.FiltroMemoria, bool, string, time.Time) (d.EstadoMemoria, error) {
	return d.EstadoMemoria{Estado: "cerrada"}, nil
}
func (s cierreConcurrenteDesglose) RevisionConocida(_ context.Context, _ d.FiltroMemoria, conocido time.Time) (d.RevisionMemoria, bool, error) {
	if conocido.Before(s.cerradoEn) {
		return d.RevisionMemoria{}, false, nil
	}
	return d.RevisionMemoria{Informe: s.foto}, true, nil
}
func TestDesgloseCierreConcurrenteUsaFotoDespuesDelBloqueo(t *testing.T) {
	antes := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	reloj := &relojBloqueo{ahora: antes}
	f := d.FiltroMemoria{Anio: 2026, EquipoID: "33333333-3333-4333-8333-333333333333"}
	informe := func(n int) d.InformeMemoria {
		r := d.InformeMemoria{FiltroMemoria: f, ConocidoEn: antes, VersionDesglose: 1, Apartados: []d.ApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.Recuento{Familias: n}}}}
		for _, id := range []string{"44444444-4444-4444-8444-444444444444", "55555555-5555-4555-8555-555555555555"}[:n] {
			r.Desglose = append(r.Desglose, d.AporteMemoria{Apartado: d.CasosAtendidos, ExpedienteID: id, EquipoID: f.EquipoID, Total: d.Recuento{Familias: 1}})
		}
		return r
	}
	fuente := &fuenteDesglose{informe: informe(2)}
	svc := app.ServicioNavegacion{Fuente: fuente, Cierres: cierreConcurrenteDesglose{reloj: reloj, cerradoEn: antes.Add(time.Minute), foto: informe(1)}, Numeros: numerosDesglose{}, Transacciones: txConsulta{}, Auditoria: &auditCierre{}, Reloj: reloj}
	actor := d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"coordinacion"}}
	r, err := svc.DesglosarMemoria(context.Background(), actor, f, d.CasosAtendidos, d.SeleccionDesglose{Celda: -1, Medida: "familias"})
	if err != nil || !r.Definitivas || r.Cifra != 1 || len(r.Filas) != 1 || fuente.llamadas != 0 {
		t.Fatalf("cálculo vivo presentado como foto cerrada: %+v; fuente=%d; error=%v", r, fuente.llamadas, err)
	}
}
