package expediente_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type relojDecision struct{ fecha time.Time }

func (r relojDecision) Ahora() time.Time { return r.fecha }
func prepararDecision(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, dominio.Aceptacion) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps.Decisiones = &adaptador.Repositorio{}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// En UTC aún es el día anterior; Madrid determina el hecho administrativo.
	deps.Reloj = relojDecision{time.Date(2027, 1, 1, 0, 30, 0, 0, zona)}
	return b, ctx, deps, a, p
}
func comprobarHistoriaDecision(t *testing.T, b *postgres.Base, ctx context.Context, a dominio.Actor, id string, hitos, cambios, episodios int) {
	t.Helper()
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		for tabla, esperado := range map[string]int{"hito": hitos, "estado_cambio": cambios, "episodio": episodios} {
			var n int
			if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla+" WHERE expediente_id=$1", id).Scan(&n); err != nil {
				return err
			}
			if n != esperado {
				t.Fatalf("%s: %d, esperado %d", tabla, n, esperado)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecisionEsperaAceptacionFechaRealYNumeracion(t *testing.T) {
	b, ctx, deps, a, p := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	s := aplicacion.Nuevo(deps)
	antes := contador(t, b, ctx, a, p)
	r, err := s.DejarEnEspera(ctx, a, d.ID, "espera", dominio.Espera{Version: 1})
	if err != nil || r.Estado != dominio.ListaEspera || r.Fecha != "2027-01-01" {
		t.Fatal(r, err)
	}
	if contador(t, b, ctx, a, p) != antes {
		t.Fatal("espera consume número")
	}
	deps.Reloj = relojDecision{time.Date(2027, 1, 2, 0, 30, 0, 0, deps.Reloj.Ahora().Location())}
	s = aplicacion.Nuevo(deps)
	p.Version = 2
	if _, err := s.Aceptar(ctx, a, d.ID, "fecha-antigua", p); !errors.Is(err, dominio.ErrFechaEspera) {
		t.Fatal(err)
	}
	p.FechaActa = "2027-01-01"
	if _, err := s.Aceptar(ctx, a, d.ID, "fecha-de-espera", p); !errors.Is(err, dominio.ErrFechaEspera) {
		t.Fatal(err)
	}
	p.FechaActa = "2027-01-02"
	antesReal := contador(t, b, ctx, a, p)
	aceptada, err := s.Aceptar(ctx, a, d.ID, "aceptacion-real", p)
	if err != nil || aceptada.Anio != 2027 || aceptada.Correlativo != antesReal+1 || aceptada.FechaActa != "2027-01-02" || aceptada.Version != 3 {
		t.Fatal(aceptada, err)
	}
	repetida, err := s.Aceptar(ctx, a, d.ID, "aceptacion-real", p)
	if err != nil || repetida != aceptada {
		t.Fatal(repetida, err)
	}
	if contador(t, b, ctx, a, p) != antesReal+1 {
		t.Fatal("repetición consume número")
	}
	comprobarHistoriaDecision(t, b, ctx, a, d.ID, 2, 2, 1)
	ficha, err := s.Consultar(ctx, a, d.ID)
	if err != nil || ficha.FechaEspera == nil || *ficha.FechaEspera != "2027-01-01" {
		t.Fatal(ficha, err)
	}
	if _, err := s.NoAceptar(ctx, a, d.ID, "no-aceptar-aceptada", dominio.NoAceptacion{Version: 3, Fecha: "2027-01-01", Motivo: "La atención corresponde a otro recurso."}); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	if _, err := s.DejarEnEspera(ctx, a, d.ID, "espera-aceptada", dominio.Espera{Version: 3}); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	t.Log("Espera 01/01 sin número; aceptación 02/01 Madrid con contador 2027; idempotencia; aceptada rechaza espera y no aceptación")
}

func TestDecisionNoAceptacionDirectaYDesdeEsperaFinal(t *testing.T) {
	b, ctx, deps, a, p := prepararDecision(t)
	for _, desdeEspera := range []bool{false, true} {
		d := subir(t, ctx, deps, a)
		s := aplicacion.Nuevo(deps)
		version, hitos := 1, 1
		if desdeEspera {
			if _, err := s.DejarEnEspera(ctx, a, d.ID, "esperar-"+d.ID, dominio.Espera{Version: 1}); err != nil {
				t.Fatal(err)
			}
			version, hitos = 2, 2
		}
		antes := contador(t, b, ctx, a, p)
		peticion := dominio.NoAceptacion{Version: version, Fecha: "2027-01-01", Motivo: "El centro confirma que la familia recibe atención del recurso competente."}
		r, err := s.NoAceptar(ctx, a, d.ID, "rechazo-"+d.ID, peticion)
		if err != nil || r.Subprograma != "RIESGO_SOCIAL" || r.Estado != dominio.NoAceptada || r.Version != version+1 {
			t.Fatal(r, err)
		}
		repetida, err := s.NoAceptar(ctx, a, d.ID, "rechazo-"+d.ID, peticion)
		if err != nil || repetida != r {
			t.Fatal(repetida, err)
		}
		ficha, err := s.Consultar(ctx, a, d.ID)
		if err != nil || ficha.Numero != nil || ficha.EquipoID != nil || ficha.NoAceptacion == nil || ficha.NoAceptacion.Motivo != peticion.Motivo {
			t.Fatal(ficha, err)
		}
		if contador(t, b, ctx, a, p) != antes {
			t.Fatal("no aceptación consume número")
		}
		comprobarHistoriaDecision(t, b, ctx, a, d.ID, hitos, hitos, 0)
		peticion.Version = r.Version
		if _, err := s.NoAceptar(ctx, a, d.ID, "nuevo-rechazo-"+d.ID, peticion); !errors.Is(err, dominio.ErrEstado) {
			t.Fatal(err)
		}
		if _, err := s.DejarEnEspera(ctx, a, d.ID, "otra-espera-"+d.ID, dominio.Espera{Version: r.Version}); !errors.Is(err, dominio.ErrEstado) {
			t.Fatal(err)
		}
		p = conMenores(p, d)
		p.Version = r.Version
		if _, err := s.Aceptar(ctx, a, d.ID, "aceptar-final-"+d.ID, p); !errors.Is(err, dominio.ErrEstado) {
			t.Fatal(err)
		}
	}
	contenido, err := os.ReadFile("../../anexos/pdf/testdata/anexo4.pdf")
	if err != nil {
		t.Fatal(err)
	}
	s := aplicacion.Nuevo(deps)
	d, err := s.Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.pdf", Contenido: contenido})
	if err != nil {
		t.Fatal(err)
	}
	d = confirmarLectura(t, ctx, deps, a, d)
	if _, err := s.DejarEnEspera(ctx, a, d.ID, "spm-espera", dominio.Espera{Version: 1}); !errors.Is(err, dominio.ErrOrigenEspera) {
		t.Fatal(err)
	}
	if _, err := s.NoAceptar(ctx, a, d.ID, "spm-sin-subprograma", dominio.NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "La derivación corresponde a otro recurso."}); !errors.Is(err, dominio.ErrSubprograma) {
		t.Fatal(err)
	}
	r, err := s.NoAceptar(ctx, a, d.ID, "spm-no-aceptar", dominio.NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "La derivación corresponde a otro recurso.", Subprograma: "REUNIFICACION"})
	if err != nil || r.Subprograma != "REUNIFICACION" {
		t.Fatal(r, err)
	}
	t.Log("No aceptación directa y desde espera: final, sin número, motivo recuperable, Riesgo por defecto; Anexo 4 exige subprograma y no permite espera")
}

func TestDecisionConcurrenciaIdempotenciaYAuditoria(t *testing.T) {
	b, ctx, deps, a, _ := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	s := aplicacion.Nuevo(deps)
	const n = 12
	resultados := make([]dominio.ResultadoDecision, n)
	fallos := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resultados[i], fallos[i] = s.DejarEnEspera(ctx, a, d.ID, "misma-espera", dominio.Espera{Version: 1})
		}(i)
	}
	wg.Wait()
	for i := range n {
		if fallos[i] != nil || resultados[i] != resultados[0] {
			t.Fatal(i, resultados[i], fallos[i])
		}
	}
	comprobarHistoriaDecision(t, b, ctx, a, d.ID, 1, 1, 0)
	if _, err := s.DejarEnEspera(ctx, a, d.ID, "misma-espera", dominio.Espera{Version: 2}); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	if _, err := s.NoAceptar(ctx, a, d.ID, "misma-espera", dominio.NoAceptacion{Version: 2, Fecha: "2027-01-01", Motivo: "El caso corresponde a otro recurso."}); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	// Repetir la espera al día siguiente conserva la fecha del primer recibo.
	deps.Reloj = relojDecision{time.Date(2027, 1, 2, 0, 0, 0, 0, deps.Reloj.Ahora().Location())}
	repetida, err := aplicacion.Nuevo(deps).DejarEnEspera(ctx, a, d.ID, "misma-espera", dominio.Espera{Version: 1})
	if err != nil || repetida != resultados[0] {
		t.Fatal(repetida, err)
	}
	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	eventos := 0
	for _, ev := range exportacion.Eventos {
		if ev.ExpedienteID == d.ID && ev.Accion == "expediente.dejar_lista_espera" {
			eventos++
		}
	}
	if eventos != 1 {
		t.Fatal("auditoría repetida", eventos)
	}
	// Dos decisiones incompatibles, con claves distintas, sólo pueden confirmar una.
	d = subir(t, ctx, deps, a)
	var e1, e2 error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, e1 = s.DejarEnEspera(ctx, a, d.ID, "carrera-espera", dominio.Espera{Version: 1})
	}()
	go func() {
		defer wg.Done()
		_, e2 = s.NoAceptar(ctx, a, d.ID, "carrera-rechazo", dominio.NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "El centro ha retirado la derivación."})
	}()
	wg.Wait()
	if (e1 == nil) == (e2 == nil) {
		t.Fatal(e1, e2)
	}
	conflicto := e1
	if conflicto == nil {
		conflicto = e2
	}
	var c dominio.Conflicto
	if !errors.As(conflicto, &c) || c.VersionActual != 2 {
		t.Fatal(conflicto)
	}
	comprobarHistoriaDecision(t, b, ctx, a, d.ID, 1, 1, 0)
	t.Log("12 repeticiones concurrentes: un hito y una auditoría; clave distinta de contenido/ruta rechazada; dos decisiones: un éxito y 409 versión 2")
}

func TestDecisionRollbackAuditoriaYLista(t *testing.T) {
	b, ctx, deps, a, _ := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	for _, estado := range []string{dominio.ListaEspera, dominio.NoAceptada} {
		s := aplicacion.Nuevo(fallo)
		var err error
		if estado == dominio.ListaEspera {
			_, err = s.DejarEnEspera(ctx, a, d.ID, "fallo-espera", dominio.Espera{Version: 1})
		} else {
			_, err = s.NoAceptar(ctx, a, d.ID, "fallo-rechazo", dominio.NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "La familia recibe atención de otro servicio."})
		}
		if !errors.Is(err, dominio.ErrNoDisponible) {
			t.Fatal(err)
		}
		ficha, err := aplicacion.Nuevo(deps).Consultar(ctx, a, d.ID)
		if err != nil || ficha.Version != 1 || ficha.Estado != dominio.DerivacionRecibida {
			t.Fatal(ficha, err)
		}
		comprobarHistoriaDecision(t, b, ctx, a, d.ID, 0, 0, 0)
	}
	s := aplicacion.Nuevo(deps)
	if _, err := s.DejarEnEspera(ctx, a, d.ID, "fallo-espera", dominio.Espera{Version: 1}); err != nil {
		t.Fatal(err)
	}
	if filas, err := aplicacion.Nuevo(fallo).ListarEspera(ctx, a, entrada.FiltroEspera{Limite: 200}); !errors.Is(err, dominio.ErrNoDisponible) || filas != nil {
		t.Fatal(filas, err)
	}
	filas, err := s.ListarEspera(ctx, a, entrada.FiltroEspera{Limite: 200})
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	for i, f := range filas {
		if f.ID == d.ID {
			encontrada = true
		}
		if i > 0 && (f.FechaRecepcion.Before(filas[i-1].FechaRecepcion) || (f.FechaRecepcion.Equal(filas[i-1].FechaRecepcion) && f.ID < filas[i-1].ID)) {
			t.Fatal("orden incorrecto")
		}
	}
	if !encontrada {
		t.Fatal("espera ausente")
	}
	pagina, err := s.ListarEspera(ctx, a, entrada.FiltroEspera{Limite: 1, Desplazamiento: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(filas) > 1 && !reflect.DeepEqual(pagina, filas[1:2]) {
		t.Fatal("paginación distinta")
	}
	t.Log("Auditoría fallida revierte estado, versión, hitos e idempotencia; lista no entrega datos sin auditar; orden y paginación")
}

func TestDecisionRLSSinEquipoYBarrerasSQL(t *testing.T) {
	b, ctx, deps, a, _ := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	s := aplicacion.Nuevo(deps)
	if _, err := s.DejarEnEspera(ctx, a, d.ID, "rls-espera", dominio.Espera{Version: 1}); err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"", "consulta", "administracion", "profesional_etf"} {
		actor := dominio.Actor{UsuarioID: a.UsuarioID}
		if perfil != "" {
			actor.Perfiles = []string{perfil}
		}
		if _, err := s.ListarEspera(ctx, actor, entrada.FiltroEspera{Limite: 20}); !errors.Is(err, dominio.ErrAcceso) {
			t.Fatal(perfil, err)
		}
		if _, err := s.NoAceptar(ctx, actor, d.ID, "sin-permiso-"+perfil, dominio.NoAceptacion{Version: 2, Fecha: "2027-01-01", Motivo: "El caso corresponde a otro servicio."}); !errors.Is(err, dominio.ErrAcceso) {
			t.Fatal(perfil, err)
		}
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"expediente", "hito", "estado_cambio"} {
				col := "expediente_id"
				if tabla == "expediente" {
					col = "id"
				}
				var n int
				if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla+" WHERE "+col+"=$1", d.ID).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					t.Fatal(perfil, tabla, n)
				}
			}
			tag, err := e.Exec(ctx, `UPDATE expediente.expediente SET estado='no_aceptada' WHERE id=$1`, d.ID)
			if err == nil && tag.RowsAffected() != 0 {
				t.Fatal("RLS permite escritura", perfil)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Incluso Coordinación queda sujeta al estado final al escribir directamente.
	if _, err := s.NoAceptar(ctx, a, d.ID, "rls-final", dominio.NoAceptacion{Version: 2, Fecha: "2027-01-01", Motivo: "El recurso competente ha asumido la atención."}); err != nil {
		t.Fatal(err)
	}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `UPDATE expediente.expediente SET causa_no_aceptacion=$2 WHERE id=$1`, d.ID, "Modificación posterior")
		return err
	})
	if err == nil {
		t.Fatal("SQL modifica estado final")
	}
	contenido, err := os.ReadFile("../../anexos/pdf/testdata/anexo4.pdf")
	if err != nil {
		t.Fatal(err)
	}
	spm, err := s.Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.pdf", Contenido: contenido})
	if err != nil {
		t.Fatal(err)
	}
	spm = confirmarLectura(t, ctx, deps, a, spm)
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `UPDATE expediente.expediente SET estado='lista_espera',fecha_lista_espera='2027-01-01',version=2,modificado_por=$2,modificado_en=now() WHERE id=$1`, spm.ID, a.UsuarioID)
		return err
	})
	if err == nil {
		t.Fatal("SQL pone Anexo 4 en espera")
	}
	t.Log(fmt.Sprintf("RLS: %d perfiles sin lectura/escritura; SQL bloquea no aceptación final y espera de Anexo 4", 4))
}

func TestDecisionNoAceptacionIdempotenciaConcurrente(t *testing.T) {
	b, ctx, deps, a, p := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	s := aplicacion.Nuevo(deps)
	antes := contador(t, b, ctx, a, p)
	peticion := dominio.NoAceptacion{Version: 1, Fecha: "2027-01-01", Motivo: "El centro ha retirado la derivación tras coordinar otro recurso."}
	const n = 12
	resultados := make([]dominio.ResultadoDecision, n)
	fallos := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resultados[i], fallos[i] = s.NoAceptar(ctx, a, d.ID, "rechazo-concurrente", peticion)
		}(i)
	}
	wg.Wait()
	for i := range n {
		if fallos[i] != nil || resultados[i] != resultados[0] {
			t.Fatal(i, resultados[i], fallos[i])
		}
	}
	comprobarHistoriaDecision(t, b, ctx, a, d.ID, 1, 1, 0)
	if contador(t, b, ctx, a, p) != antes {
		t.Fatal("no aceptación consume número")
	}
	peticion.Motivo = "La familia recibe atención del recurso competente."
	if _, err := s.NoAceptar(ctx, a, d.ID, "rechazo-concurrente", peticion); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	eventos := 0
	for _, ev := range exportacion.Eventos {
		if ev.ExpedienteID == d.ID && ev.Accion == "expediente.no_aceptar_derivacion" {
			eventos++
		}
	}
	if eventos != 1 {
		t.Fatal("auditoría repetida", eventos)
	}
	t.Log("12 no aceptaciones concurrentes: mismo recibo, un hito, una auditoría y ningún número; motivo distinto con misma clave rechazado")
}
