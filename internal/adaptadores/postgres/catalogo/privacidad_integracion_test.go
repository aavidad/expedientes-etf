package catalogo_test

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

func TestIntegracionPrivacidadConsultaUmbralYRLS(t *testing.T) {
	b, ctx := base(t)
	r := &repo.Repositorio{}
	fecha := time.Date(2038, 1, 1, 0, 0, 0, 0, time.UTC)
	// Los cambios de esta prueba se revierten, incluso la publicación.
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		anterior, e := r.Ultima(ctx, "plazos")
		if e != nil {
			return e
		}
		fecha = anterior.Version.VigenteDesde.AddDate(15, 0, 0)
		_, e = r.Publicar(ctx, "plazos", d.Cambio{VersionEsperada: anterior.Version.Numero, VigenteDesde: fecha, Plazo: &d.Plazo{Clave: "umbral_privacidad_consulta", Valor: 4, Unidad: "casos"}})
		if e != nil {
			return e
		}
		sql, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		for _, perfil := range []string{"consulta", ""} {
			if _, e = sql.Exec(ctx, `SELECT set_config('app.perfiles',$1,true)`, perfil); e != nil {
				return e
			}
			n, e := r.UmbralConsulta(ctx, fecha)
			if e != nil {
				return e
			}
			esperado := 3
			if perfil == "consulta" {
				esperado = 4
			}
			if n != esperado {
				t.Fatalf("umbral RLS %s: %d", perfil, n)
			}
			if perfil == "consulta" {
				var otras int
				if e = sql.QueryRow(ctx, `SELECT count(*) FROM catalogo.parametro WHERE clave<>'umbral_privacidad_consulta'`).Scan(&otras); e != nil {
					return e
				}
				if otras != 0 {
					t.Fatal("otros parámetros expuestos", otras)
				}
			}
		}
		return d.ErrEdicion
	})
	if err != d.ErrEdicion {
		t.Fatal(err)
	}
	t.Log("PRIVACIDAD-RLS-OK Consulta usa umbral 4; sin contexto no ve reglas; otros parámetros cerrados")
}
