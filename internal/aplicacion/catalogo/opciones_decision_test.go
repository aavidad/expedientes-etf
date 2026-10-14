package catalogo

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"testing"
	"time"
)

type repoOpciones struct{ repositorioPrueba }

func (*repoOpciones) Datos(_ context.Context, v dominio.Version) (dominio.Datos, error) {
	return dominio.Datos{Equipos: []dominio.Equipo{{ID: "equipo", ClaveTexto: "equipo.01", ActivoDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, Opciones: []dominio.Opcion{{Lista: "subprograma", Codigo: "RIESGO_SOCIAL", Activa: true}, {Lista: "subprograma", Codigo: "RETIRADA", Activa: false}, {Lista: "otra", Codigo: "OTRA", Activa: true}}}, nil
}

type textosOpciones struct{ paquete.PublicacionCatalogos }

func (textosOpciones) Catalogos(context.Context) (paquete.Catalogos, error) {
	return paquete.Catalogos{Textos: map[string]map[string]string{"es": {"equipo.01": "Equipo provincial", "subprograma.RIESGO_SOCIAL": "Riesgo social"}, "en": {"subprograma.RIESGO_SOCIAL": "Social risk"}}}, nil
}

type registroOpciones struct {
	registro.RegistroTransaccional
	fallo error
}

func (r registroOpciones) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, r.fallo
}
func TestOpcionesDecisionPermisoFiltradoYConfirmacion(t *testing.T) {
	ctx := context.Background()
	repo := &repoOpciones{}
	tx := &txPrueba{}
	c := ConsultaDecision{Repositorio: repo, Textos: textosOpciones{}, Transacciones: tx, Registro: registroOpciones{}, Ahora: func() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }}
	for _, perfil := range []string{"administracion", "consulta", "profesional_etf"} {
		if _, err := c.ConsultarOpcionesDecision(ctx, entrada.Actor{UsuarioID: "persona", Perfiles: []string{perfil}}, "es"); !errors.Is(err, dominio.ErrPermiso) {
			t.Fatalf("perfil %s: %v", perfil, err)
		}
	}
	actor := entrada.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", Perfiles: []string{"coordinacion"}}
	r, err := c.ConsultarOpcionesDecision(ctx, actor, "en")
	if err != nil || len(r.Subprogramas) != 1 || r.Textos["equipo.01"] != "Equipo provincial" || r.Textos["subprograma.RIESGO_SOCIAL"] != "Social risk" || tx.identidad.UsuarioID != actor.UsuarioID {
		t.Fatal(r, err)
	}
	tx.fallo = errors.New("prueba.confirmacion")
	r, err = c.ConsultarOpcionesDecision(ctx, actor, "es")
	if err == nil || r.Textos != nil {
		t.Fatal("datos entregados sin commit")
	}
	tx.fallo = nil
	c.Registro = registroOpciones{fallo: errors.New("prueba.auditoria")}
	r, err = c.ConsultarOpcionesDecision(ctx, actor, "es")
	if err == nil || r.Textos != nil {
		t.Fatal("datos entregados sin auditoría")
	}
}
