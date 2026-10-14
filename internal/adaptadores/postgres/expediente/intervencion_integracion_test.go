package expediente_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func prepararIntervencion(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, string) {
	b, ctx, d, a, id := prepararMenores(t)
	d.Intervencion = &adaptador.Repositorio{}
	d.CatalogosIntervencion = &catalogo.Repositorio{}
	return b, ctx, d, a, id
}
func datosIntervencion() dominio.Intervencion {
	factores := []dominio.MarcaIntervencion{{Codigo: "ABSENTISMO"}}
	recursos := []dominio.MarcaIntervencion{{Codigo: "OTROS", TextoOtros: "Servicio de apoyo de la Vega"}}
	vacia := []dominio.MarcaIntervencion{}
	return dominio.Intervencion{Fase: texto("EVALUACION"), Listas: map[string]*[]dominio.MarcaIntervencion{"coordinacion": &recursos, "factor_familia": nil, "factor_menor": &factores, "tipologia": &vacia}}
}
func TestIntervencionVersionConcurrenteIdempotenciaYAuditoria(t *testing.T) {
	_, ctx, d, a, id := prepararIntervencion(t)
	s := aplicacion.Nuevo(d)
	i := datosIntervencion()
	p := dominio.CambioIntervencion{Version: 2, Desde: "2026-02-03", Intervencion: &i}
	errores := make([]error, 2)
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for n := range errores {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-inicio
			_, errores[n] = s.EditarIntervencion(ctx, a, id, fmt.Sprintf("intervencion-%d", n), p)
		}(n)
	}
	close(inicio)
	wg.Wait()
	ganadora := -1
	for n, err := range errores {
		if err == nil {
			ganadora = n
		} else {
			var c dominio.Conflicto
			if !errors.As(err, &c) || c.VersionActual != 3 {
				t.Fatal(err)
			}
		}
	}
	if ganadora < 0 || errores[1-ganadora] == nil {
		t.Fatal(errores)
	}
	r, err := s.EditarIntervencion(ctx, a, id, fmt.Sprintf("intervencion-%d", ganadora), p)
	if err != nil || r.Version != 3 {
		t.Fatal(r, err)
	}
	i.Fase = texto("INTERVENCION")
	if _, err = s.EditarIntervencion(ctx, a, id, fmt.Sprintf("intervencion-%d", ganadora), p); !errors.Is(err, dominio.ErrIdempotencia) {
		t.Fatal(err)
	}
	d.RegistroTransaccional = registroFallido{}
	p.Version = 3
	if _, err = aplicacion.Nuevo(d).EditarIntervencion(ctx, a, id, "auditoria-fallida", p); err == nil {
		t.Fatal("Escritura sin auditoría")
	}
	f, err := s.ConsultarIntervencion(ctx, a, id)
	if err != nil || f.Version != 3 || *f.Intervencion.Fase != "EVALUACION" {
		t.Fatal(f, err)
	}
	if f, err := aplicacion.Nuevo(d).ConsultarIntervencion(ctx, a, id); err == nil || f.ID != "" {
		t.Fatal("Lectura sin auditoría", f, err)
	}
	t.Log("PostgreSQL: un éxito y un 409; recibo idéntico; clave reutilizada rechazada; auditoría fallida revierte escritura y no entrega lectura")
}
func TestIntervencionHistoriaExclusionesYConsulta(t *testing.T) {
	b, ctx, d, a, id := prepararIntervencion(t)
	s := aplicacion.Nuevo(d)
	menores, err := s.ConsultarMenores(ctx, a, id)
	if err != nil || len(menores.Menores) < 1 {
		t.Fatal(menores, err)
	}
	i := datosIntervencion()
	(*i.Listas["factor_menor"])[0].MenoresExcluidos = []string{menores.Menores[0].ID}
	guardar := func(v int, desde string, i dominio.Intervencion) {
		t.Helper()
		if _, err := s.EditarIntervencion(ctx, a, id, fmt.Sprintf("revision-%d", v), dominio.CambioIntervencion{Version: v, Desde: desde, Intervencion: &i}); err != nil {
			t.Fatal(err)
		}
	}
	guardar(2, "2026-02-03", i)
	i.Fase = texto("INTERVENCION")
	guardar(3, "2026-04-01", i)
	i.Fase = texto("RECEPCION")
	(*i.Listas["coordinacion"])[0].TextoOtros = "Recurso municipal de apoyo"
	guardar(4, "2026-03-01", i)
	for _, c := range []struct {
		corte, fase string
		revision    int
	}{{"2026-02-28", "EVALUACION", 3}, {"2026-03-01", "RECEPCION", 5}, {"2026-04-01", "INTERVENCION", 4}} {
		h, err := s.ConsultarHistoriaIntervencion(ctx, a, id, c.corte)
		if err != nil || h.Vigencia == nil || h.Vigencia.Revision != c.revision || *h.Intervencion.Fase != c.fase || len(h.Menores) < 1 || len(*h.Menores[0].Listas["factor_menor"]) != 0 {
			t.Fatal(h, err)
		}
		if h.Intervencion.Listas["factor_familia"] != nil || h.Intervencion.Listas["tipologia"] == nil {
			t.Fatal(h)
		}
		datos, _ := json.Marshal(h)
		if strings.Contains(string(datos), "Vega") {
			t.Fatal("Historia con texto libre")
		}
	}
	h, err := s.ConsultarHistoriaIntervencion(ctx, a, id, "2026-02-02")
	if err != nil || h.Intervencion != nil || h.Vigencia != nil {
		t.Fatal(h, err)
	}
	actual, err := s.ConsultarIntervencion(ctx, a, id)
	if err != nil || *actual.Intervencion.Fase != "INTERVENCION" || (*actual.Intervencion.Listas["coordinacion"])[0].TextoOtros != "Servicio de apoyo de la Vega" {
		t.Fatal(actual, err)
	}
	i.Fase = texto("SEGUIMIENTO")
	guardar(5, "2026-04-01", i)
	h, err = s.ConsultarHistoriaIntervencion(ctx, a, id, "2026-04-01")
	if err != nil || h.Vigencia.Revision != 6 || h.FaseDesde == nil || *h.FaseDesde != "2026-04-01" {
		t.Fatal(h, err)
	}
	marcada := []dominio.MarcaIntervencion{{Codigo: "NEGLIGENCIA"}}
	i.Listas["tipologia"] = &marcada
	guardar(6, "2026-05-01", i)
	h, err = s.ConsultarHistoriaIntervencion(ctx, a, id, "2026-05-01")
	if err != nil || h.FaseDesde == nil || *h.FaseDesde != "2026-04-01" {
		t.Fatal(h, err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		consulta, err := (adaptador.ConsultaIntervencion{}).IntervencionVigente(ctx, id, "2026-03-15")
		if err != nil {
			return err
		}
		if consulta.Vigencia.Revision != 5 || *consulta.Intervencion.Fase != "RECEPCION" {
			t.Fatal(consulta)
		}
		e, _ := postgres.DesdeContexto(ctx)
		var n int
		if err = e.QueryRow(ctx, `SELECT count(*) FROM expediente.intervencion_revision WHERE expediente_id=$1`, id).Scan(&n); err != nil {
			return err
		}
		if n != 5 {
			t.Fatal(n)
		}
		// El plan real queda en la evidencia del ensayo, sin valores personales.
		rows, err := e.Query(ctx, `EXPLAIN SELECT datos FROM expediente.consulta_intervencion WHERE expediente_id=$1 AND vigente_desde<='2026-03-15'::date AND (vigente_hasta IS NULL OR '2026-03-15'::date<vigente_hasta)`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var plan string
			if err = rows.Scan(&plan); err != nil {
				return err
			}
			t.Log(plan)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Historia: límites exactos, retroactividad, corrección sin borrar revisiones, exclusión por menor, desconocido distinto de ninguno y consulta F4.05/F2.02 auditada")
}
func TestIntervencionRLSEquiposYVinculosCompuestos(t *testing.T) {
	b, ctx, d, a, id := prepararIntervencion(t)
	s := aplicacion.Nuevo(d)
	i := datosIntervencion()
	_, ctx2, d2, a2, id2 := prepararIntervencion(t)
	ajenos, err := aplicacion.Nuevo(d2).ConsultarMenores(ctx2, a2, id2)
	if err != nil {
		t.Fatal(err)
	}
	(*i.Listas["factor_menor"])[0].MenoresExcluidos = []string{ajenos.Menores[0].ID}
	p := dominio.CambioIntervencion{Version: 2, Desde: "2026-02-03", Intervencion: &i}
	if _, err = s.EditarIntervencion(ctx, a, id, "exclusion-ajena", p); !errors.Is(err, dominio.ErrIntervencion) {
		t.Fatal(err)
	}
	(*i.Listas["factor_menor"])[0].MenoresExcluidos = nil
	if _, err = s.EditarIntervencion(ctx, a, id, "guardar-rls", p); err != nil {
		t.Fatal(err)
	}
	propio := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	otro := a
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		return e.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE id<>$1 LIMIT 1`, a.EquipoID).Scan(&otro.EquipoID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConsultarIntervencion(ctx, otro, id); !errors.Is(err, dominio.ErrNoEncontrada) {
		t.Fatal(err)
	}
	for _, actor := range []dominio.Actor{otro, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, {}} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"intervencion_revision", "intervencion_marca", "intervencion_exclusion", "consulta_intervencion", "consulta_intervencion_vigencia"} {
				var n int
				if err := e.QueryRow(ctx, `SELECT count(*) FROM expediente.`+tabla+` WHERE expediente_id=$1`, id).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					t.Fatal(tabla, n)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `INSERT INTO expediente.intervencion_exclusion(revision_id,marca_id,menor_id,expediente_id,equipo_id) SELECT r.revision_id,r.marca_id,$2,r.expediente_id,r.equipo_id FROM expediente.intervencion_marca r WHERE r.expediente_id=$1 LIMIT 1`, id, ajenos.Menores[0].ID)
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "23503") {
		t.Fatal("FK no rechazó cruce", err)
	}
	for _, sql := range []string{`DELETE FROM expediente.intervencion_revision WHERE expediente_id=$1`, `UPDATE expediente.intervencion_revision SET fase='RECEPCION' WHERE expediente_id=$1`} {
		err = b.Ejecutar(ctx, propio, func(ctx context.Context) error {
			e, _ := postgres.DesdeContexto(ctx)
			_, err := e.Exec(ctx, sql, id)
			return err
		})
		if err == nil {
			t.Fatal("Historia editable")
		}
	}
	t.Log("RLS: sin contexto, equipo ajeno, Consulta y Administración sin filas; FK rechaza menor de otro expediente; revisiones no modificables")
}

func TestIntervencionCerradaRequiereReapertura(t *testing.T) {
	b, ctx, d, a, id := prepararIntervencion(t)
	r := &adaptador.Repositorio{}
	d.EmisionesFinalizacion = documentos.Repositorio{}
	d.Cierres = r
	d.Propuestas = r
	d.SalidasSPM = r
	d.MotivosCierre = &catalogo.Repositorio{}
	d.ConsultaCierreSPM = aplicacion.ConsultaCierreSPM{Repositorio: r, Salidas: r}
	s := aplicacion.Nuevo(d)
	i := datosIntervencion()
	if _, err := s.EditarIntervencion(ctx, a, id, "antes-cierre", dominio.CambioIntervencion{Version: 2, Desde: "2026-02-03", Intervencion: &i}); err != nil {
		t.Fatal(err)
	}
	final, err := s.GuardarFinalizacion(ctx, a, id, "final-cierre", dominio.Finalizacion{Version: 3, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true})
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := s.ProponerCierre(ctx, a, id, "proponer-cierre", dominio.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, coord := preparar(t)
	cerrado, err := s.Cerrar(ctx, coord, id, "cerrar-intervencion", dominio.PeticionCierre{Version: propuesta.Version, Fecha: "2026-03-01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditarIntervencion(ctx, a, id, "intervencion-cerrada", dominio.CambioIntervencion{Version: cerrado.Version, Desde: "2026-03-01", Intervencion: &i}); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `INSERT INTO expediente.intervencion_revision(expediente_id,equipo_id,version,vigente_desde,listas_conocidas,registrado_por,registrado_en) VALUES($1,$2,999,'2026-03-01','{}',$3,transaction_timestamp())`, id, a.EquipoID, a.UsuarioID)
		return err
	})
	if err == nil {
		t.Fatal("SQL permitió editar cerrado")
	}
	nuevo, err := s.Reabrir(ctx, coord, id, "reabrir-intervencion", dominio.PeticionReapertura{Version: cerrado.Version, Fecha: "2026-03-02", Motivo: "La familia solicita continuar el acompañamiento."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.EditarIntervencion(ctx, a, id, "intervencion-reabierta", dominio.CambioIntervencion{Version: nuevo.Version, Desde: "2026-03-02", Intervencion: &i}); err != nil {
		t.Fatal(err)
	}
	t.Log("Cerrado: edición bloqueada en aplicación y SQL; reapertura habilita edición sin borrar historia")
}
