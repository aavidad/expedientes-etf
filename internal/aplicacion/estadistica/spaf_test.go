package estadistica_test

import (
	"context"
	"errors"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

const usuarioSPAF = "11111111-1111-4111-8111-111111111111"
const equipoSPAF = "22222222-2222-4222-8222-222222222222"
const otroEquipoSPAF = "33333333-3333-4333-8333-333333333333"

type claveTxSPAF struct{}
type transaccionesSPAF struct {
	identidad   plataforma.Identidad
	llamadas    int
	errorCommit bool
}

func (t *transaccionesSPAF) Ejecutar(ctx context.Context, a plataforma.Identidad, fn func(context.Context) error) error {
	t.llamadas++
	t.identidad = a
	if err := fn(context.WithValue(ctx, claveTxSPAF{}, true)); err != nil {
		return err
	}
	if t.errorCommit {
		return errors.New("persistencia.error")
	}
	return nil
}

type auditoriaSPAF struct {
	datos                     []audit.Datos
	fallos                    []audit.Datos
	errorRegistro, errorFallo bool
}

func (a *auditoriaSPAF) RegistrarEnTransaccion(ctx context.Context, dato audit.Datos) (audit.Evento, error) {
	if ctx.Value(claveTxSPAF{}) != true {
		return audit.Evento{}, errors.New("auditoria.sin_transaccion")
	}
	if a.errorRegistro {
		return audit.Evento{}, errors.New("auditoria.error")
	}
	a.datos = append(a.datos, dato)
	return audit.Evento{}, nil
}
func (a *auditoriaSPAF) Confirmar(ctx context.Context, dato audit.Datos) (audit.Evento, error) {
	if ctx.Value(claveTxSPAF{}) != nil || a.errorFallo {
		return audit.Evento{}, errors.New("auditoria.error")
	}
	a.fallos = append(a.fallos, dato)
	return audit.Evento{}, nil
}

type relojSPAF struct{}

func (relojSPAF) Ahora() time.Time {
	return time.Date(2026, 2, 1, 0, 30, 0, 0, time.FixedZone("", 3600))
}

// lecturaMemoriaSPAF implementa el puerto pendiente de F2.01 con una foto
// inventada. Rechaza consultas sin el contexto transaccional de la aplicación.
type lecturaMemoriaSPAF struct {
	llamadas     int
	filtro       d.FiltroSPAF
	ajeno, falla bool
}

func (l *lecturaMemoriaSPAF) ConsultarSPAF(ctx context.Context, f d.FiltroSPAF, corte, conocido time.Time) ([]d.ExpedienteSPAF, d.Historia[d.ReglasSPAF], error) {
	l.llamadas++
	l.filtro = f
	if ctx.Value(claveTxSPAF{}) != true || l.falla || corte.Day() != 31 || conocido.IsZero() {
		return nil, nil, errors.New("lectura.error")
	}
	equipo := f.EquipoID
	if l.ajeno {
		equipo = otroEquipoSPAF
	}
	aceptacion := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	exps := []d.ExpedienteSPAF{{ID: "44444444-4444-4444-8444-444444444444", EquipoID: equipo, Numero: "2-001-2026", Episodios: []d.Episodio{{ID: "episodio-1", Apertura: aceptacion, Aceptacion: aceptacion}}}}
	reglas := d.Historia[d.ReglasSPAF]{{ID: "reglas-1", Vigencia: d.Vigencia{Desde: aceptacion}, RegistradoEn: aceptacion, Dato: d.Dato[d.ReglasSPAF]{Conocido: true}}}
	return exps, reglas, nil
}

func TestSPAFLecturaAutorizadaCalculaYConfirmaAuditoria(t *testing.T) {
	for _, caso := range []struct {
		perfiles                                    []string
		equipoActor, equipoPeticion, equipoEsperado string
	}{
		{[]string{"profesional_etf"}, equipoSPAF, "", equipoSPAF},
		{[]string{"coordinacion"}, "", otroEquipoSPAF, otroEquipoSPAF},
		{[]string{"administracion", "coordinacion"}, "", equipoSPAF, equipoSPAF},
	} {
		l, tx, a := &lecturaMemoriaSPAF{}, &transaccionesSPAF{}, &auditoriaSPAF{}
		s := app.NuevoSPAF(app.DependenciasSPAF{Lectura: l, Transacciones: tx, Auditoria: a, Reloj: relojSPAF{}})
		actor := d.ActorSPAF{UsuarioID: usuarioSPAF, EquipoID: caso.equipoActor, Perfiles: caso.perfiles}
		got, err := s.ConsultarSPAF(context.Background(), actor, d.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: caso.equipoPeticion})
		if err != nil || got.Resumen.Altas != 1 || got.Resumen.Casos != 1 || l.filtro.EquipoID != caso.equipoEsperado || tx.identidad.UsuarioID != usuarioSPAF || tx.identidad.EquipoID != caso.equipoActor || len(a.datos) != 1 || a.datos[0].EntidadID != caso.equipoEsperado || len(a.fallos) != 0 {
			t.Fatal(got, err, tx, a)
		}
	}
}

func TestSPAFDeniegaAntesDeLeerYAuditaFallo(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		actor    d.ActorSPAF
		filtro   d.FiltroSPAF
		esperado error
	}{
		{"otro equipo", d.ActorSPAF{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: otroEquipoSPAF}, d.ErrPermisoSPAF},
		{"sin equipo", d.ActorSPAF{UsuarioID: usuarioSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroSPAF{Anio: 2026, Mes: 1}, d.ErrPermisoSPAF},
		{"consulta", d.ActorSPAF{UsuarioID: usuarioSPAF, Perfiles: []string{"consulta"}}, d.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: equipoSPAF}, d.ErrPermisoSPAF},
		{"administración", d.ActorSPAF{UsuarioID: usuarioSPAF, Perfiles: []string{"administracion"}}, d.FiltroSPAF{Anio: 2026, Mes: 1, EquipoID: equipoSPAF}, d.ErrPermisoSPAF},
		{"provincial", d.ActorSPAF{UsuarioID: usuarioSPAF, Perfiles: []string{"coordinacion"}}, d.FiltroSPAF{Anio: 2026, Mes: 1}, d.ErrEntrada},
		{"mes no vencido", d.ActorSPAF{UsuarioID: usuarioSPAF, Perfiles: []string{"coordinacion"}}, d.FiltroSPAF{Anio: 2026, Mes: 2, EquipoID: equipoSPAF}, d.ErrEntrada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			l, tx, a := &lecturaMemoriaSPAF{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			s := app.NuevoSPAF(app.DependenciasSPAF{Lectura: l, Transacciones: tx, Auditoria: a, Reloj: relojSPAF{}})
			got, err := s.ConsultarSPAF(context.Background(), caso.actor, caso.filtro)
			if !errors.Is(err, caso.esperado) || got.Filas != nil || l.llamadas != 0 || tx.llamadas != 0 || len(a.fallos) != 1 {
				t.Fatal(got, err, l, tx, a)
			}
		})
	}
}

func TestSPAFNoEntregaDatosSiFallaFuenteAuditoriaOCommit(t *testing.T) {
	for _, caso := range []string{"fuente ausente", "lectura", "equipo ajeno", "auditoría", "commit", "auditoría de fallo"} {
		t.Run(caso, func(t *testing.T) {
			l, tx, a := &lecturaMemoriaSPAF{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			dep := app.DependenciasSPAF{Lectura: l, Transacciones: tx, Auditoria: a, Reloj: relojSPAF{}}
			switch caso {
			case "fuente ausente":
				dep.Lectura = nil
			case "lectura":
				l.falla = true
			case "equipo ajeno":
				l.ajeno = true
			case "auditoría":
				a.errorRegistro = true
			case "commit":
				tx.errorCommit = true
			case "auditoría de fallo":
				l.falla = true
				a.errorFallo = true
			}
			got, err := app.NuevoSPAF(dep).ConsultarSPAF(context.Background(), d.ActorSPAF{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroSPAF{Anio: 2026, Mes: 1})
			if !errors.Is(err, d.ErrDisponibleSPAF) || got.Filas != nil || got.Resumen.Casos != 0 {
				t.Fatal("datos entregados sin confirmación", got, err)
			}
		})
	}
}
