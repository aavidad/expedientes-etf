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

type repoSeguimiento struct{ repositorioPrueba }

func (*repoSeguimiento) Datos(context.Context, d.Version) (d.Datos, error) {
	return d.Datos{Opciones: []d.Opcion{{Lista: "fin_seguimiento", Codigo: "CAMBIO", Activa: true}, {Lista: "fin_seguimiento", Codigo: "RETIRADO"}, {Lista: "otra", Codigo: "OTRA", Activa: true}}}, nil
}

type textosSeguimiento struct{ paquete.PublicacionCatalogos }

func (textosSeguimiento) Catalogos(context.Context) (paquete.Catalogos, error) {
	return paquete.Catalogos{Textos: map[string]map[string]string{"es": {"fin_seguimiento.CAMBIO": "Cambio de situación familiar", "fin_seguimiento.RETIRADO": "Motivo retirado"}}}, nil
}
func TestOpcionesSeguimientoFiltraAutorizaYNoEntregaSinAuditoria(t *testing.T) {
	c := ConsultaSeguimiento{ConsultaDecision{Repositorio: &repoSeguimiento{}, Textos: textosSeguimiento{}, Transacciones: &txPrueba{}, Registro: registroOpciones{}, Ahora: time.Now}}
	ctx := context.Background()
	for _, a := range []entrada.Actor{{UsuarioID: "persona", Perfiles: []string{"consulta"}}, {UsuarioID: "persona", Perfiles: []string{"administracion"}}, {UsuarioID: "persona", Perfiles: []string{"profesional_etf"}}, {Perfiles: []string{"coordinacion"}}} {
		if _, err := c.ConsultarOpcionesSeguimiento(ctx, a, "es"); !errors.Is(err, d.ErrPermiso) {
			t.Fatalf("permiso: %v", err)
		}
	}
	for _, a := range []entrada.Actor{{UsuarioID: "persona", Perfiles: []string{"coordinacion"}}, {UsuarioID: "persona", Perfiles: []string{"profesional_etf"}, EquipoID: "equipo"}} {
		r, err := c.ConsultarOpcionesSeguimiento(ctx, a, "en")
		if err != nil || len(r.Motivos) != 1 || r.Textos["fin_seguimiento.CAMBIO"] != "Cambio de situación familiar" || r.Textos["fin_seguimiento.RETIRADO"] != "Motivo retirado" {
			t.Fatalf("opciones: %+v, %v", r, err)
		}
	}
	c.Registro = registroOpciones{fallo: errors.New("auditoria.no_disponible")}
	r, err := c.ConsultarOpcionesSeguimiento(ctx, entrada.Actor{UsuarioID: "persona", Perfiles: []string{"coordinacion"}}, "es")
	if err == nil || len(r.Motivos) != 0 || len(r.Textos) != 0 {
		t.Fatalf("datos sin auditoría: %+v, %v", r, err)
	}
}
