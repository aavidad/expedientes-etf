package expediente_test

import (
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"os"
	"testing"
	"time"
)

func TestConservacionHistoricoConEpisodios(t *testing.T) {
	if os.Getenv("ETF_CONSERVACION_INTEGRACION") != "1" {
		t.Skip("ETF_CONSERVACION_INTEGRACION=1")
	}
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	caso := casoHistorico(p, numeroHistoricoLibre(t, b, ctx, coord, p))
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{caso}}
	vista, err := s.ValidarCargaHistorica(ctx, coord, carga)
	if err != nil {
		t.Fatal(err)
	}
	lote, err := s.CargarHistoricos(ctx, coord, vista.LoteID, "conservacion-historico-"+vista.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	id := lote.Expedientes[0]
	_, _, _, otra := preparar(t)
	admin := d.Actor{UsuarioID: coord.UsuarioID, Perfiles: []string{"administracion"}}
	segunda := d.Actor{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}
	conservar := &app.ServicioConservacion{Repositorio: &repo.Repositorio{}, Catalogo: &catalogo.Repositorio{}, Defecto: nombres.Conservacion{Ruta: "../../../../datos/catalogos/conservacion.json"}, Reloj: &relojConservacion{time.Date(2051, 5, 1, 12, 0, 0, 0, time.UTC)}, Transacciones: b, Registro: postgres.NuevoRegistroAuditoria(b)}
	filas, err := conservar.PendientesConservacion(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	var fila d.Conservacion
	for _, v := range filas {
		if v.ExpedienteID == id {
			fila = v
		}
	}
	if fila.Fin != "2051-05-01" {
		t.Fatal("no usa último episodio", fila)
	}
	propuesta, err := conservar.ConfirmarConservacion(ctx, admin, id, fila.Version, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conservar.ConfirmarConservacion(ctx, segunda, id, propuesta.Version, true); err != nil {
		t.Fatal("retirada con restricciones diferidas y sin acceso a ficha", err)
	}
	var episodios int
	var libres int
	consultaSQL(t, b, ctx, coord, `SELECT count(*),count(*) FILTER(WHERE motivo_reapertura IS NOT NULL AND motivo_reapertura<>'—') FROM expediente.episodio WHERE expediente_id=$1`, []any{id}, &episodios, &libres)
	if episodios != 2 || libres != 0 {
		t.Fatal("pierde episodios o conserva textos", episodios, libres)
	}
	t.Log("Carga histórica con dos episodios: vence desde último cierre; restricciones diferidas y retirada confirmada: OK")
}
