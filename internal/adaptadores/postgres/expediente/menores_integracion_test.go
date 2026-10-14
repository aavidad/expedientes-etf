package expediente_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documentos "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"sync"
	"testing"
)

func prepararMenores(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, string) {
	b, ctx, d, a, id := prepararFamilia(t)
	d.Menores = &adaptador.Repositorio{}
	d.CatalogosMenores = nombres.CatalogosMenores{Ruta: "../../../../idiomas/es/catalogos/menores.json"}
	return b, ctx, d, a, id
}
func datosMenores() []dominio.Menor {
	return []dominio.Menor{{Orden: 1, Nombre: "Lucía", Apellidos: "Moreno Castillo", Sexo: "M", Nacimiento: texto("2016-05-14"), Escolarizacion: texto("PRIMARIA"), CentroProteccionTexto: "Centro de acogida de la Vega", AtencionTemprana: nil}, {Orden: 2, Nombre: "Pablo", Apellidos: "Moreno Castillo", Sexo: "H", Subprograma: "REUNIFICACION"}}
}
func TestMenoresVersionConcurrenteIdempotenciaYAuditoria(t *testing.T) {
	b, ctx, d, a, id := prepararMenores(t)
	s := aplicacion.Nuevo(d)
	ms := datosMenores()
	p := dominio.CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &ms}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-inicio
			_, errs[i] = s.EditarMenores(ctx, a, id, fmt.Sprintf("menores-%d", i), p)
		}(i)
	}
	close(inicio)
	wg.Wait()
	ganadora := -1
	for i, err := range errs {
		if err == nil {
			ganadora = i
		} else {
			var c dominio.Conflicto
			if !errors.As(err, &c) || c.VersionActual != 3 {
				t.Fatal(err)
			}
		}
	}
	if ganadora < 0 || errs[1-ganadora] == nil {
		t.Fatal(errs)
	}
	r, err := s.EditarMenores(ctx, a, id, fmt.Sprintf("menores-%d", ganadora), p)
	if err != nil || r.Version != 3 {
		t.Fatal(r, err)
	}
	ms[0].Nombre = "Elena"
	if _, err = s.EditarMenores(ctx, a, id, fmt.Sprintf("menores-%d", ganadora), p); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	f, err := s.ConsultarMenores(ctx, a, id)
	if err != nil || len(f.Menores) != 2 || f.Menores[0].Subprograma != "RIESGO_SOCIAL" || f.Menores[1].Subprograma != "REUNIFICACION" {
		t.Fatal(f, err)
	}
	d.RegistroTransaccional = registroFallido{}
	p.Version = 3
	if _, err = aplicacion.Nuevo(d).EditarMenores(ctx, a, id, "menores-auditoria", p); err == nil {
		t.Fatal("Auditoría fallida permitió guardar")
	}
	f, err = s.ConsultarMenores(ctx, a, id)
	if err != nil || f.Version != 3 || f.Menores[0].Nombre != "Lucía" {
		t.Fatal(f, err)
	}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		consulta, err := (adaptador.ConsultaMenores{}).MenoresVigentes(ctx, id, "2026-02-03")
		if err != nil {
			return err
		}
		if len(consulta) != 2 || consulta[0].Vigencia.Revision != 3 || consulta[0].EquipoID != a.EquipoID {
			t.Fatal(consulta)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("PostgreSQL: un éxito y un 409; recibo idéntico; reutilización 422; auditoría fallida revierte; subprograma por defecto y consulta F2.05/F4.05")
}
func TestMenoresVigenciasRetiradaYReincorporacion(t *testing.T) {
	_, ctx, d, a, id := prepararMenores(t)
	s := aplicacion.Nuevo(d)
	ms := datosMenores()
	guardar := func(version int, desde, clave string, ms []dominio.Menor) {
		t.Helper()
		if _, err := s.EditarMenores(ctx, a, id, clave, dominio.CambioMenores{Version: version, Desde: desde, Menores: &ms}); err != nil {
			t.Fatal(err)
		}
	}
	guardar(2, "2026-02-03", "menores-febrero", ms)
	f, err := s.ConsultarMenores(ctx, a, id)
	if err != nil {
		t.Fatal(err)
	}
	ms = f.Menores
	uno := ms[:1]
	guardar(3, "2026-04-01", "menores-abril", uno)
	ms[0].Sexo = "H"
	guardar(4, "2026-03-01", "menores-marzo", ms)
	for _, c := range []struct {
		corte    string
		total    int
		sexo     string
		revision int
	}{{"2026-02-28", 2, "M", 3}, {"2026-03-01", 2, "H", 5}, {"2026-04-01", 1, "M", 4}} {
		h, err := s.ConsultarHistoriaMenores(ctx, a, id, c.corte)
		if err != nil || len(h.Menores) != c.total || h.Menores[0].Sexo != c.sexo || h.Vigencia.Revision != c.revision {
			t.Fatal(h, err)
		}
		b, _ := json.Marshal(h)
		if strings.Contains(string(b), "Lucía") || strings.Contains(string(b), "Vega") {
			t.Fatal("Historia con nombres o textos libres")
		}
	}
	f, err = s.ConsultarMenores(ctx, a, id)
	if err != nil || len(f.Menores) != 1 || f.Menores[0].Sexo != "M" {
		t.Fatal(f, err)
	}
	vacia := []dominio.Menor{}
	guardar(5, "2026-05-01", "menores-vacia", vacia)
	h, err := s.ConsultarHistoriaMenores(ctx, a, id, "2026-05-01")
	if err != nil || len(h.Menores) != 0 || h.Vigencia == nil {
		t.Fatal(h, err)
	}
	guardar(6, "2026-06-01", "menores-reincorporar", ms)
	f, err = s.ConsultarMenores(ctx, a, id)
	if err != nil || len(f.Menores) != 2 || f.Menores[1].ID != ms[1].ID {
		t.Fatal(f, err)
	}
	// Una corrección de la misma fecha añade revisión y conserva la sustituida.
	ms[0].Sexo = "M"
	guardar(7, "2026-06-01", "menores-corregir", ms)
	h, err = s.ConsultarHistoriaMenores(ctx, a, id, "2026-06-01")
	if err != nil || h.Vigencia.Revision != 8 || h.Menores[0].Sexo != "M" {
		t.Fatal(h, err)
	}
	t.Log("Vigencias: aceptación inicial; intervalos sin solape; retroactividad conserva abril; colección vacía; identidad estable al reincorporar; corrección por adición")
}

func TestMenoresRLSEquiposYVinculosCompuestos(t *testing.T) {
	b, ctx, d, a, id := prepararMenores(t)
	s := aplicacion.Nuevo(d)
	ficha, err := s.ConsultarMenores(ctx, a, id)
	if err != nil || len(ficha.Menores) == 0 {
		t.Fatal(ficha, err)
	}
	_, _, d2, a2, id2 := prepararMenores(t)
	ajenos, err := aplicacion.Nuevo(d2).ConsultarMenores(ctx, a2, id2)
	if err != nil {
		t.Fatal(err)
	}
	ms := []dominio.Menor{ajenos.Menores[0]}
	if _, err = s.EditarMenores(ctx, a, id, "menor-otro-expediente", dominio.CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &ms}); !errors.Is(err, dominio.ErrMenores) {
		t.Fatal(err)
	}
	propio := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	var marca string
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		return e.QueryRow(ctx, `INSERT INTO expediente.marca(expediente_id,equipo_id,lista,codigo) VALUES($1,$2,'factor_menor','ABSENTISMO') RETURNING id::text`, id, a.EquipoID).Scan(&marca)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ exp, menor string }{{id, ajenos.Menores[0].ID}, {id2, ajenos.Menores[0].ID}} {
		err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			_, err := e.Exec(ctx, `INSERT INTO expediente.marca_menor_excluido(marca_id,menor_id,expediente_id,equipo_id) VALUES($1,$2,$3,$4)`, marca, c.menor, c.exp, a.EquipoID)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "23503") {
			t.Fatal("Vínculo cruzado no rechazado por FK", err)
		}
	}
	otro := a
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		return e.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE id<>$1 LIMIT 1`, a.EquipoID).Scan(&otro.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConsultarMenores(ctx, otro, id); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal(err)
	}
	for _, actor := range []dominio.Actor{otro, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, {}} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"menor", "menores_revision", "menor_revision", "marca", "marca_menor_excluido", "consulta_menor", "consulta_menores_vigencia"} {
				var n int
				if err := e.QueryRow(ctx, `SELECT count(*) FROM expediente.`+tabla+` WHERE expediente_id=$1`, id).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					return fmt.Errorf("RLS dejó ver %s", tabla)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	d.RegistroTransaccional = registroFallido{}
	if lectura, err := aplicacion.Nuevo(d).ConsultarMenores(ctx, a, id); err == nil || lectura.ID != "" {
		t.Fatal("Lectura entregada sin auditoría", lectura, err)
	}
	// Ni UPDATE ni DELETE permiten reescribir las revisiones estadísticas.
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `DELETE FROM expediente.menores_revision WHERE expediente_id=$1`, id)
		return err
	})
	if err == nil {
		t.Fatal("Historia borrable")
	}
	// Acredita que la consulta histórica se puede planificar con los índices.
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		rows, err := e.Query(ctx, `EXPLAIN SELECT id FROM expediente.consulta_menor WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta)`, id, "2026-02-03")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var plan string
			if err = rows.Scan(&plan); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("RLS: otro equipo, Consulta, Administración y contexto ausente sin filas; FK rechaza menor o marca de otro expediente; historia inmutable; EXPLAIN válido; lectura sin auditoría bloqueada")
}

func TestMenoresCierreBloqueaFichaYReaperturaPermiteEditar(t *testing.T) {
	b, ctx, deps, a, id := prepararMenores(t)
	r := &adaptador.Repositorio{}
	deps.Cierres = r
	deps.Propuestas = r
	deps.MotivosCierre = &catalogo.Repositorio{}
	deps.SalidasSPM = r
	deps.EmisionesFinalizacion = documentos.Repositorio{}
	deps.ConsultaCierreSPM = aplicacion.ConsultaCierreSPM{Repositorio: r, Salidas: r}
	s := aplicacion.Nuevo(deps)
	_, _, _, coord := preparar(t)
	f, err := s.ConsultarMenores(ctx, a, id)
	if err != nil {
		t.Fatal(err)
	}
	// Coordinación lee la ficha, pero no obtiene UPDATE al ampliar los menores.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		tag, err := e.Exec(ctx, `UPDATE expediente.menor SET datos='{"nombre":"Ana"}'::jsonb WHERE id=$1`, f.Menores[0].ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			return fmt.Errorf("Coordinación pudo editar ficha")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	final, err := s.GuardarFinalizacion(ctx, a, id, "menores-final", dominio.Finalizacion{Version: 2, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true})
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := s.ProponerCierre(ctx, a, id, "menores-proponer", dominio.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	cerrado, err := s.Cerrar(ctx, coord, id, "menores-cerrar", dominio.PeticionCierre{Version: propuesta.Version, Fecha: "2026-03-01"})
	if err != nil {
		t.Fatal(err)
	}
	ms := datosMenores()
	if _, err = s.EditarMenores(ctx, a, id, "menores-cerrados", dominio.CambioMenores{Version: cerrado.Version, Desde: "2026-03-01", Menores: &ms}); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	// La base impide añadir historia al expediente cerrado aun con Coordinación.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en) VALUES($1,$2,100,'2026-03-01',$3,now())`, id, a.EquipoID, coord.UsuarioID)
		return err
	})
	if err == nil {
		t.Fatal("Base permitió historia de cerrado")
	}
	nuevo, err := s.Reabrir(ctx, coord, id, "menores-reabrir", dominio.PeticionReapertura{Version: cerrado.Version, Fecha: "2026-03-02", Motivo: "Se retoma el acompañamiento familiar."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.EditarMenores(ctx, a, id, "menores-reabiertos", dominio.CambioMenores{Version: nuevo.Version, Desde: "2026-03-02", Menores: &ms}); err != nil {
		t.Fatal(err)
	}
	t.Log("Coordinación no edita ficha; cerrado bloquea aplicación y SQL; reapertura permite guardar menores")
}
