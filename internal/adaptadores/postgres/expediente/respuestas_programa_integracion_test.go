package expediente_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	memoria "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dom "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestRESP1EsperaMemoriaRLSCorteYConocimiento(t *testing.T) {
	b, ctx, deps, actor, _ := prepararDecision(t)
	recibida := subir(t, ctx, deps, actor)
	equipo := recibida.Datos.UbicacionConfirmada.EquipoID
	entrada := deps.Reloj.Ahora()
	s := app.Nuevo(deps)
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err := app.Nuevo(fallo).DejarEnEspera(ctx, actor, recibida.ID, "resp1-rollback-"+recibida.ID, dom.Espera{Version: recibida.Version}); !errors.Is(err, dom.ErrNoDisponible) {
		t.Fatal("fallo de auditoría", err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.espera_memoria WHERE expediente_id=$1 AND desde IS NOT NULL`, recibida.ID).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Fatal("proyección de espera no revertida")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	espera, err := s.DejarEnEspera(ctx, actor, recibida.ID, "memoria-espera-"+recibida.ID, dom.Espera{Version: recibida.Version})
	if err != nil {
		t.Fatal(err)
	}
	lector := &exp.LecturaEstadistica{}
	corte := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	comprobar := func(base *postgres.Base, actor plataforma.Identidad, lector *exp.LecturaEstadistica, filtro string, conocido time.Time, visible bool) {
		t.Helper()
		err := base.Ejecutar(ctx, actor, func(ctx context.Context) error {
			datos, err := lector.ConsultarMemoria(ctx, filtro, corte, conocido)
			if err != nil {
				return err
			}
			encontrados := 0
			for _, x := range datos {
				if x.ID == recibida.ID {
					encontrados++
					if x.EquipoID != equipo || len(x.Episodios) != 0 {
						t.Fatal("espera asigna equipo definitivo o aceptación", x)
					}
					reglas, err := (reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria"}).ReglasMemoria(ctx, 2027, conocido)
					if err != nil {
						return err
					}
					m, err := d.Calcular(d.Entrada{Anio: 2027, FechaCalculo: corte, ConocidoEn: conocido, Reglas: reglas, Expedientes: []d.Expediente{x}})
					if err != nil {
						return err
					}
					if m.Apartados[d.Pendientes].Total.Familias != 1 || m.Apartados[d.CasosAtendidos].Total.Familias != 0 {
						t.Fatal("D2c", m.Apartados[d.Pendientes].Total)
					}
				}
			}
			if (encontrados == 1) != visible || encontrados > 1 {
				t.Fatalf("visibilidad de espera: %d, esperado %v", encontrados, visible)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, caso := range []struct {
		perfil, equipo string
		visible        bool
	}{{"coordinacion", "", true}, {"profesional_etf", equipo, true}, {"profesional_etf", "00000000-0000-4000-8000-000000000012", false}, {"consulta", "", false}, {"administracion", "", false}, {"", "", false}} {
		identidad := plataforma.Identidad{}
		if caso.perfil != "" {
			identidad = plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: []string{caso.perfil}, EquipoID: caso.equipo}
		}
		comprobar(b, identidad, lector, "", entrada.Add(time.Second), caso.visible)
	}
	coord := plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}
	comprobar(b, coord, lector, equipo, entrada.Add(-time.Microsecond), false)
	deps.Reloj = relojDecision{entrada.AddDate(0, 0, 1)}
	_, err = app.Nuevo(deps).NoAceptar(ctx, actor, recibida.ID, "memoria-fin-"+recibida.ID, dom.NoAceptacion{Version: espera.Version, Fecha: "2027-01-02", Motivo: "La familia recibe atención de otro recurso."})
	if err != nil {
		t.Fatal(err)
	}
	comprobar(b, coord, lector, equipo, entrada.AddDate(0, 0, 2), true)
	corte = corte.AddDate(0, 0, 1)
	comprobar(b, coord, lector, equipo, entrada.AddDate(0, 0, 2), false)
	t.Log("RESP1-ESPERA-OK: apartado 6, equipo/provincia, RLS, corte y conocimiento; sin aceptación ni número")
}

func TestRESP1EsperaAvisosMemoriaCerradaYEquipoDistinto(t *testing.T) {
	b, ctx, deps, actor, p := prepararDecision(t)
	deps.Reloj = relojMemoria{}
	recibida := subir(t, ctx, deps, actor)
	equipoAntes := recibida.Datos.UbicacionConfirmada.EquipoID
	p = conMenores(p, recibida)
	var equipoDespues string
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE id<>$1 ORDER BY codigo LIMIT 1`, equipoAntes).Scan(&equipoDespues)
	})
	if err != nil {
		t.Fatal(err)
	}
	persistencia := &repo.Repositorio{}
	fuente := memoria.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: persistencia}
	registro := postgres.NuevoRegistroAuditoria(b)
	svc := memoria.NuevoCierre(memoria.DependenciasCierre{Repositorio: persistencia, Fuente: fuente, Transacciones: b, Auditoria: registro, Reloj: relojMemoria{}})
	a := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}
	filtros := []d.FiltroMemoria{{Anio: 2026, EquipoID: equipoAntes}, {Anio: 2026, EquipoID: equipoDespues}, {Anio: 2026}}
	cerrar := func(f d.FiltroMemoria) {
		t.Helper()
		estado, err := svc.EstadoMemoria(ctx, a, f)
		if err != nil {
			t.Fatal(err)
		}
		if estado.Estado == "cerrada" {
			r, err := svc.ReabrirMemoria(ctx, a, f, "resp1-reabrir-"+recibida.ID+"-"+f.EquipoID+"-"+time.Now().Format("150405.000000000"), d.ActoMemoria{Version: estado.Version, Motivo: "Revisar los casos pendientes de atender."})
			if err != nil {
				t.Fatal(err)
			}
			estado = r.EstadoMemoria
		}
		_, err = svc.CerrarMemoria(ctx, a, f, "resp1-cerrar-"+recibida.ID+"-"+f.EquipoID+"-"+time.Now().Format("150405.000000000"), d.ActoMemoria{Version: estado.Version})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range filtros {
		cerrar(f)
	}
	deps.RegistroTransaccional = memoria.RegistroConAvisos{Registro: registro, Repositorio: persistencia, Fuente: fuente, Equipos: &exp.LecturaEstadistica{}}
	espera, err := app.Nuevo(deps).DejarEnEspera(ctx, actor, recibida.ID, "resp1-avisos-espera-"+recibida.ID, dom.Espera{Version: recibida.Version})
	if err != nil {
		t.Fatal(err)
	}
	comprobarAviso := func(f d.FiltroMemoria, apartado d.Apartado) {
		t.Helper()
		avisos, err := svc.DiscrepanciasMemoria(ctx, a, f)
		if err != nil {
			t.Fatal(err)
		}
		for _, aviso := range avisos.Avisos {
			if aviso.ExpedienteID == recibida.ID && slices.Contains(aviso.Apartados, apartado) {
				return
			}
		}
		t.Fatalf("falta aviso de %s para equipo %s", apartado, f.EquipoID)
	}
	comprobarAviso(filtros[0], d.Pendientes)
	comprobarAviso(filtros[2], d.Pendientes)
	for _, f := range filtros {
		cerrar(f)
	}
	p.Version = espera.Version
	p.EquipoID = equipoDespues
	p.FechaActa = dom.FechaCivil(deps.Reloj.Ahora())
	_, err = app.Nuevo(deps).Aceptar(ctx, actor, recibida.ID, "resp1-avisos-aceptar-"+recibida.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	comprobarAviso(filtros[0], d.Pendientes)
	comprobarAviso(filtros[1], d.CasosAtendidos)
	comprobarAviso(filtros[2], d.Pendientes)
	t.Log("RESP1-AVISOS-OK: cierre→espera→aviso y aceptación en otro equipo avisa origen, destino y provincia")
}

func TestRESP1EsperaConsultaTecnicaSinDatosPersonales(t *testing.T) {
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" || os.Getenv("ETF_CALCULO_MEMORIA_CONTRASENA") == "" {
		t.Skip("ensayo técnico en migraciones/estadistica/ensayar.py")
	}
	_, ctx, deps, actor, _ := prepararDecision(t)
	recibida := subir(t, ctx, deps, actor)
	_, err := app.Nuevo(deps).DejarEnEspera(ctx, actor, recibida.ID, "resp1-consulta-espera-"+recibida.ID, dom.Espera{Version: recibida.Version})
	if err != nil {
		t.Fatal(err)
	}
	equipo := recibida.Datos.UbicacionConfirmada.EquipoID
	calc, err := postgres.NuevaCalculoMemoria(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer calc.Cerrar()
	for _, caso := range []struct {
		perfil, equipo string
		visible        bool
	}{{"consulta", "", true}, {"consulta", equipo, true}, {"consulta", "00000000-0000-4000-8000-000000000012", false}, {"administracion", "", false}, {"", "", false}} {
		identidad := plataforma.Identidad{}
		if caso.perfil != "" {
			identidad = plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: []string{caso.perfil}, EquipoID: caso.equipo}
		}
		err = calc.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			datos, err := (&exp.LecturaEstadistica{CalculoInterno: true}).ConsultarMemoria(ctx, caso.equipo, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), deps.Reloj.Ahora().Add(time.Second))
			if err != nil {
				return err
			}
			visible := false
			for _, x := range datos {
				if x.ID == recibida.ID {
					visible = true
				}
			}
			if visible != caso.visible {
				t.Fatalf("vista técnica %s/%s: %v", caso.perfil, caso.equipo, visible)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("RESP1-CONSULTA-OK: rol técnico, contexto y equipo; espera sin nombres ni contactos")
}

func TestRESP1SPAFMunicipioDeLaFamilia(t *testing.T) {
	b, ctx, deps, actor, p := prepararAceptacion(t)
	recibida := subir(t, ctx, deps, actor)
	p = conMenores(p, recibida)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT m.id::text,c.id::text,e.id::text FROM catalogo.municipio m,catalogo.centro_ss c,catalogo.equipo e WHERE m.ine='18102' AND c.codigo='PINOS_PUENTE_MOCLIN' AND e.codigo='06'`).Scan(&p.MunicipioID, &p.CentroID, &p.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	aceptada, err := app.Nuevo(deps).Aceptar(ctx, actor, recibida.ID, "resp1-municipio-"+recibida.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	fecha, err := time.Parse("2006-01-02", p.FechaActa)
	if err != nil {
		t.Fatal(err)
	}
	servicio := memoria.NuevoSPAF(memoria.DependenciasSPAF{Lectura: &exp.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: relojMemoria{}})
	tabla, err := servicio.ConsultarSPAF(ctx, d.ActorSPAF{UsuarioID: actor.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}, d.FiltroSPAF{Anio: fecha.Year(), Mes: int(fecha.Month())})
	if err != nil {
		t.Fatal(err)
	}
	for _, fila := range tabla.Filas {
		if fila.ExpedienteID == aceptada.ID {
			if !fila.Municipio.Conocido || fila.Municipio.Valor != "18102" {
				t.Fatal("SPAF debe usar municipio familiar, no centro ni equipo", fila.Municipio)
			}
			t.Log("RESP1-MUNICIPIO-OK: Íllora, centro Pinos Puente-Moclín y ETF 6; SPAF conserva sólo código municipal")
			return
		}
	}
	t.Fatal("expediente de Íllora omitido")
}
