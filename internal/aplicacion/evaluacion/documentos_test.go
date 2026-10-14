package evaluacion

import (
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	exp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/evaluacion"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"testing"
)

type familiaDocumento struct {
	expediente.Familias
	ficha exp.FichaFamilia
}

func (f familiaDocumento) ObtenerFamilia(context.Context, string) (exp.FichaFamilia, error) {
	return f.ficha, nil
}

type repoDocumento struct {
	salida.Repositorio
	registro d.Registro
	fallo    error
	lecturas int
}

func (r *repoDocumento) Consultar(context.Context, string) (d.Registro, error) {
	r.lecturas++
	return r.registro, r.fallo
}

type catalogoDocumento struct {
	salida.Catalogo
	escala d.Escala
}

func (c catalogoDocumento) Escala(context.Context, string) (d.Escala, error) { return c.escala, nil }

func TestLecturaDocumentoEscalasOpcionales(t *testing.T) {
	usuario := "11111111-1111-4111-8111-111111111111"
	id := "22222222-2222-4222-8222-222222222222"
	repo := &repoDocumento{}
	s := &Servicio{Repo: repo, Familias: familiaDocumento{ficha: exp.FichaFamilia{EquipoID: "33333333-3333-4333-8333-333333333333", Adultos: []exp.Adulto{{ID: "persona", Nombre: "Lucía", Apellidos: "Ortega Salas"}}}}}
	actor := d.Actor{UsuarioID: usuario, EquipoID: "33333333-3333-4333-8333-333333333333", Perfiles: []string{"profesional_etf"}}
	datos, err := s.ParaDocumentos(t.Context(), actor, id)
	if err != nil || len(datos.Registro.Escalas) != 0 || datos.Iniciales["persona"] != "L. O. S." {
		t.Fatal(datos, err)
	}
	// Sin escalas no se exige catálogo, baremo ni otra transacción.
	s.Catalogo = catalogoDocumento{escala: d.Escala{ID: "version", Version: 1, Instrumento: "SCORE15"}}
	repo.registro.Escalas = []d.Aplicacion{{VersionEscalaID: "version", Instrumento: "SCORE15"}}
	if datos, err = s.ParaDocumentos(t.Context(), actor, id); err != nil || len(datos.Registro.Escalas) != 1 {
		t.Fatal(datos, err)
	}
	for _, perfil := range []string{"consulta", "administracion", "coordinacion_ajena"} {
		actor.Perfiles = []string{perfil}
		antes := repo.lecturas
		if _, err := s.ParaDocumentos(t.Context(), actor, id); !errors.Is(err, d.ErrAcceso) || repo.lecturas != antes {
			t.Fatal("perfil", perfil, err)
		}
	}
	actor.Perfiles, actor.EquipoID = []string{"profesional_etf"}, "44444444-4444-4444-8444-444444444444"
	if _, err := s.ParaDocumentos(t.Context(), actor, id); !errors.Is(err, d.ErrAcceso) {
		t.Fatal("33333333-3333-4333-8333-333333333333", err)
	}
	actor.EquipoID = "33333333-3333-4333-8333-333333333333"
	repo.fallo = d.ErrNoDisponible
	if datos, err := s.ParaDocumentos(t.Context(), actor, id); err == nil || len(datos.Registro.Escalas) != 0 || len(datos.Iniciales) != 0 {
		t.Fatal("datos tras fallo", datos, err)
	}
}
