package expediente_test

import (
	"context"
	"errors"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"os"
	"testing"
	"time"
)

func TestConservacionCambioCatalogoYReapertura(t *testing.T) {
	if os.Getenv("ETF_CONSERVACION_INTEGRACION") != "1" {
		t.Skip("ETF_CONSERVACION_INTEGRACION=1")
	}
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	r := &repo.Repositorio{}
	f, peticion := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "OBJ_TOTAL")
	cierre, err := app.Nuevo(deps).Cerrar(ctx, otra, f.ID, "cerrar-plazo-"+f.ID, peticion)
	if err != nil {
		t.Fatal(err)
	}
	admin := d.Actor{UsuarioID: coord.UsuarioID, Perfiles: []string{"administracion"}}
	segundo := d.Actor{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}
	reloj := &relojConservacion{time.Date(2051, 10, 1, 12, 0, 0, 0, time.UTC)}
	s := &app.ServicioConservacion{Repositorio: r, Catalogo: &catalogo.Repositorio{}, Defecto: nombres.Conservacion{Ruta: "../../../../datos/catalogos/conservacion.json"}, Reloj: reloj, Transacciones: b, Registro: postgres.NuevoRegistroAuditoria(b)}
	buscar := func() *d.Conservacion {
		t.Helper()
		filas, err := s.PendientesConservacion(ctx, admin)
		if err != nil {
			t.Fatal(err)
		}
		for _, fila := range filas {
			if fila.ExpedienteID == f.ID {
				return &fila
			}
		}
		return nil
	}
	fila := buscar()
	if fila == nil {
		t.Fatal("falta caso vencido")
	}
	propuesta, err := s.ConfirmarConservacion(ctx, admin, f.ID, fila.Version, false)
	if err != nil {
		t.Fatal(err)
	}
	publicar := func(anios int, desde string) {
		t.Helper()
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: admin.UsuarioID, Perfiles: admin.Perfiles}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			_, err := tx.Exec(ctx, `SELECT catalogo.editar('plazos',(SELECT max(numero) FROM catalogo.version WHERE ambito='plazos' AND estado='publicada'),$1::date,jsonb_build_object('clave','conservacion','valor',$2::integer,'unidad','anos'))`, desde, anios)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	publicar(30, "2051-10-02")
	reloj.fecha = time.Date(2051, 10, 2, 12, 0, 0, 0, time.UTC)
	if fila := buscar(); fila != nil {
		t.Fatal("ignora ampliación a 30 años", fila)
	}
	if _, err := s.ConfirmarConservacion(ctx, segundo, f.ID, propuesta.Version, true); !errors.Is(err, d.ErrConservacion) {
		t.Fatal("usa propuesta tras ampliar plazo", err)
	}
	reloj.fecha = time.Date(2056, 10, 1, 12, 0, 0, 0, time.UTC)
	fila = buscar()
	if fila == nil || fila.Anios != 30 || fila.Fin != "2056-10-01" || fila.Estado != "pendiente_propuesta" {
		t.Fatal(fila)
	}
	nueva, err := s.ConfirmarConservacion(ctx, admin, f.ID, fila.Version, false)
	if err != nil {
		t.Fatal(err)
	}
	deps.Reloj = reloj
	if _, err = app.Nuevo(deps).Reabrir(ctx, otra, f.ID, "reabrir-plazo-"+f.ID, d.PeticionReapertura{Version: cierre.Version, Fecha: "2056-10-01", Motivo: "Se revisa la necesidad de retomar la intervención."}); err != nil {
		t.Fatal(err)
	}
	if fila := buscar(); fila != nil {
		t.Fatal("lista expediente reabierto")
	}
	if _, err = s.ConfirmarConservacion(ctx, segundo, f.ID, nueva.Version, true); !errors.Is(err, d.ErrConservacion) {
		t.Fatal("anonimiza expediente reabierto", err)
	}
	t.Log("Catálogo publicado a 30 años: fecha 2056-10-01; propuesta invalidada; reapertura bloquea retirada: OK")
}
