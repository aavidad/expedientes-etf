package estadistica_test

import (
	"context"
	"errors"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	catalogo "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"testing"
	"time"
)

type catalogoOpcionesSPAF struct{ salida.Repositorio }

func (catalogoOpcionesSPAF) Vigente(ctx context.Context, _ string, _ time.Time) (catalogo.Version, error) {
	if ctx.Value(claveTxSPAF{}) != true {
		return catalogo.Version{}, errors.New("sin_transaccion")
	}
	return catalogo.Version{}, nil
}
func (catalogoOpcionesSPAF) Datos(context.Context, catalogo.Version) (catalogo.Datos, error) {
	return catalogo.Datos{Equipos: []catalogo.Equipo{{ID: equipoSPAF, ClaveTexto: "equipo.02"}, {ID: otroEquipoSPAF, ClaveTexto: "equipo.03"}}}, nil
}

type etiquetasOpcionesSPAF struct{ paquete.PublicacionCatalogos }

func (etiquetasOpcionesSPAF) Catalogos(context.Context) (paquete.Catalogos, error) {
	return paquete.Catalogos{Textos: map[string]map[string]string{"es": {"equipo.02": "ETF 02"}}}, nil
}

func TestSPAFOpcionesAmbitoYAuditoriaAntesDeEntregar(t *testing.T) {
	for _, c := range []struct {
		perfil                  string
		fallaAudit, fallaCommit bool
		equipos                 int
		fallo                   error
	}{
		{"coordinacion", false, false, 2, nil}, {"profesional_etf", false, false, 1, nil},
		{"consulta", false, false, 0, d.ErrPermisoSPAF}, {"administracion", false, false, 0, d.ErrPermisoSPAF},
		{"coordinacion", true, false, 0, d.ErrDisponibleSPAF}, {"coordinacion", false, true, 0, d.ErrDisponibleSPAF},
	} {
		t.Run(c.perfil, func(t *testing.T) {
			tx := &transaccionesSPAF{errorCommit: c.fallaCommit}
			audit := &auditoriaSPAF{errorRegistro: c.fallaAudit}
			s := app.NuevoSPAF(app.DependenciasSPAF{Transacciones: tx, Auditoria: audit, Reloj: relojSPAF{}, Catalogo: catalogoOpcionesSPAF{}, Etiquetas: etiquetasOpcionesSPAF{}})
			r, err := s.ConsultarOpcionesSPAF(context.Background(), d.ActorSPAF{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{c.perfil}})
			if !errors.Is(err, c.fallo) || len(r.Equipos) != c.equipos {
				t.Fatalf("equipos=%d err=%v", len(r.Equipos), err)
			}
			if err == nil && (len(audit.datos) != 1 || audit.datos[0].Accion != "estadistica.consultar_opciones_spaf" || r.Textos["es"]["equipo.02"] == "") {
				t.Fatal("opciones sin auditoría o etiquetas")
			}
			if c.perfil == "profesional_etf" && r.Equipos[0].ID != equipoSPAF {
				t.Fatal("equipo ajeno")
			}
			if err != nil && r.Textos != nil {
				t.Fatal("datos tras fallo")
			}
		})
	}
}
