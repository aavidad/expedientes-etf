package estadistica_test

import (
	"context"
	"encoding/json"
	"errors"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"testing"
	"time"
)

type cerradasConsulta struct {
	r        d.CifrasConsulta
	cerrada  bool
	llamadas int
}

func (s *cerradasConsulta) CifrasCerradas(context.Context, d.FiltroMemoria) (d.CifrasConsulta, bool, error) {
	s.llamadas++
	return s.r, s.cerrada, nil
}

type calculoConsulta struct {
	r        d.CifrasConsulta
	llamadas int
	actor    d.ActorMemoria
}

func (s *calculoConsulta) CalcularConsulta(_ context.Context, a d.ActorMemoria, _ d.FiltroMemoria, _ time.Time) (d.CifrasConsulta, error) {
	s.llamadas++
	s.actor = a
	return s.r, nil
}

type txConsulta struct{ fallar bool }

func (s txConsulta) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	if s.fallar {
		return errors.New("transaccion.fallo")
	}
	return nil
}

type auditConsultaFallida struct{ auditCierre }

type auditValidaConsulta struct{}

func (auditValidaConsulta) Confirmar(_ context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, e.Validar()
}
func (s auditValidaConsulta) RegistrarEnTransaccion(ctx context.Context, e auditoria.Datos) (auditoria.Evento, error) {
	return s.Confirmar(ctx, e)
}

func TestNavegacionYRechazosAjustanPrecisionDelRelojReal(t *testing.T) {
	reloj := &relojBloqueo{ahora: time.Date(2026, 10, 14, 10, 0, 0, 123456789, time.UTC)}
	nav := app.ServicioNavegacion{Fuente: &fuenteCierre{}, Transacciones: txConsulta{}, Auditoria: auditValidaConsulta{}, Reloj: reloj}
	actor := d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"coordinacion"}}
	if _, err := nav.ConsultarMemoria(context.Background(), actor, d.FiltroMemoria{Anio: 2026}); err != nil {
		t.Fatal("lectura con precisión nanosegundo", err)
	}
	actor.Perfiles = []string{"profesional_etf"}
	actor.EquipoID = "33333333-3333-4333-8333-333333333333"
	if _, err := nav.ConsultarMemoria(context.Background(), actor, d.FiltroMemoria{Anio: 2026, EquipoID: "44444444-4444-4444-8444-444444444444"}); !errors.Is(err, d.ErrPermisoMemoria) {
		t.Fatal("rechazo sin auditoría válida", err)
	}
	if err := nav.RechazarPeticionMemoria(context.Background(), actor); err != nil {
		t.Fatal("parámetros inválidos sin auditoría", err)
	}
}

func (auditConsultaFallida) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.fallo")
}
func TestConsultaFotoYCalculoNoElevanPerfilNiEntreganSinAuditoria(t *testing.T) {
	actor := d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"consulta"}}
	fecha := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	foto := &cerradasConsulta{r: d.CifrasConsulta{Anio: 2026, Fecha: fecha.Add(-time.Hour), Definitivas: true}, cerrada: true}
	motor := &calculoConsulta{r: d.CifrasConsulta{Anio: 2026, Fecha: fecha}}
	deps := app.DependenciasConsulta{Cerradas: foto, Calculo: motor, Transacciones: txConsulta{}, Auditoria: &auditCierre{}, Reloj: &relojBloqueo{ahora: fecha}}
	r, err := app.NuevoConsulta(deps).ConsultarCifras(context.Background(), actor, d.FiltroMemoria{Anio: 2026})
	if err != nil || !r.Definitivas || !r.Fecha.Equal(foto.r.Fecha) || motor.llamadas != 0 {
		t.Fatal("no respeta foto", r, err)
	}
	foto.cerrada = false
	r, err = app.NuevoConsulta(deps).ConsultarCifras(context.Background(), actor, d.FiltroMemoria{Anio: 2026})
	if err != nil || r.Definitivas || motor.llamadas != 1 || strings.Join(motor.actor.Perfiles, ",") != "consulta" {
		t.Fatal("cálculo o perfiles", r, err)
	}
	for _, fallo := range []string{"auditoria", "commit"} {
		t.Run(fallo, func(t *testing.T) {
			dep := deps
			if fallo == "auditoria" {
				dep.Auditoria = &auditConsultaFallida{}
			} else {
				dep.Transacciones = txConsulta{fallar: true}
			}
			r, err := app.NuevoConsulta(dep).ConsultarCifras(context.Background(), actor, d.FiltroMemoria{Anio: 2026})
			if err == nil || r.Anio != 0 || len(r.Apartados) != 0 {
				t.Fatal("entrega sin confirmar", r, err)
			}
		})
	}
	actor.Perfiles = []string{"administracion"}
	antes := foto.llamadas
	_, err = app.NuevoConsulta(deps).ConsultarCifras(context.Background(), actor, d.FiltroMemoria{Anio: 2026})
	if !errors.Is(err, d.ErrPermisoMemoria) || foto.llamadas != antes {
		t.Fatal("se leyó sin permiso", err)
	}
}
func TestConsultaTipoIndependienteNoArrastraTextoLibre(t *testing.T) {
	// Un campo futuro en el resultado completo no amplía la lista permitida.
	m := d.Memoria{Anio: 2026, Apartados: map[d.Apartado]*d.Resultado{d.CasosAtendidos: {Total: d.Recuento{Familias: 3}, Celdas: map[d.ClaveCelda]d.Celda{{Unidad: d.Familias}: {Numerador: 3, Poblacion: "Ana Ruiz vive en Calle Luna", VersionRegla: "Notas privadas del expediente", Porcentaje: &d.Porcentaje{Aviso: "Ana Ruiz"}}}, Avisos: map[string]int{"Ana Ruiz": 3}}}}
	b, err := json.Marshal(d.ProyectarCifras(m, ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []string{"Ana Ruiz", "Notas privadas", "Poblacion", "VersionRegla", "Fuentes", "Narrativas", "Composicion"} {
		if strings.Contains(string(b), valor) {
			t.Fatal("texto o campo filtrado", valor)
		}
	}
	if !strings.Contains(string(b), `"familias":3`) {
		t.Fatal("perdió las cifras", string(b))
	}
}

// El commit que importa para la entrega es el de auditoría, después del cálculo.
type txConsultaFallaConfirmacion struct{ llamadas int }

func (s *txConsultaFallaConfirmacion) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	s.llamadas++
	if err := fn(ctx); err != nil {
		return err
	}
	if s.llamadas == 2 {
		return errors.New("transaccion.fallo")
	}
	return nil
}

func TestConsultaNoEntregaCalculoSiFallaCommitAuditoria(t *testing.T) {
	actor := d.ActorMemoria{UsuarioID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{"consulta"}}
	fecha := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	tx := &txConsultaFallaConfirmacion{}
	motor := &calculoConsulta{r: d.CifrasConsulta{Anio: 2026, Fecha: fecha}}
	svc := app.NuevoConsulta(app.DependenciasConsulta{Cerradas: &cerradasConsulta{}, Calculo: motor, Transacciones: tx, Auditoria: &auditCierre{}, Reloj: &relojBloqueo{ahora: fecha}})
	cifras, err := svc.ConsultarCifras(context.Background(), actor, d.FiltroMemoria{Anio: 2026})
	if !errors.Is(err, d.ErrDisponibleMemoria) || cifras.Anio != 0 || motor.llamadas != 1 || tx.llamadas != 2 {
		t.Fatal("entrega sin commit de auditoría", cifras, err, tx.llamadas)
	}
}
