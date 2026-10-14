package expediente_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestLecturaEstadisticaColeccionesCortesYRecuentosManuales(t *testing.T) {
	b, ctx, deps, a, id := prepararIntervencion(t)
	s := app.Nuevo(deps)
	si, no := true, false
	ms := []dominio.Menor{
		{Orden: 1, Nombre: "Lucía", Apellidos: "Moreno Castillo", Sexo: "M", Nacimiento: texto("2016-05-14"), AtencionTemprana: &si},
		{Orden: 2, Nombre: "Pablo", Apellidos: "Moreno Castillo", Sexo: "H", Subprograma: "REUNIFICACION", Nacimiento: texto("2012-03-02"), AtencionTemprana: &no},
		{Orden: 3, Nombre: "Elena", Apellidos: "Moreno Castillo", Sexo: "M", Nacimiento: texto("2008-01-01")},
	}
	r, err := s.EditarMenores(ctx, a, id, "estadistica-menores-"+id, dominio.CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &ms})
	if err != nil {
		t.Fatal(err)
	}
	menores, err := s.ConsultarMenores(ctx, a, id)
	if err != nil {
		t.Fatal(err)
	}
	excluido := menores.Menores[1].ID
	i := datosIntervencion()
	recursos := []dominio.MarcaIntervencion{{Codigo: "SALUD_MENTAL", MenoresExcluidos: []string{menores.Menores[0].ID, excluido}}}
	factores := []dominio.MarcaIntervencion{{Codigo: "SALUD_MENTAL", MenoresExcluidos: []string{excluido}}}
	i.Listas["coordinacion"], i.Listas["factor_menor"] = &recursos, &factores
	r, err = s.EditarIntervencion(ctx, a, id, "estadistica-intervencion-"+id, dominio.CambioIntervencion{Version: r.Version, Desde: "2026-02-03", Intervencion: &i})
	if err != nil {
		t.Fatal(err)
	}
	conocido := time.Now().UTC()
	ms = menores.Menores[:1]
	r, err = s.EditarMenores(ctx, a, id, "estadistica-retirada-"+id, dominio.CambioMenores{Version: r.Version, Desde: "2026-05-01", Menores: &ms})
	if err != nil {
		t.Fatal(err)
	}
	ms = []dominio.Menor{}
	r, err = s.EditarMenores(ctx, a, id, "estadistica-vacia-"+id, dominio.CambioMenores{Version: r.Version, Desde: "2026-05-01", Menores: &ms})
	if err != nil {
		t.Fatal(err)
	}
	i.Fase = texto("INTERVENCION")
	recursos[0].MenoresExcluidos = nil
	factores[0].MenoresExcluidos = nil
	r, err = s.EditarIntervencion(ctx, a, id, "estadistica-fase-"+id, dominio.CambioIntervencion{Version: r.Version, Desde: "2026-05-01", Intervencion: &i})
	if err != nil {
		t.Fatal(err)
	}
	lector := &adaptador.LecturaEstadistica{}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		for _, caso := range []struct {
			corte    string
			conocido time.Time
			cantidad int
		}{
			{"2026-04-30", time.Now().UTC(), 3}, {"2026-06-30", conocido, 3}, {"2026-06-30", time.Now().UTC(), 0},
		} {
			corte, _ := time.Parse("2006-01-02", caso.corte)
			filas, err := lector.ConsultarMemoria(ctx, a.EquipoID, corte, caso.conocido)
			if err != nil {
				return err
			}
			var exp d.Expediente
			for _, fila := range filas {
				if fila.ID == id {
					exp = fila
				}
			}
			lista, _, err := exp.Menores.ValorEn(corte, caso.conocido)
			if err != nil || !lista.Conocido || len(lista.Valor) != caso.cantidad {
				t.Fatal("cortes", lista, caso, err)
			}
			spaf, _, err := (&adaptador.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}).ConsultarSPAF(ctx, d.FiltroSPAF{Anio: 2026, Mes: int(corte.Month()), EquipoID: a.EquipoID}, corte, caso.conocido)
			if err != nil {
				return err
			}
			for _, fila := range spaf {
				if fila.ID != id {
					continue
				}
				ficha, _, err := fila.Ficha.ValorEn(corte, caso.conocido)
				esperada := "EVALUACION"
				if caso.cantidad == 0 {
					esperada = "INTERVENCION"
				}
				if err != nil || !ficha.Valor.FaseIntervencion.Conocido || ficha.Valor.FaseIntervencion.Valor != esperada || !ficha.Valor.Menores.Conocido || len(ficha.Valor.Menores.Valor) != caso.cantidad {
					t.Fatal("SPAF al corte", ficha, err)
				}
			}
			b, _ := json.Marshal(exp)
			for _, prohibido := range []string{"Lucía", "Moreno", "Vega", "Servicio de apoyo"} {
				if strings.Contains(string(b), prohibido) {
					t.Fatal("identidad en Memoria")
				}
			}
			if caso.cantidad == 0 {
				continue
			}
			// Tres menores: dos niñas de Riesgo (una cumple 18 en 2026) y
			// un niño de Reunificación excluido de recurso y factor.
			tramos := []d.Tramo{{Codigo: "todos", DesdeMeses: 0}}
			params := d.Parametros{TramosRiesgo: tramos, TramosReunificacion: tramos, RecursosCoordinacion: []string{"SALUD_MENTAL"}, FactoresFamilia: []string{"DESEMPLEO"}, FactoresMenores: []string{"SALUD_MENTAL"}, TipologiasDesproteccion: []string{"NEGLIGENCIA"}}
			reglas := d.Historia[d.Parametros]{{ID: "reglas", Vigencia: d.Vigencia{Desde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, RegistradoEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Dato: d.Dato[d.Parametros]{Conocido: true, Valor: params}}}
			memoria, err := d.Calcular(d.Entrada{Anio: 2026, FechaCalculo: corte, ConocidoEn: caso.conocido, Reglas: reglas, Expedientes: []d.Expediente{exp}})
			if err != nil {
				t.Fatal(err)
			}
			if memoria.Apartados[d.MenoresPorSexo].Total.Menores != 3 {
				t.Fatal("total manual", memoria.Apartados[d.MenoresPorSexo])
			}
			for _, c := range []struct {
				apartado        d.Apartado
				grupo           d.Grupo
				unidad          d.Unidad
				numero, sinDato int
			}{
				{d.MenoresPorSexo, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina}, d.Menores, 2, 0},
				{d.MenoresPorSexo, d.Grupo{Subprograma: d.Reunificacion, Sexo: d.Nino}, d.Menores, 1, 0},
				{d.MenoresPorEdad, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Tramo: "edad_6_11"}, d.Menores, 1, 0},
				{d.MenoresPorEdad, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Tramo: "edad_12_17"}, d.Menores, 1, 0},
				{d.MenoresPorEdad, d.Grupo{Subprograma: d.Reunificacion, Sexo: d.Nino, Tramo: "edad_12_17"}, d.Menores, 1, 0},
				{d.CoordinacionRecursos, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Estado: "SALUD_MENTAL"}, d.Menores, 1, 0},
				{d.CoordinacionRecursos, d.Grupo{Subprograma: d.Reunificacion, Sexo: d.Nino, Estado: "SALUD_MENTAL"}, d.Menores, 0, 0},
				{d.FactoresMenores, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Estado: "SALUD_MENTAL"}, d.Menores, 2, 0},
				{d.FactoresMenores, d.Grupo{Subprograma: d.Reunificacion, Sexo: d.Nino, Estado: "SALUD_MENTAL"}, d.Menores, 0, 0},
				{d.FactoresFamilia, d.Grupo{Subprograma: d.Riesgo, Estado: "DESEMPLEO"}, d.Familias, 0, 1},
				{d.TipologiasDesproteccion, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Estado: "NEGLIGENCIA"}, d.Menores, 0, 0},
				{d.AtencionTemprana, d.Grupo{Subprograma: d.Riesgo, Sexo: d.Nina, Estado: "si"}, d.Menores, 1, 0},
			} {
				celda := memoria.Apartados[c.apartado].Celdas[d.ClaveCelda{Grupo: c.grupo, Unidad: c.unidad}]
				if celda.Numerador != c.numero || celda.SinDato != c.sinDato {
					t.Fatal("recuento manual", c, celda)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Memoria: 3 menores, 2 niñas Riesgo y 1 niño Reunificación; exclusiones, listas sin valorar/vacías y correcciones al corte comprobadas")
}

func TestLecturaSPAFJulioSeisMasTresSonNueve(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps.Menores = &adaptador.Repositorio{}
	deps.CatalogosMenores = nombres.CatalogosMenores{Ruta: "../../../../idiomas/es/catalogos/menores.json"}
	s := app.Nuevo(deps)
	p.FechaActa = "2026-07-31"
	ids := []string{}
	for n := 0; n < 2; n++ {
		recibida := subir(t, ctx, deps, a)
		aceptada, err := s.Aceptar(ctx, a, recibida.ID, "julio-"+recibida.ID, conMenores(p, recibida))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, aceptada.ID)
	}
	a.Perfiles, a.EquipoID = []string{"profesional_etf"}, p.EquipoID
	menores, err := s.ConsultarMenores(ctx, a, ids[1])
	if err != nil || len(menores.Menores) != 6 {
		t.Fatal("seis menores de origen", menores, err)
	}
	ms := menores.Menores[:3]
	if _, err = s.EditarMenores(ctx, a, ids[1], "julio-tres-"+ids[1], dominio.CambioMenores{Version: menores.Version, Desde: "2026-07-31", Menores: &ms}); err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		ahora := time.Now().UTC()
		corte := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
		filas, reglas, err := (&adaptador.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}).ConsultarSPAF(ctx, d.FiltroSPAF{Anio: 2026, Mes: 7, EquipoID: a.EquipoID}, corte, ahora)
		if err != nil {
			return err
		}
		propias := []d.ExpedienteSPAF{}
		for _, fila := range filas {
			if fila.ID == ids[0] || fila.ID == ids[1] {
				propias = append(propias, fila)
			}
		}
		tabla, err := d.CalcularSPAF(d.EntradaSPAF{Filtro: d.FiltroSPAF{Anio: 2026, Mes: 7, EquipoID: a.EquipoID}, FechaCalculo: ahora, ConocidoEn: ahora, Reglas: reglas, Expedientes: propias})
		if err != nil {
			return err
		}
		if tabla.Resumen.Casos != 2 || tabla.Resumen.Altas != 2 || tabla.Resumen.Menores != 9 || tabla.Resumen.CasosSinDatoMenores != 0 {
			t.Fatal("SPAF manual julio", tabla.Resumen)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SPAF julio: 2 altas, 2 casos y 6 + 3 = 9 menores conocidos")
}
