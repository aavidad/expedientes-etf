package expediente_test

import (
	"context"
	"errors"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	familia "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dom "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type relojMemoria struct{}

func (relojMemoria) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type auditMemoriaFallida struct{}

func (auditMemoriaFallida) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.fallo")
}
func (auditMemoriaFallida) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.fallo")
}
func TestMemoriaCierreFotoCorreccionProvincialYReapertura(t *testing.T) {
	b, ctx, deps, actor, p := prepararAceptacion(t)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo<>'08' ORDER BY codigo LIMIT 1`).Scan(&p.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	recibida := subir(t, ctx, deps, actor)
	id := recibida.ID
	if _, err := familia.Nuevo(deps).Aceptar(ctx, actor, id, "aceptar-"+id, conMenores(p, recibida)); err != nil {
		t.Fatal(err)
	}
	actor.Perfiles = []string{"profesional_etf"}
	actor.EquipoID = p.EquipoID
	deps.Familias = &exp.Repositorio{}
	deps.SMI = &catalogo.Repositorio{}
	deps.CatalogosFamilia = nombres.CatalogosFamilia{Listas: "../../../../idiomas/es/catalogos/listas.json", Familia: "../../../../idiomas/es/catalogos/familia.json", Parametros: "../../../../datos/catalogos/familia.json"}
	repositorio := &repo.Repositorio{}
	fuente := app.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: repositorio}
	registro := postgres.NuevoRegistroAuditoria(b)
	cierreDeps := app.DependenciasCierre{Repositorio: repositorio, Fuente: fuente, Transacciones: b, Auditoria: registro, Reloj: relojMemoria{}}
	svc := app.NuevoCierre(cierreDeps)
	a := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}
	coord := a
	coord.Perfiles = []string{"coordinacion"}
	coord.EquipoID = ""
	f := d.FiltroMemoria{Anio: 2026, EquipoID: a.EquipoID}
	estado, err := svc.EstadoMemoria(ctx, a, f)
	if err != nil {
		t.Fatal(err)
	}
	if estado.Version != 0 || estado.ID != "" {
		t.Fatal("la lectura creó una fila", estado)
	}
	n, err := app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: repositorio, Transacciones: b, Auditoria: registro, Reloj: relojMemoria{}}).GuardarNarrativas(ctx, a, d.Narrativas{FiltroMemoria: f, D: "Las familias acordaron rutinas semanales de cuidado."})
	if err != nil {
		t.Fatal(err)
	}
	// Dos cierres con la misma clave devuelven una sola revisión.
	var wg sync.WaitGroup
	rs := make([]d.ResultadoActoMemoria, 2)
	errs := make([]error, 2)
	for i := range rs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rs[i], errs[i] = svc.CerrarMemoria(ctx, a, f, "cerrar-"+id, d.ActoMemoria{Version: n.Version})
		}(i)
	}
	wg.Wait()
	for i := range rs {
		if errs[i] != nil || rs[i].RevisionID != rs[0].RevisionID || rs[i].Huella != rs[0].Huella {
			t.Fatal("cierre concurrente", i, errs[i], rs[i])
		}
	}
	foto, err := svc.InstantaneaMemoria(ctx, a, f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if foto.Informe.Narrativas.D != n.D || foto.Informe.VersionReglas == "" || len(foto.Huella) != 64 || foto.Motor != d.VersionMotorMemoria {
		t.Fatal("foto incompleta", foto)
	}
	h, err := d.HuellaMemoria(foto.Informe)
	if err != nil || h != foto.Huella {
		t.Fatal("huella", err)
	}
	_, err = svc.CerrarMemoria(ctx, a, f, "cerrar-"+id, d.ActoMemoria{Version: rs[0].Version})
	if !errors.Is(err, d.ErrIdempotenciaMemoria) {
		t.Fatal("clave distinta entrada", err)
	}
	_, err = svc.CerrarMemoria(ctx, a, f, "conflicto-"+id, d.ActoMemoria{Version: n.Version})
	var conflicto *d.ConflictoMemoria
	if !errors.As(err, &conflicto) || conflicto.Actual != rs[0].Version {
		t.Fatal("versión 409", err)
	}
	_, err = app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: repositorio, Transacciones: b, Auditoria: registro, Reloj: relojMemoria{}}).GuardarNarrativas(ctx, a, d.Narrativas{FiltroMemoria: f, Version: rs[0].Version, D: "Se revisaron los horarios."})
	if !errors.Is(err, d.ErrCerradaMemoria) {
		t.Fatal("editó cerrada", err)
	}
	provincial := d.FiltroMemoria{Anio: 2026}

	_, err = svc.CerrarMemoria(ctx, coord, provincial, "provincial-"+id, d.ActoMemoria{})
	if err != nil {
		t.Fatal(err)
	}
	fotoProv, err := svc.InstantaneaMemoria(ctx, coord, provincial, 0)
	if err != nil {
		t.Fatal(err)
	}
	deps.RegistroTransaccional = app.RegistroConAvisos{Registro: registro, Repositorio: repositorio, Fuente: fuente, Equipos: &exp.LecturaEstadistica{}}
	// Corregir una dimensión histórica afecta a 17 y deja ambas fotos intactas.
	fam := datosFamilia()
	fam.Composicion = texto("MONOPARENTAL_MADRE")
	cambio, err := familia.Nuevo(deps).EditarFamilia(ctx, actor, id, "corregir-"+id, dom.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &fam})
	if err != nil {
		t.Fatal(err)
	}
	avisos, err := svc.DiscrepanciasMemoria(ctx, coord, provincial)
	if err != nil {
		t.Fatal(err)
	}
	if len(avisos.Avisos) != 2 || len(avisos.Diferencias) != 2 {
		t.Fatalf("avisos %d diferencias %d", len(avisos.Avisos), len(avisos.Diferencias))
	}
	for _, aviso := range avisos.Avisos {
		if !slices.Contains(aviso.Apartados, d.ComposicionFamiliar) || aviso.ExpedienteID != id || aviso.EventoID == 0 {
			t.Fatal("apartados aviso", aviso)
		}
	}
	fotoDespues, err := svc.InstantaneaMemoria(ctx, a, f, 0)
	if err != nil || !reflect.DeepEqual(foto, fotoDespues) {
		t.Fatal("se alteró instantánea", err)
	}
	cerrada := app.FuenteConCierre{Fuente: fuente, Repositorio: repositorio}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		informe, err := cerrada.MemoriaEnTransaccion(ctx, coord, provincial, time.Now().UTC())
		if !reflect.DeepEqual(informe, fotoProv.Informe) {
			t.Fatal("Word no usa foto")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// La misma corrección estadística no se repite al editar sólo un contacto.
	fam.Telefono = "958456789"
	contacto, err := familia.Nuevo(deps).EditarFamilia(ctx, actor, id, "contacto-"+id, dom.CambioFamilia{Version: cambio.Version, Desde: "2026-02-03", Familia: &fam})
	if err != nil {
		t.Fatal(err)
	}
	sinImpacto, err := svc.DiscrepanciasMemoria(ctx, coord, provincial)
	if err != nil || len(sinImpacto.Avisos) != len(avisos.Avisos) {
		t.Fatal("aviso sin impacto", err, sinImpacto)
	}
	// Si no puede guardarse el aviso, se revierten ficha y auditoría del cambio.
	deps.RegistroTransaccional = app.RegistroConAvisos{Registro: registro, Repositorio: avisosFallidos{repositorio}, Fuente: fuente, Equipos: &exp.LecturaEstadistica{}}
	fam.Composicion = texto("BIPARENTAL")
	_, err = familia.Nuevo(deps).EditarFamilia(ctx, actor, id, "aviso-fallido-"+id, dom.CambioFamilia{Version: contacto.Version, Desde: "2026-02-03", Familia: &fam})
	if !errors.Is(err, dom.ErrNoDisponible) {
		t.Fatal("fallo del aviso", err)
	}
	ficha, err := familia.Nuevo(deps).ConsultarFamilia(ctx, actor, id)
	if err != nil || ficha.Version != contacto.Version || *ficha.Familia.Composicion != "MONOPARENTAL_MADRE" {
		t.Fatal("corrección parcial", err, ficha.Version)
	}
	// Una edición de contacto sin cambio estadístico no genera otro aviso
	// cuando se evalúa sobre una foto nueva (tras reabrir y volver a cerrar).
	_, err = svc.ReabrirMemoria(ctx, a, f, "abrir-sin-permiso", d.ActoMemoria{Version: rs[0].Version, Motivo: "Revisar la composición."})
	if !errors.Is(err, d.ErrPermisoMemoria) {
		t.Fatal("equipo reabre", err)
	}
	_, err = svc.ReabrirMemoria(ctx, coord, f, "abrir-sin-motivo", d.ActoMemoria{Version: rs[0].Version})
	if !errors.Is(err, d.ErrActoMemoria) {
		t.Fatal("motivo vacío", err)
	}
	abierto, err := svc.ReabrirMemoria(ctx, coord, f, "abrir-"+id, d.ActoMemoria{Version: rs[0].Version, Motivo: "Corregir la composición familiar del apartado 17."})
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := svc.CerrarMemoria(ctx, a, f, "recerrar-"+id, d.ActoMemoria{Version: abierto.Version})
	if err != nil || nueva.RevisionActual != 2 || nueva.Huella == foto.Huella {
		t.Fatal("segunda revisión", err, nueva)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		historica, err := cerrada.MemoriaEnTransaccion(ctx, a, f, foto.CerradaEn)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(historica, foto.Informe) {
			t.Fatal("una emisión anterior cambió al reabrir o recerrar")
		}
		previo, err := cerrada.MemoriaEnTransaccion(ctx, a, f, foto.CerradaEn.Add(-time.Microsecond))
		if err != nil {
			return err
		}
		if !previo.ConocidoEn.Equal(foto.CerradaEn.Add(-time.Microsecond)) {
			t.Fatal("se usó una foto futura")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	antigua, err := svc.InstantaneaMemoria(ctx, a, f, 1)
	if err != nil || !reflect.DeepEqual(antigua, foto) {
		t.Fatal("perdió revisión anterior", err)
	}
	// Fallar auditoría revierte el cambio de estado, el motivo y el recibo.
	cierreDeps.Auditoria = auditMemoriaFallida{}
	_, err = app.NuevoCierre(cierreDeps).ReabrirMemoria(ctx, coord, f, "fallida-"+id, d.ActoMemoria{Version: nueva.Version, Motivo: "Revisar cifras de cierre."})
	if !errors.Is(err, d.ErrDisponibleMemoria) {
		t.Fatal("fallo auditoría", err)
	}
	actual, err := svc.EstadoMemoria(ctx, a, f)
	if err != nil || actual.Estado != "cerrada" || actual.Version != nueva.Version {
		t.Fatal("estado parcial", err, actual)
	}
	// La historia de actos conserva el motivo ligado al actor y a la versión.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var motivo string
		err = tx.QueryRow(ctx, `SELECT motivo FROM estadistica.memoria_acto WHERE memoria_id=$1 AND version=$2 AND usuario_id=$3`, foto.MemoriaID, abierto.Version, coord.UsuarioID).Scan(&motivo)
		if motivo != "Corregir la composición familiar del apartado 17." {
			t.Fatal("motivo no auditado")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// RLS protege fotos y fuentes incluso mediante SQL directo.
	for _, caso := range []struct {
		perfiles []string
		equipo   string
	}{{[]string{"consulta"}, ""}, {[]string{"administracion"}, ""}, {[]string{"profesional_etf"}, "00000000-0000-4000-8000-000000000012"}, {nil, ""}} {
		identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: caso.perfiles, EquipoID: caso.equipo}
		if caso.perfiles == nil {
			identidad = plataforma.Identidad{}
		}
		err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			for _, tabla := range []string{"memoria_revision", "memoria_base_equipo", "memoria_acto", "memoria_discrepancia"} {
				var n int
				err = tx.QueryRow(ctx, "SELECT count(*) FROM estadistica."+tabla+" WHERE memoria_id=$1", foto.MemoriaID).Scan(&n)
				if err != nil {
					return err
				}
				if n != 0 {
					t.Fatal("RLS", tabla, caso.perfiles, n)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	propia, err := svc.InstantaneaMemoria(ctx, a, provincial, 1)
	if err != nil || propia.Informe.EquipoID != a.EquipoID {
		t.Fatal("omitir equipo no se limitó al propio", err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var n int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.memoria_revision WHERE memoria_id=$1`, fotoProv.MemoriaID).Scan(&n)
		if n != 0 {
			t.Fatal("RLS provincial")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`UPDATE estadistica.memoria_revision SET huella=repeat('0',64) WHERE memoria_id=$1`,
		`DELETE FROM estadistica.memoria_revision WHERE memoria_id=$1`,
		`UPDATE estadistica.memoria SET textos='{}',version=version+1 WHERE id=$1`,
		`UPDATE estadistica.memoria SET estado='borrador',version=version+1 WHERE id=$1`,
		`INSERT INTO estadistica.memoria_acto(memoria_id,version,accion,motivo,usuario_id,ocurrido_en) VALUES($1,100,'reabrir',NULL,$2,now())`,
	} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			args := []any{foto.MemoriaID}
			if strings.Contains(sql, "$2") {
				args = append(args, coord.UsuarioID)
			}
			_, err = tx.Exec(ctx, sql, args...)
			return err
		})
		if err == nil {
			t.Fatal("SQL permitió violar inmutabilidad o motivo", sql)
		}
	}
	// La función de bases sólo publica cifras y huellas del equipo propio.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		bases, err := repositorio.BasesEquipo(ctx, a.EquipoID)
		if err != nil {
			return err
		}
		for _, base := range bases {
			if base.Informe.EquipoID != a.EquipoID || len(base.Informe.Fuentes) > 0 || len(base.Informe.Composicion) > 0 || base.Informe.Narrativas.D != "" {
				t.Fatal("proyección de bases expone contenido")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("CIERRE-OK: foto y reglas inmutables, idempotencia concurrente, aviso equipo/provincia, dos revisiones, motivo y rollback")
}

type avisosFallidos struct{ salida.Cierres }

func (avisosFallidos) AvisarDiscrepancia(context.Context, d.DiscrepanciaMemoria) error {
	return errors.New("estadistica.aviso_fallido")
}

func TestMemoriaCierreDerivacionSinEquipo(t *testing.T) {
	b, ctx, deps, a := preparar(t)
	recibida := subir(t, ctx, deps, a)
	registro := app.RegistroConAvisos{Registro: postgres.NuevoRegistroAuditoria(b), Repositorio: &repo.Repositorio{}, Fuente: app.FuenteMemoria{}, Equipos: &exp.LecturaEstadistica{}}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		_, err := registro.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: time.Now().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "expediente.confirmar_derivacion", Entidad: "expediente", EntidadID: recibida.ID, ExpedienteID: recibida.ID, Resultado: "ok"})
		return err
	})
	if err != nil {
		t.Fatal("el observador bloqueó una derivación todavía sin equipo", err)
	}
}
