package expediente_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// Esta prueba requiere la semilla exclusiva de medir_rendimiento.py.
func TestRendimientoMemoria3000(t *testing.T) {
	if os.Getenv("ETF_RENDIMIENTO") != "1" {
		t.Skip("ETF_RENDIMIENTO=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	a := d.ActorMemoria{UsuarioID: "40404040-4040-4040-8040-404040404040", Perfiles: []string{"coordinacion"}}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}
	fuente := app.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{},
		Reglas:  reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}},
		Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: &repo.Repositorio{}}
	fecha := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	duraciones := make([]time.Duration, 0, 20)
	for i := 0; i < 20; i++ {
		inicio := time.Now()
		var informe d.InformeMemoria
		err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			var err error
			informe, err = fuente.MemoriaEnTransaccion(ctx, a, d.FiltroMemoria{Anio: 2026}, fecha)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(informe.Fuentes) != 3000 || len(informe.Apartados) != 33 {
			t.Fatalf("volumen incorrecto: %d fuentes, %d apartados", len(informe.Fuentes), len(informe.Apartados))
		}
		duracion := time.Since(inicio)
		duraciones = append(duraciones, duracion)
		if duracion >= 2*time.Second {
			t.Fatal("objetivo de rendimiento incumplido")
		}
	}
	slices.Sort(duraciones)
	t.Logf("memoria provincial SQL y motor: p95=%.3f ms; mínimo=%.3f ms; máximo=%.3f ms; muestras=20; fuentes=3000; apartados=33", float64(duraciones[18].Microseconds())/1000, float64(duraciones[0].Microseconds())/1000, float64(duraciones[19].Microseconds())/1000)
	for _, perfiles := range [][]string{{"consulta"}, {"administracion"}, nil} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: perfiles}, func(ctx context.Context) error {
			fuentes, err := (&exp.LecturaEstadistica{}).ConsultarMemoria(ctx, "", fecha, fecha)
			if err != nil {
				return err
			}
			if len(fuentes) != 0 {
				t.Fatalf("filtración de fuentes a %v: %d", perfiles, len(fuentes))
			}
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			// Las tablas están pobladas: cero filas acredita aislamiento real.
			for _, tabla := range []string{"expediente", "derivacion_datos", "episodio", "hito", "familia_revision", "adulto_revision", "menor_revision", "intervencion_revision"} {
				var n int
				if err := tx.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					t.Fatalf("RLS: %v ve %s: %d", perfiles, tabla, n)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("RLS: Consulta, Administración y sesión sin perfil no reciben fuentes ni datos personales")
}
