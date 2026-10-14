package documento_test

import (
	"context"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"testing"
	"time"
)

type idiomasConsultaDecision struct{ paquete.Paquetes }

func (idiomasConsultaDecision) Idiomas(context.Context) ([]paquete.Idioma, error) {
	return []paquete.Idioma{{Codigo: "es"}, {Codigo: "en"}}, nil
}

type memoriaConsultaDecision struct{ *memoria }

func (m memoriaConsultaDecision) Ultima(_ context.Context, p d.Solicitud) (d.Emision, error) {
	var ultima d.Emision
	for _, e := range m.emisiones {
		if e.ExpedienteID == p.ExpedienteID && e.Tipo == p.Tipo && e.Plantilla.Idioma == p.Idioma && e.GeneradaEn.After(ultima.GeneradaEn) {
			ultima = e
		}
	}
	return ultima, nil
}
func TestConsultaDecisionEmisionesPermisoRegistroYAuditoria(t *testing.T) {
	ahora := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	m := &memoria{emisiones: []d.Emision{{ID: "33333333-3333-4333-8333-333333333333", ExpedienteID: expediente, Tipo: d.OficioSSCC, GeneradaEn: ahora, Plantilla: d.Plantilla{Idioma: "es", Version: "1"}}}, registros: map[string]d.RegistroSalida{"33333333-3333-4333-8333-333333333333": {Version: 2, Estado: "anotado", Numero: "2026/14873", Fecha: "2026-10-13"}}}
	c := app.ConsultaDecision{Idiomas: idiomasConsultaDecision{}, Repositorio: memoriaConsultaDecision{m}, Registros: m, Transacciones: m, Auditoria: m, Ahora: func() time.Time { return ahora }}
	actor := d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	r, err := c.ConsultarEmisionesDecision(context.Background(), actor, expediente)
	if err != nil || len(r) != 1 || r[0].Registro.Numero != "2026/14873" {
		t.Fatal(r, err)
	}
	for _, perfil := range []string{"administracion", "consulta", "profesional_etf"} {
		actor.Perfiles = []string{perfil}
		if _, err := c.ConsultarEmisionesDecision(context.Background(), actor, expediente); err == nil {
			t.Fatal(perfil)
		}
	}
	actor.Perfiles = []string{"coordinacion"}
	m.fallaAuditoria = true
	r, err = c.ConsultarEmisionesDecision(context.Background(), actor, expediente)
	if err == nil || r != nil {
		t.Fatal("datos sin auditoría")
	}
	m.fallaAuditoria = false
	m.fallaCommit = true
	r, err = c.ConsultarEmisionesDecision(context.Background(), actor, expediente)
	if err == nil || r != nil {
		t.Fatal("datos sin commit")
	}
}
