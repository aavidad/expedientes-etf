package evaluacion

import (
	"errors"
	"math"
	"testing"
)

func numero(v float64) *float64 { return &v }
func codigo(v string) *string   { return &v }

func escalaPrueba() Escala {
	return Escala{
		ID: "bb772e0e-e46a-4141-9c28-b6a44ef7fa1d", Instrumento: "SDQ_AUTO", Version: 1,
		Items: []Item{{Codigo: "sintomas_emocionales", Tipo: "numerico", Minimo: 0, Maximo: 10, Paso: 1}},
	}
}

func aplicacionPrueba(e Escala) Aplicacion {
	return Aplicacion{
		ID: "d40fd851-5d40-4878-b915-4a9dd9402d52", ExpedienteID: "31b36279-f2a4-41d4-a520-30e2b17d744f",
		EquipoID: "55bcb890-5516-44b0-8425-ed4604458281", VersionEscalaID: e.ID, Instrumento: e.Instrumento,
		Momento: "inicial", Sujeto: Sujeto{Tipo: "menor", MenorID: "f75e37c6-2b94-4cba-8688-1b787aa34cc4"}, Version: 1,
		Puntuaciones: []Puntuacion{{Item: "sintomas_emocionales", ValorNumerico: numero(4)}},
	}
}

func TestLimitesPorItem(t *testing.T) {
	for _, c := range []struct {
		nombre string
		valor  float64
		valido bool
	}{
		{"minimo", 0, true}, {"maximo", 10, true}, {"inferior", -1, false}, {"superior", 11, false},
		{"fraccion", 4.5, false}, {"nan", math.NaN(), false}, {"infinito", math.Inf(1), false},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			e := escalaPrueba()
			a := aplicacionPrueba(e)
			a.Puntuaciones[0].ValorNumerico = numero(c.valor)
			err := a.Validar(e)
			if c.valido && err != nil {
				t.Fatal(err)
			}
			if !c.valido && !errors.Is(err, ErrPuntuacion) {
				t.Fatal(err)
			}
		})
	}
}

func TestReglasVienenDeVersion(t *testing.T) {
	e := escalaPrueba()
	a := aplicacionPrueba(e)
	a.Puntuaciones[0].ValorNumerico = numero(4.5)
	if !errors.Is(a.Validar(e), ErrPuntuacion) {
		t.Fatal("fraccion admitida")
	}
	e.ID = "bd22d46f-e313-48e9-83b0-7feab618e64c"
	e.Version = 2
	e.Items[0].Paso = 0.5
	if !errors.Is(a.Validar(e), ErrVersionEscala) {
		t.Fatal("version distinta admitida")
	}
	a.VersionEscalaID = e.ID
	if err := a.Validar(e); err != nil {
		t.Fatal(err)
	}
}

func TestValoresDesconocidosDuplicadosYTipos(t *testing.T) {
	e := escalaPrueba()
	for _, c := range []struct {
		nombre       string
		puntuaciones []Puntuacion
	}{
		{"desconocido", []Puntuacion{{Item: "inexistente", ValorNumerico: numero(1)}}},
		{"duplicado", []Puntuacion{{Item: "sintomas_emocionales", ValorNumerico: numero(1)}, {Item: "sintomas_emocionales", ValorNumerico: numero(2)}}},
		{"vacio", []Puntuacion{{Item: "sintomas_emocionales"}}},
		{"dos_valores", []Puntuacion{{Item: "sintomas_emocionales", ValorNumerico: numero(1), ValorCodigo: codigo("alto")}}},
		{"tipo_erroneo", []Puntuacion{{Item: "sintomas_emocionales", ValorCodigo: codigo("alto")}}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			a := aplicacionPrueba(e)
			a.Puntuaciones = c.puntuaciones
			if !errors.Is(a.Validar(e), ErrPuntuacion) {
				t.Fatal("puntuacion admitida")
			}
		})
	}
	e.Items = []Item{{Codigo: "global", Tipo: "codigo", Valores: []string{"riesgo_leve", "riesgo_grave"}}}
	a := aplicacionPrueba(e)
	a.Puntuaciones = []Puntuacion{{Item: "global", ValorCodigo: codigo("riesgo_leve")}}
	if err := a.Validar(e); err != nil {
		t.Fatal(err)
	}
	a.Puntuaciones[0].ValorCodigo = codigo("no_publicado")
	if !errors.Is(a.Validar(e), ErrPuntuacion) {
		t.Fatal("codigo admitido")
	}
}

func TestSujetoCompatible(t *testing.T) {
	id := "f75e37c6-2b94-4cba-8688-1b787aa34cc4"
	for _, s := range []Sujeto{
		{Tipo: "menor"}, {Tipo: "familia", AdultoID: id}, {Tipo: "menor", MenorID: id, AdultoID: id},
		{Tipo: "figura_parental", MenorID: id}, {Tipo: "profesional"}, {Tipo: "profesional", ProfesionalID: id, MenorID: id},
	} {
		if !errors.Is(s.Validar(), ErrSujeto) {
			t.Fatal(s)
		}
	}
	for _, s := range []Sujeto{{Tipo: "familia"}, {Tipo: "menor", MenorID: id}, {Tipo: "figura_parental", AdultoID: id}, {Tipo: "profesional", ProfesionalID: id}} {
		if err := s.Validar(); err != nil {
			t.Fatal(s, err)
		}
	}
}

func TestPermisosCoordinacionYEquipo(t *testing.T) {
	a := Actor{UsuarioID: "3c99ec35-5e03-4e01-950a-bfb9cce55033", Perfiles: []string{"coordinacion"}}
	if err := a.Autorizar(false); err != nil {
		t.Fatal(err)
	}
	if err := a.Autorizar(true); err != nil {
		t.Fatal(err)
	}
	a.Perfiles = []string{"profesional_etf"}
	if !errors.Is(a.Autorizar(false), ErrAcceso) {
		t.Fatal("lectura sin equipo admitida")
	}
	a.EquipoID = "55bcb890-5516-44b0-8425-ed4604458281"
	if err := a.Autorizar(true); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogoInvalidoNoPermitePuntuar(t *testing.T) {
	for _, i := range []Item{
		{Codigo: "sintomas_emocionales", Tipo: "numerico", Maximo: 10},
		{Codigo: "sintomas_emocionales", Tipo: "numerico", Minimo: 11, Maximo: 10, Paso: 1},
		{Codigo: "sintomas_emocionales", Tipo: "numerico", Maximo: math.Inf(1), Paso: 1},
		{Codigo: "sintomas_emocionales", Tipo: "numerico", Minimo: -1e308, Maximo: 1e308, Paso: 1e308},
		{Codigo: "sintomas_emocionales", Tipo: "codigo", Valores: []string{"a", "a"}},
		{Codigo: "sintomas_emocionales", Tipo: "codigo", Valores: []string{""}},
	} {
		e := escalaPrueba()
		e.Items = []Item{i}
		if !errors.Is(aplicacionPrueba(e).Validar(e), ErrVersionEscala) {
			t.Fatal(i)
		}
	}
}
