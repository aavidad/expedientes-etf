package catalogo_test

import (
	"context"
	"errors"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

func TestIntegracionAvisosCatalogoVersionado(t *testing.T) {
	b, ctx := base(t)
	revertir := errors.New("prueba.revertir")
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		r := &repo.Repositorio{}
		anterior, err := r.Ultima(ctx, "plazos")
		if err != nil {
			return err
		}
		if len(anterior.DuracionesFase) != 10 {
			t.Fatal(anterior)
		}
		desde := anterior.Version.VigenteDesde.AddDate(0, 0, 1)
		nueva, err := r.Publicar(ctx, "plazos", dominio.Cambio{VersionEsperada: anterior.Version.Numero, VigenteDesde: desde, DuracionFase: &dominio.DuracionFase{Subprograma: "RIESGO_SOCIAL", Fase: "EVALUACION", DiasOrientativos: 120}})
		if err != nil {
			return err
		}
		datos, err := r.Datos(ctx, nueva)
		if err != nil {
			return err
		}
		if len(datos.Plazos) != len(anterior.Plazos) || len(datos.DuracionesFase) != 10 {
			t.Fatal(datos)
		}
		for _, f := range datos.DuracionesFase {
			if f.Subprograma == "RIESGO_SOCIAL" && f.Fase == "EVALUACION" && f.DiasOrientativos != 120 {
				t.Fatal(f)
			}
		}
		historica, err := r.Vigente(ctx, "plazos", desde.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		if historica.Numero != anterior.Version.Numero {
			t.Fatal(historica)
		}
		return revertir
	})
	if !errors.Is(err, revertir) {
		t.Fatal(err)
	}
	t.Log("AVISOS-CATALOGO-OK diez fases, edición por Administración, parámetros conservados y versión histórica")
}
