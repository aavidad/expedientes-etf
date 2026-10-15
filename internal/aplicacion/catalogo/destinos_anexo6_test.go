package catalogo

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"testing"
	"time"
)

type repoDestinosAnexo6 struct{ repositorioPrueba }

func (*repoDestinosAnexo6) Datos(context.Context, d.Version) (d.Datos, error) {
	return d.Datos{Opciones: []d.Opcion{{Lista: "servicio_destino", Codigo: "CAMBIO", Activa: true}, {Lista: "servicio_destino", Codigo: "RETIRADO"}, {Lista: "otra", Codigo: "OTRA", Activa: true}}}, nil
}

type textosDestinosAnexo6 struct{ paquete.PublicacionCatalogos }

func (textosDestinosAnexo6) Catalogos(context.Context) (paquete.Catalogos, error) {
	return paquete.Catalogos{Textos: map[string]map[string]string{"es": {"servicio_destino.CAMBIO": "Cambio de situación familiar", "servicio_destino.RETIRADO": "Motivo retirado"}}}, nil
}
func TestOpcionesDestinosAnexo6FiltraAutorizaYNoEntregaSinAuditoria(t *testing.T) {
	c := ConsultaDestinosAnexo6{ConsultaDecision{Repositorio: &repoDestinosAnexo6{}, Textos: textosDestinosAnexo6{}, Transacciones: &txPrueba{}, Registro: registroOpciones{}, Ahora: time.Now}}
	ctx := context.Background()
	for _, a := range []entrada.Actor{{UsuarioID: "persona", Perfiles: []string{"consulta"}}, {UsuarioID: "persona", Perfiles: []string{"administracion"}}, {UsuarioID: "persona", Perfiles: []string{"profesional_etf"}}, {Perfiles: []string{"coordinacion"}}} {
		if _, err := c.ConsultarDestinosAnexo6(ctx, a, "es"); !errors.Is(err, d.ErrPermiso) {
			t.Fatalf("permiso: %v", err)
		}
	}
	for _, a := range []entrada.Actor{{UsuarioID: "persona", Perfiles: []string{"coordinacion"}}, {UsuarioID: "persona", Perfiles: []string{"profesional_etf"}, EquipoID: "equipo"}} {
		r, err := c.ConsultarDestinosAnexo6(ctx, a, "en")
		if err != nil || len(r.Servicios) != 1 || r.Textos["servicio_destino.CAMBIO"] != "Cambio de situación familiar" || r.Textos["servicio_destino.RETIRADO"] != "Motivo retirado" {
			t.Fatalf("opciones: %+v, %v", r, err)
		}
	}
	c.Registro = registroOpciones{fallo: errors.New("auditoria.no_disponible")}
	r, err := c.ConsultarDestinosAnexo6(ctx, entrada.Actor{UsuarioID: "persona", Perfiles: []string{"coordinacion"}}, "es")
	if err == nil || len(r.Servicios) != 0 || len(r.Textos) != 0 {
		t.Fatalf("datos sin auditoría: %+v, %v", r, err)
	}
}
