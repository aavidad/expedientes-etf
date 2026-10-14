package expediente

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type repositorioBandejaPrueba struct{ llamadas int }

type avisosBandejaPrueba struct{ *entornoAvisos }

func (a avisosBandejaPrueba) DatosAvisosEquipo(_ context.Context, ids []string, equipo, _ string) ([]d.DatosAvisos, error) {
	r := make([]d.DatosAvisos, len(ids))
	for i, id := range ids {
		r[i] = d.DatosAvisos{ID: id, EquipoID: equipo, Estado: d.Aceptada, Aceptacion: "2026-10-13"}
	}
	return r, nil
}

func (r *repositorioBandejaPrueba) ListarEquipo(context.Context, d.FiltroEquipo) (d.PaginaEquipo, error) {
	r.llamadas++
	return d.PaginaEquipo{Filas: []d.ResumenEquipo{{ID: "33333333-3333-4333-8333-333333333333", EquipoID: "22222222-2222-4222-8222-222222222222", Numero: "02-14-26", Familia: "O. S."}}}, nil
}
func (*repositorioBandejaPrueba) FichaEquipo(context.Context, string) (d.FichaEquipo, error) {
	return d.FichaEquipo{}, nil
}

type txBandejaPrueba struct{ fallo bool }

func (tx txBandejaPrueba) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	if tx.fallo {
		return errors.New("fallo.commit")
	}
	return nil
}

type registroBandejaPrueba struct {
	fallo   bool
	apuntes []auditoria.Datos
}

func (r *registroBandejaPrueba) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, nil
}
func (r *registroBandejaPrueba) RegistrarEnTransaccion(_ context.Context, a auditoria.Datos) (auditoria.Evento, error) {
	r.apuntes = append(r.apuntes, a)
	if r.fallo {
		return auditoria.Evento{}, errors.New("fallo.auditoria")
	}
	return auditoria.Evento{}, nil
}

type relojBandejaPrueba struct{}

func (relojBandejaPrueba) Ahora() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }

func TestLecturaEquipoNoEntregaDatosSinAuditoriaOCommit(t *testing.T) {
	a := d.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", EquipoID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"profesional_etf"}}
	for _, caso := range []struct {
		nombre            string
		auditoria, commit bool
	}{{"correcto", false, false}, {"auditoria", true, false}, {"commit", false, true}} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := &repositorioBandejaPrueba{}
			audit := &registroBandejaPrueba{fallo: caso.auditoria}
			_, av, _ := prepararAvisos(t)
			s := LecturaEquipo{Avisos: avisosBandejaPrueba{av}, Asignacion: av, Repositorio: r, Registro: audit, Reloj: relojBandejaPrueba{}, Transacciones: txBandejaPrueba{fallo: caso.commit}}
			p, err := s.ListarEquipo(context.Background(), a, d.FiltroEquipo{Limite: 50})
			if caso.auditoria || caso.commit {
				if !errors.Is(err, d.ErrNoDisponible) || len(p.Filas) != 0 {
					t.Fatal("datos sin confirmación", p, err)
				}
			} else if err != nil || len(p.Filas) != 1 || audit.apuntes[0].Accion != "equipo.bandeja_consultada.filas_1" {
				t.Fatal(p, err, audit.apuntes)
			}
		})
	}
}

func TestLecturaEquipoNoOcultaAvisosSiFaltaSuPuerto(t *testing.T) {
	a := d.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", EquipoID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"profesional_etf"}}
	s := LecturaEquipo{Repositorio: &repositorioBandejaPrueba{}, Registro: &registroBandejaPrueba{}, Reloj: relojBandejaPrueba{}, Transacciones: txBandejaPrueba{}}
	p, err := s.ListarEquipo(context.Background(), a, d.FiltroEquipo{Limite: 50})
	if !errors.Is(err, d.ErrNoDisponible) || len(p.Filas) != 0 {
		t.Fatal("datos sin fuente de avisos", p, err)
	}
}

func TestLecturaEquipoDeniegaConsultaAdministracionYSinEquipo(t *testing.T) {
	for _, perfil := range []string{"consulta", "administracion", "profesional_etf"} {
		r := &repositorioBandejaPrueba{}
		s := LecturaEquipo{Repositorio: r}
		_, err := s.ListarEquipo(context.Background(), d.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", Perfiles: []string{perfil}}, d.FiltroEquipo{Limite: 50})
		if !errors.Is(err, d.ErrAcceso) || r.llamadas != 0 {
			t.Fatal(perfil, err, r.llamadas)
		}
	}
}
