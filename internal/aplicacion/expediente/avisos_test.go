package expediente

import (
	"context"
	"errors"
	catalogo "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
	"time"
)

type entornoAvisos struct {
	*entorno
	fecha  time.Time
	datos  []dominio.DatosAvisos
	cortes []string
	plazo  int
}

func (e *entornoAvisos) Ahora() time.Time { return e.fecha }
func (e *entornoAvisos) DatosAvisos(_ context.Context, _, _, corte string) ([]dominio.DatosAvisos, error) {
	e.cortes = append(e.cortes, corte)
	return e.datos, nil
}
func (e *entornoAvisos) Vigente(context.Context, string, time.Time) (catalogo.Version, error) {
	return catalogo.Version{Numero: 4}, nil
}
func (e *entornoAvisos) Datos(context.Context, catalogo.Version) (catalogo.Datos, error) {
	return catalogo.Datos{Plazos: []catalogo.Plazo{{Clave: "plazo_consentimiento", Valor: e.plazo, Unidad: "dias_naturales"}}}, nil
}
func prepararAvisos(t *testing.T) (*Servicio, *entornoAvisos, dominio.Actor) {
	s, e, a, _ := preparar(t)
	av := &entornoAvisos{entorno: e, fecha: time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC), plazo: 30, datos: []dominio.DatosAvisos{{ID: municipioPrueba, EquipoID: equipoPrueba, Estado: dominio.Aceptada, Aceptacion: "2026-12-01", AtendidoAnio: true}}}
	s.d.Avisos = av
	s.d.Catalogo = av
	s.d.Reloj = av
	return s, av, a
}
func TestAvisosRecuentoEneroYCorteHistorico(t *testing.T) {
	s, e, a := prepararAvisos(t)
	r, err := s.ContarAvisos(context.Background(), a, equipoPrueba)
	if err != nil || r.Expedientes != 1 || r.PorCodigo["consentimiento"] != 1 || r.PorCodigo["memoria_dato_faltante"] != 11 || len(e.cortes) != 2 || e.cortes[1] != "2026-12-31" {
		t.Fatal(r, e.cortes, err)
	}
	e.fecha = time.Date(2027, 2, 5, 0, 0, 0, 0, time.UTC)
	e.cortes = nil
	r, err = s.ContarAvisos(context.Background(), a, equipoPrueba)
	if err != nil || r.Total != 1 || len(e.cortes) != 1 {
		t.Fatal(r, e.cortes, err)
	}
	e.plazo = 90
	r, err = s.ContarAvisos(context.Background(), a, equipoPrueba)
	if err != nil || r.Total != 0 {
		t.Fatal(r, err)
	}
}
func TestAvisosPermisosAuditoriaYNoEntregaParcial(t *testing.T) {
	s, e, a := prepararAvisos(t)
	for _, perfil := range []string{"consulta", "administracion"} {
		a.Perfiles = []string{perfil}
		if _, err := s.ConsultarAvisos(context.Background(), a, municipioPrueba); !errors.Is(err, dominio.ErrAcceso) {
			t.Fatal(perfil, err)
		}
	}
	a.Perfiles = []string{"profesional_etf"}
	a.EquipoID = centroPrueba
	if r, err := s.ConsultarAvisos(context.Background(), a, municipioPrueba); !errors.Is(err, dominio.ErrAcceso) || r.ID != "" {
		t.Fatal(r, err)
	}
	if _, err := s.ContarAvisos(context.Background(), a, equipoPrueba); !errors.Is(err, dominio.ErrAcceso) {
		t.Fatal(err)
	}
	a.EquipoID = equipoPrueba
	e.falloAuditoria = true
	if r, err := s.ConsultarAvisos(context.Background(), a, municipioPrueba); !errors.Is(err, dominio.ErrNoDisponible) || r.ID != "" {
		t.Fatal(r, err)
	}
}
