package estadistica_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCifrasMemoriaPostgresFuenteRealPublicaContratoCompleto(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	fuente := app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: &adaptador.Repositorio{}}
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: app.FuenteConCierre{Fuente: fuente, Repositorio: &adaptador.Repositorio{}}, CifrasPublicadas: &adaptador.CifrasMemoria{}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
	for _, filtro := range []d.FiltroMemoria{{Anio: 2026}, {Anio: 2026, EquipoID: equipo}} {
		got, err := svc.ConsultarMemoria(ctx, actor, filtro)
		if err != nil || len(got.Apartados) != 33 {
			t.Fatal("contrato real rechazado", len(got.Apartados), err)
		}
		consulta := actor
		consulta.Perfiles = []string{"consulta"}
		foto, err := svc.ConsultarMemoria(ctx, consulta, filtro)
		if err != nil || len(foto.Apartados) != 33 || !foto.ConocidoEn.Equal(got.ConocidoEn) {
			t.Fatal(foto, err)
		}
	}
	t.Log("CIFRAS-FUENTE-OK 33 apartados reales provincial/equipo y lectura de Consulta")
}

type fuentePublicacionCifras struct{ familias int }

type fuenteDerivadosSSCC struct{ estado string }

func (s fuenteDerivadosSSCC) MemoriaEnTransaccion(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.InformeMemoria, error) {
	informe, err := (fuentePublicacionCifras{familias: 7}).MemoriaEnTransaccion(ctx, a, f, ahora)
	if err != nil {
		return d.InformeMemoria{}, err
	}
	n := 0
	if s.estado == "si" {
		n = 1
	}
	informe.Apartados = append(informe.Apartados, d.ApartadoMemoria{Codigo: d.DerivadosSSCC,
		Contrato: d.Contrato{Poblacion: "CI", Denominador: "cerrados_en_el_anio", Desconocidos: "sin_dato_separado", FechaDatos: "cierre_episodio", Corte: ahora, FechaEdad: ahora, VersionRegla: "2026.1:1", DivisionCero: "estadistica.divisor_cero"},
		Total:    d.Recuento{Familias: 1},
		Celdas:   []d.CeldaMemoria{{Grupo: d.Grupo{Subprograma: d.Riesgo, Estado: s.estado}, Unidad: d.Familias, Celda: d.Celda{Numerador: n, Poblacion: "CI", VersionRegla: "2026.1:1"}}}, Avisos: map[string]int{}})
	return informe, nil
}

func TestCifrasMemoriaPostgresPublicaDerivacionSSCCConocidaSiYNo(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	for _, estado := range []string{"si", "no"} {
		t.Run(estado, func(t *testing.T) {
			svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuenteDerivadosSSCC{estado: estado}, CifrasPublicadas: &adaptador.CifrasMemoria{}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
			f := d.FiltroMemoria{Anio: 2075, EquipoID: equipo}
			got, err := svc.ConsultarMemoria(ctx, actor, f)
			if err != nil || got.Apartados[1].Codigo != d.DerivadosSSCC || got.Apartados[1].Celdas[0].Grupo.Estado != estado {
				t.Fatal("estado conocido del apartado 10 rechazado", got, err)
			}
			consulta := actor
			consulta.Perfiles = []string{"consulta"}
			foto, err := svc.ConsultarMemoria(ctx, consulta, f)
			esperado := 0
			if estado == "si" {
				esperado = 1
			}
			if err != nil || foto.Apartados[1].Celdas[0].Numerador != esperado || foto.Apartados[1].Celdas[0].Grupo.Estado != estado {
				t.Fatal(foto, err)
			}
		})
	}
	t.Log("CIFRAS-SSCC-OK apartado 10 con sí/no conocido publicado y visible para Consulta")
}

type fuenteCifrasCierre struct {
	entrada.FuenteCierreMemoria
	actual *fuentePublicacionCifras
}

func (f fuenteCifrasCierre) MemoriaEnTransaccion(ctx context.Context, a d.ActorMemoria, filtro d.FiltroMemoria, t time.Time) (d.InformeMemoria, error) {
	return f.actual.MemoriaEnTransaccion(ctx, a, filtro, t)
}
func (f fuenteCifrasCierre) MemoriaFijadaEnTransaccion(ctx context.Context, a d.ActorMemoria, informe d.InformeMemoria, t time.Time) (d.InformeMemoria, error) {
	return f.actual.MemoriaEnTransaccion(ctx, a, informe.FiltroMemoria, t)
}

func TestCifrasMemoriaPostgresCierreCongelaLecturaYReaperturaPublicaActual(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	repo := &adaptador.Repositorio{}
	actual := &fuentePublicacionCifras{familias: 7}
	fuente := fuenteCifrasCierre{actual: actual}
	lectura := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: app.FuenteConCierre{Fuente: fuente, Repositorio: repo}, CifrasPublicadas: &adaptador.CifrasMemoria{}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
	cierre := app.NuevoCierre(app.DependenciasCierre{Fuente: fuente, Repositorio: repo, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
	f := d.FiltroMemoria{Anio: 2074, EquipoID: equipo}
	if _, err := lectura.ConsultarMemoria(ctx, actor, f); err != nil {
		t.Fatal(err)
	}
	estado, err := cierre.EstadoMemoria(ctx, actor, f)
	if err != nil {
		t.Fatal(err)
	}
	cerrada, err := cierre.CerrarMemoria(ctx, actor, f, "cerrar-cifras-"+actor.UsuarioID, d.ActoMemoria{Version: estado.Version})
	if err != nil {
		t.Fatal(err)
	}
	actual.familias = 9
	got, err := lectura.ConsultarMemoria(ctx, actor, f)
	if err != nil || got.Apartados[0].Total.Familias != 7 {
		t.Fatal("lectura cerrada cambia", got, err)
	}
	consulta := actor
	consulta.Perfiles = []string{"consulta"}
	foto, err := lectura.ConsultarMemoria(ctx, consulta, f)
	if err != nil || foto.Apartados[0].Total.Familias != 7 || !foto.ConocidoEn.Equal(got.ConocidoEn) {
		t.Fatal("Consulta no recibe foto cerrada", foto, err)
	}
	if _, err := cierre.ReabrirMemoria(ctx, actor, f, "reabrir-cifras-"+actor.UsuarioID, d.ActoMemoria{Version: cerrada.Version, Motivo: "Se revisaron los recuentos después de incorporar una corrección."}); err != nil {
		t.Fatal(err)
	}
	got, err = lectura.ConsultarMemoria(ctx, actor, f)
	if err != nil || got.Apartados[0].Total.Familias != 9 {
		t.Fatal("reabierta no cambia", got, err)
	}
	foto, err = lectura.ConsultarMemoria(ctx, consulta, f)
	if err != nil || foto.Apartados[0].Total.Familias != 9 {
		t.Fatal("Consulta no recibe foto nueva", foto, err)
	}
	t.Log("CIFRAS-CIERRE-OK lectura y Consulta conservan 7 tras corrección; reapertura publica 9")
}

func (s fuentePublicacionCifras) MemoriaEnTransaccion(ctx context.Context, _ d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.InformeMemoria, error) {
	if _, err := postgres.DesdeContexto(ctx); err != nil {
		return d.InformeMemoria{}, err
	}
	codigoEquipo := ""
	if f.EquipoID != "" {
		codigoEquipo = "08"
	}
	return d.InformeMemoria{FiltroMemoria: f, EquipoCodigo: codigoEquipo, Corte: ahora, ConocidoEn: ahora,
		Narrativas: d.Narrativas{ID: "identificador-restringido", D: "El equipo revisó los acuerdos de escolarización."},
		Apartados: []d.ApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.Recuento{Familias: s.familias},
			Celdas: []d.CeldaMemoria{{Grupo: d.Grupo{Subprograma: d.Riesgo}, Unidad: d.Familias, Celda: d.Celda{Numerador: s.familias, Porcentaje: &d.Porcentaje{Centesimas: 10000}}}}, Avisos: map[string]int{"estadistica.denominador_pendiente": 1}}}}, nil
}

func TestCifrasMemoriaPostgresPublicacionConsultaRLSYAuditoria(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	repo := &adaptador.CifrasMemoria{}
	dep := app.DependenciasLecturaMemoria{Fuente: fuentePublicacionCifras{familias: 7}, CifrasPublicadas: repo, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}
	svc := app.NuevoLecturaMemoria(dep)
	f := d.FiltroMemoria{Anio: 2071, EquipoID: equipo}
	publicada, err := svc.ConsultarMemoria(ctx, actor, f)
	if err != nil {
		t.Fatal(err)
	}
	provincial := d.FiltroMemoria{Anio: 2071}
	if _, err := svc.ConsultarMemoria(ctx, actor, provincial); err != nil {
		t.Fatal(err)
	}
	consulta := actor
	consulta.Perfiles = []string{"consulta"}
	got, err := svc.ConsultarMemoria(ctx, consulta, f)
	if err != nil || got.Apartados[0].Total.Familias != 7 || !got.ConocidoEn.Equal(publicada.ConocidoEn) {
		t.Fatal(got, err)
	}
	bJSON, err := json.Marshal(got)
	if err != nil || strings.Contains(string(bJSON), "escolarización") || strings.Contains(string(bJSON), "identificador-restringido") || strings.Contains(string(bJSON), "narrativas") {
		t.Fatal("contenido restringido en cifras", err)
	}
	_, err = svc.ConsultarMemoria(ctx, consulta, d.FiltroMemoria{Anio: 2072})
	if !errors.Is(err, d.ErrCifrasNoPublicadasMemoria) {
		t.Fatal("ausencia no distinguida", err)
	}

	for _, caso := range []struct {
		perfil, equipo string
		tabla, vista   int
	}{
		{"coordinacion", "", 2, 2}, {"consulta", "", 0, 2}, {"profesional_etf", equipo, 1, 1},
		{"profesional_etf", "00000000-0000-4000-8000-000000000012", 0, 0}, {"administracion", "", 0, 0}, {"", "", 0, 0},
	} {
		t.Run(caso.perfil+caso.equipo, func(t *testing.T) {
			perfiles := []string{}
			if caso.perfil != "" {
				perfiles = append(perfiles, caso.perfil)
			}
			err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: perfiles, EquipoID: caso.equipo}, func(ctx context.Context) error {
				tx, err := postgres.DesdeContexto(ctx)
				if err != nil {
					return err
				}
				var tabla, vista int
				if err = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.foto_totales WHERE anio=2071`).Scan(&tabla); err != nil {
					return err
				}
				if err = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.api_cifras_consulta WHERE anio=2071`).Scan(&vista); err != nil {
					return err
				}
				if tabla != caso.tabla || vista != caso.vista {
					t.Fatalf("RLS tabla=%d vista=%d", tabla, vista)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, caso := range []struct {
		perfil, equipo string
		cifras         d.CifrasMemoria
	}{
		{"consulta", "", publicada}, {"administracion", "", publicada},
		{"profesional_etf", "00000000-0000-4000-8000-000000000012", publicada},
		{"profesional_etf", equipo, d.CifrasMemoria{Anio: 2071, EquipoID: "", Corte: publicada.Corte, ConocidoEn: publicada.ConocidoEn, Apartados: publicada.Apartados}},
	} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: []string{caso.perfil}, EquipoID: caso.equipo}, func(ctx context.Context) error { return repo.Publicar(ctx, caso.cifras) })
		if err == nil {
			t.Fatal("publicación indebida permitida", caso.perfil, caso.equipo)
		}
	}
	profesional := actor
	profesional.Perfiles = []string{"profesional_etf"}
	profesional.EquipoID = equipo
	if _, err := svc.ConsultarMemoria(ctx, profesional, f); err != nil {
		t.Fatal("equipo no puede publicar", err)
	}
	dep.Fuente = fuentePublicacionCifras{familias: 9}
	dep.Auditoria = auditoriaFallida{}
	if got, err := app.NuevoLecturaMemoria(dep).ConsultarMemoria(ctx, actor, f); !errors.Is(err, d.ErrDisponibleMemoria) || got.Apartados != nil {
		t.Fatal("entrega sin auditoría", got, err)
	}
	got, err = svc.ConsultarMemoria(ctx, consulta, f)
	if err != nil || got.Apartados[0].Total.Familias != 7 {
		t.Fatal("foto no revertida", got, err)
	}
	t.Log("CIFRAS-OK publicación propia/provincial, Consulta sólo vista, ausencia explícita, rollback auditoría")
}

func TestCifrasMemoriaPostgresRechazaContenidoFueraDeContrato(t *testing.T) {
	b, ctx, actor, equipo := preparar(t)
	dep := app.DependenciasLecturaMemoria{Fuente: fuentePublicacionCifras{familias: 3}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}
	cifras, err := app.NuevoLecturaMemoria(dep).ConsultarMemoria(ctx, actor, d.FiltroMemoria{Anio: 2073, EquipoID: equipo})
	if err != nil {
		t.Fatal(err)
	}
	bJSON, err := json.Marshal(cifras)
	if err != nil {
		t.Fatal(err)
	}
	var base map[string]any
	if err = json.Unmarshal(bJSON, &base); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []string{"narrativas", "composicion", "grupo texto libre", "grupo identificador", "grupo número expediente", "grupo código ajeno", "equipo nombre", "equipo código ajeno", "aviso nombre", "contrato nombre", "regla nombre", "aviso identificador", "numerador negativo", "año ajeno", "fecha ajena"} {
		t.Run(caso, func(t *testing.T) {
			var entrada map[string]any
			if err := json.Unmarshal(bJSON, &entrada); err != nil {
				t.Fatal(err)
			}
			apartado := entrada["apartados"].([]any)[0].(map[string]any)
			celda := apartado["celdas"].([]any)[0].(map[string]any)
			switch caso {
			case "narrativas":
				entrada["narrativas"] = map[string]any{"d": "Texto que no debe publicarse."}
			case "composicion":
				apartado["composicion"] = []string{"00000000-0000-4000-8000-000000000012"}
			case "grupo texto libre":
				celda["grupo"].(map[string]any)["causa"] = "No hubo consentimiento de la familia."
			case "grupo identificador":
				celda["grupo"].(map[string]any)["causa"] = "00000000-0000-4000-8000-000000000012"
			case "grupo número expediente":
				celda["grupo"].(map[string]any)["causa"] = "08-01-26"
			case "grupo código ajeno":
				celda["grupo"].(map[string]any)["estado"] = "Ana.Garcia"
			case "equipo nombre":
				entrada["equipo_codigo"] = "Ana.Garcia"
			case "equipo código ajeno":
				entrada["equipo_codigo"] = "07"
			case "aviso nombre":
				apartado["avisos"] = map[string]any{"estadistica.ana_garcia": 1}
			case "contrato nombre":
				apartado["contrato"].(map[string]any)["denominador"] = "Ana.Garcia"
			case "regla nombre":
				apartado["contrato"].(map[string]any)["version_regla"] = "Ana.Garcia"
			case "aviso identificador":
				apartado["avisos"] = map[string]any{"00000000-0000-4000-8000-000000000012": 1}
			case "numerador negativo":
				celda["numerador"] = -1
			case "año ajeno":
				entrada["anio"] = 2074
			case "fecha ajena":
				entrada["conocido_en"] = "2026-01-01T00:00:00Z"
			}
			datos, err := json.Marshal(entrada)
			if err != nil {
				t.Fatal(err)
			}
			err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
				tx, err := postgres.DesdeContexto(ctx)
				if err != nil {
					return err
				}
				_, err = tx.Exec(ctx, `INSERT INTO estadistica.foto_totales(anio,equipo_id,fecha,cifras,publicado_por) VALUES($1,$2,$3,$4,$5)`, 2073, equipo, cifras.ConocidoEn, datos, actor.UsuarioID)
				return err
			})
			var rechazo *pgconn.PgError
			if !errors.As(err, &rechazo) || rechazo.Code != "23514" {
				t.Fatalf("se esperaba rechazo del contrato, no otro fallo: %v", err)
			}
		})
	}
	t.Log("CIFRAS-FORMA-OK narrativas, composición, textos, UUID, negativos y ámbito rechazados")
}
