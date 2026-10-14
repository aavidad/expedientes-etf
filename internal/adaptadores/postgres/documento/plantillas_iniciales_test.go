package documento_test

import (
	"context"
	"os"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/docx"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestPlantillasInicialesCorregidasPostgres(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx := t.Context()
	base, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Cerrar()
	almacen := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: idiomaDistribucion{}}
	r := repo.Repositorio{}
	err = base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		ps, err := r.Listar(ctx)
		if err != nil {
			t.Fatalf("listado: %v", err)
		}
		activas, memoria := 0, 0
		for _, p := range ps {
			if !p.Activa {
				continue
			}
			activas++
			// La Memoria nació después de la revisión F3.03: su primera versión ya es la vigente.
			nueva := p.Plantilla.Tipo == "memoria"
			if nueva {
				memoria++
			}
			if d.EsFinalizacion(p.Plantilla.Tipo) && p.Plantilla.Version != "1.0.2" {
				t.Fatal("finalización sin escalas opcionales", p.Plantilla.Version)
			}
			if !nueva && !d.EsFinalizacion(p.Plantilla.Tipo) && p.Plantilla.Version != "1.0.1" && p.Plantilla.Version != "1.1.1" {
				t.Fatalf("predeterminada histórica: %+v", p.Plantilla)
			}
			idioma, err := almacen.Resolver(ctx, p.Plantilla.Idioma)
			if err != nil {
				t.Fatalf("fichero %s/%s: %v", p.Plantilla.Idioma, p.Plantilla.Tipo, err)
			}
			b, err := almacen.Leer(ctx, idioma, p.Plantilla)
			if err != nil {
				t.Fatalf("fichero %s/%s: %v", p.Plantilla.Idioma, p.Plantilla.Tipo, err)
			}
			if d.Huella(b) != p.Plantilla.Huella {
				t.Fatal("huella inicial distinta del fichero")
			}
			if err := docx.ValidarCambio(b, b); err != nil {
				return err
			}
		}
		if activas-memoria != 26 {
			t.Fatalf("referencias activas=%d", activas)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("26 referencias activas corregidas; huellas y seguridad comprobadas")
}
