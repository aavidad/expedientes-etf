package catalogo

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"testing"
	"time"
)

func TestOpcionesCargaFiltraEquipoYNoAmpliaDecision(t *testing.T) {
	c := ConsultaDecision{Repositorio: &repoOpciones{}, Textos: textosOpciones{}, Transacciones: &txPrueba{}, Registro: registroOpciones{}, Ahora: func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }}
	a := entrada.Actor{UsuarioID: "persona", EquipoID: "equipo", Perfiles: []string{"profesional_etf"}}
	r, err := c.ConsultarOpcionesCarga(context.Background(), a, "en")
	if err != nil || len(r.Equipos) != 1 || r.Textos["equipo.01"] != "Equipo provincial" {
		t.Fatal(r, err)
	}
	a.EquipoID = "otro"
	r, err = c.ConsultarOpcionesCarga(context.Background(), a, "es")
	if err != nil || len(r.Equipos) != 0 {
		t.Fatal(r, err)
	}
	if _, err = c.ConsultarOpcionesDecision(context.Background(), a, "es"); !errors.Is(err, d.ErrPermiso) {
		t.Fatal("el permiso de decisión se ha ampliado", err)
	}
	for _, perfil := range []string{"consulta", "administracion", "profesional_etf"} {
		if _, err = c.ConsultarOpcionesCarga(context.Background(), entrada.Actor{UsuarioID: "persona", Perfiles: []string{perfil}}, "es"); !errors.Is(err, d.ErrPermiso) {
			t.Fatal(perfil, err)
		}
	}
	c.Registro = registroOpciones{fallo: errors.New("prueba.auditoria")}
	if r, err = c.ConsultarOpcionesCarga(context.Background(), entrada.Actor{UsuarioID: "persona", Perfiles: []string{"coordinacion"}}, "es"); err == nil || r.Textos != nil {
		t.Fatal("datos entregados sin auditoría")
	}
}
