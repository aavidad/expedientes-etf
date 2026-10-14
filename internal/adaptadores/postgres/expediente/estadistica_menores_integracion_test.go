package expediente_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestLecturaEstadisticaMenoresCorreccionRetroactivaYCoordinacion(t *testing.T) {
	b, ctx, deps, actor, id := prepararIntervencion(t)
	s := app.Nuevo(deps)
	ms := datosMenores()
	ms[0].Subprograma = "RIESGO_SOCIAL"
	if _, err := s.EditarMenores(ctx, actor, id, "lectura-menores-"+id, dominio.CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &ms}); err != nil {
		t.Fatal(err)
	}
	ficha, err := s.ConsultarMenores(ctx, actor, id)
	if err != nil {
		t.Fatal(err)
	}
	ms = ficha.Menores
	intervencion := datosIntervencion()
	if _, err := s.EditarIntervencion(ctx, actor, id, "lectura-coordinacion-"+id, dominio.CambioIntervencion{Version: 3, Desde: "2026-02-03", Intervencion: &intervencion}); err != nil {
		t.Fatal(err)
	}
	antesCorreccion := time.Now().UTC()
	ms[0].Sexo = "H"
	at := true
	ms[0].AtencionTemprana = &at
	if _, err := s.EditarMenores(ctx, actor, id, "lectura-correccion-"+id, dominio.CambioMenores{Version: 4, Desde: "2026-02-03", Menores: &ms}); err != nil {
		t.Fatal(err)
	}
	vacia := []dominio.MarcaIntervencion{}
	intervencion.Listas["coordinacion"] = &vacia
	if _, err := s.EditarIntervencion(ctx, actor, id, "lectura-coordinacion-vacia-"+id, dominio.CambioIntervencion{Version: 5, Desde: "2026-04-01", Intervencion: &intervencion}); err != nil {
		t.Fatal(err)
	}
	lector := &adaptador.LecturaEstadistica{}
	consultar := func(corte, conocimiento time.Time) d.Expediente {
		t.Helper()
		var resultado d.Expediente
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			rows, err := lector.ConsultarMemoria(ctx, actor.EquipoID, corte, conocimiento)
			for _, e := range rows {
				if e.ID == id {
					resultado = e
				}
			}
			return err
		})
		if err != nil || resultado.ID != id {
			t.Fatal(resultado, err)
		}
		return resultado
	}
	febrero := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	actual := time.Now().UTC()
	anterior := consultar(febrero, antesCorreccion)
	antiguos, _, err := anterior.Menores.ValorEn(febrero, antesCorreccion)
	if err != nil || !antiguos.Conocido || len(antiguos.Valor) != 2 || antiguos.Valor[0].Sexo != d.Nina || antiguos.Valor[0].PropuestaAtencionTemprana.Conocido {
		t.Fatal(antiguos, err)
	}
	corregido := consultar(febrero, actual)
	menores, _, err := corregido.Menores.ValorEn(febrero, actual)
	if err != nil || !menores.Conocido || len(menores.Valor) != 2 || menores.Valor[0].Sexo != d.Nino || !menores.Valor[0].PropuestaAtencionTemprana.Conocido || !menores.Valor[0].PropuestaAtencionTemprana.Valor || menores.Valor[1].Subprograma != d.Reunificacion {
		t.Fatal(menores, err)
	}
	recursos, _, err := corregido.RecursosCoordinacion.ValorEn(febrero, actual)
	if err != nil || !recursos.Conocido || len(recursos.Valor) != 1 || recursos.Valor[0] != "OTROS" {
		t.Fatal(recursos, err)
	}
	abril := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	e := consultar(abril, actual)
	recursos, _, err = e.RecursosCoordinacion.ValorEn(abril, actual)
	if err != nil || !recursos.Conocido || len(recursos.Valor) != 0 {
		t.Fatal(recursos, err)
	}
	excluida := []dominio.MarcaIntervencion{{Codigo: "OTROS", TextoOtros: "Recurso municipal de apoyo", MenoresExcluidos: []string{ms[0].ID}}}
	intervencion.Listas["coordinacion"] = &excluida
	if _, err := s.EditarIntervencion(ctx, actor, id, "lectura-coordinacion-excluida-"+id, dominio.CambioIntervencion{Version: 6, Desde: "2026-04-01", Intervencion: &intervencion}); err != nil {
		t.Fatal(err)
	}
	posterior := time.Now().UTC()
	conExclusion := consultar(abril, posterior)
	recursos, _, err = conExclusion.RecursosCoordinacion.ValorEn(abril, posterior)
	if err != nil || !recursos.Conocido || len(recursos.Valor) != 1 || recursos.Valor[0] != "OTROS" {
		t.Fatal("se perdió la lista valorada con exclusiones por menor", recursos, err)
	}
	factores, _, err := conExclusion.Factores.ValorEn(abril, posterior)
	if err != nil || !factores.Conocido {
		t.Fatal("no se recuperaron las marcas valoradas", factores, err)
	}
	exclusionConservada := false
	for _, factor := range factores.Valor {
		if factor.Lista == "coordinacion" && factor.Codigo == "OTROS" {
			exclusionConservada = len(factor.MenoresExcluidos) == 1 && factor.MenoresExcluidos[0] == ms[0].ID
		}
	}
	if !exclusionConservada {
		t.Fatal("se perdió la exclusión individual de coordinación", factores)
	}
	conocimientoAnterior := consultar(abril, actual)
	recursos, _, err = conocimientoAnterior.RecursosCoordinacion.ValorEn(abril, actual)
	if err != nil || !recursos.Conocido || len(recursos.Valor) != 0 {
		t.Fatal("corrección de coordinación contaminó el conocimiento anterior", recursos, err)
	}
	sinFuente := consultar(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), actual)
	if len(sinFuente.Menores) != 0 || len(sinFuente.RecursosCoordinacion) != 0 {
		t.Fatal("se incorporó una revisión posterior al corte")
	}
	serializado, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, texto := range []string{"Lucía", "Moreno Castillo", "Servicio de apoyo de la Vega", "texto_otros", "nombre", "apellidos"} {
		if strings.Contains(string(serializado), texto) {
			t.Fatal("se expuso identidad o texto libre")
		}
	}
	if len(e.Propuestas) != 0 || len(e.Responsables) != 0 {
		t.Fatal("se dedujeron datos sin fuente validada")
	}
	t.Log("PostgreSQL: menores y coordinación por corte y conocimiento; corrección retroactiva; vacío conocido; exclusiones sin sobreconteo; sin nombres ni textos libres")
}
