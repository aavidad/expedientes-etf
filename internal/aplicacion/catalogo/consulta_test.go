package catalogo

import (
	"context"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type repositorioPrueba struct {
	fecha    time.Time
	ctx      context.Context
	llamadas int
}

func (r *repositorioPrueba) Vigente(ctx context.Context, a string, f time.Time) (dominio.Version, error) {
	r.fecha = f
	r.ctx = ctx
	r.llamadas++
	return dominio.Version{ID: "v1", Ambito: a}, nil
}
func (r *repositorioPrueba) Versiones(context.Context, string) ([]dominio.Version, error) {
	r.llamadas++
	return []dominio.Version{{ID: "v1"}}, nil
}
func (r *repositorioPrueba) Datos(ctx context.Context, v dominio.Version) (dominio.Datos, error) {
	if ctx != r.ctx {
		return dominio.Datos{}, errors.New("prueba.contexto_distinto")
	}
	return dominio.Datos{Version: v}, nil
}

type txPrueba struct {
	identidad plataforma.Identidad
	fallo     error
}

func (tx *txPrueba) Ejecutar(ctx context.Context, i plataforma.Identidad, f func(context.Context) error) error {
	tx.identidad = i
	if err := f(ctx); err != nil {
		return err
	}
	return tx.fallo
}
func TestConsultaMemoriaYTransaccionUnica(t *testing.T) {
	repo := &repositorioPrueba{}
	tx := &txPrueba{}
	c := NuevaConsulta(repo, tx)
	actor := dominio.Actor{UsuarioID: "usuario", Perfiles: []string{"administracion"}}
	_, err := c.Memoria(context.Background(), actor, "listas", 2026)
	if err != nil || repo.fecha.Format(time.DateOnly) != "2026-12-31" || tx.identidad.UsuarioID != "usuario" {
		t.Fatal(err, repo.fecha, tx.identidad)
	}
	_, err = c.Datos(context.Background(), actor, "listas", repo.fecha)
	if err != nil {
		t.Fatal(err)
	}
}
func TestFalloConfirmacionNoEntregaDatos(t *testing.T) {
	repo := &repositorioPrueba{}
	tx := &txPrueba{fallo: errors.New("prueba.confirmacion")}
	c := NuevaConsulta(repo, tx)
	v, err := c.Vigente(context.Background(), dominio.Actor{}, "listas", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != tx.fallo || v.ID != "" {
		t.Fatal(v, err)
	}
	vs, err := c.Versiones(context.Background(), dominio.Actor{}, "listas")
	if err != tx.fallo || vs != nil {
		t.Fatal(vs, err)
	}
	d, err := c.Datos(context.Background(), dominio.Actor{}, "listas", repo.fecha)
	if err != tx.fallo || d.Version.ID != "" {
		t.Fatal(d, err)
	}
}
func TestEntradaInvalidaNoAbreTransaccion(t *testing.T) {
	repo := &repositorioPrueba{}
	c := NuevaConsulta(repo, &txPrueba{})
	if _, err := c.Memoria(context.Background(), dominio.Actor{}, "listas", 0); err != dominio.ErrVigencia {
		t.Fatal(err)
	}
	if _, err := c.Vigente(context.Background(), dominio.Actor{}, "desconocido", time.Now()); err != dominio.ErrAmbito {
		t.Fatal(err)
	}
	if repo.llamadas != 0 {
		t.Fatal("consultó base")
	}
}
