package expediente

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func cargaCivil() CargaHistorica {
	a := Aceptacion{FechaActa: "2026-02-01", MunicipioID: "11111111-1111-4111-8111-111111111111", CentroID: "22222222-2222-4222-8222-222222222222", EquipoID: "33333333-3333-4333-8333-333333333333", Subprograma: "RIESGO_SOCIAL", ProfesionalPuesto: "TS", Menores: []SexoMenor{{Orden: 1, Sexo: "M"}}, Descripcion: "La familia necesita apoyo para organizar los cuidados."}
	return CargaHistorica{EquipoID: a.EquipoID, Casos: []CasoHistorico{{NumeroHistorico: NumeroHistorico{Numero: "08-31-26"}, FechaDerivacion: "2026-01-01", Estado: Cerrada, Datos: DatosDerivacion{Extraccion: Extraccion{Tipo: "anexo3", Campos: []Campo{{Codigo: "menores.persona.nombre", Valor: "Carmen"}}}}, Aceptacion: &a, Episodios: []EpisodioHistorico{{Apertura: a.FechaActa, Cierre: "2026-03-01", MotivoCierre: "OBJ_TOTAL"}, {Apertura: "2026-04-01", Cierre: "2026-05-01", MotivoCierre: "OBJ_TOTAL", MotivoReapertura: "La familia vuelve a necesitar apoyo."}}}}}
}
func TestCargaHistoricaFechasYAlcance2026(t *testing.T) {
	ahora := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		nombre  string
		cambiar func(*CargaHistorica)
	}{
		{"aceptacion anterior a derivacion", func(p *CargaHistorica) { p.Casos[0].FechaDerivacion = "2026-03-01" }},
		{"cierre anterior a apertura", func(p *CargaHistorica) { p.Casos[0].Episodios[0].Cierre = "2026-01-31" }},
		{"reapertura anterior a cierre", func(p *CargaHistorica) { p.Casos[0].Episodios[1].Apertura = "2026-02-28" }},
		{"fecha futura", func(p *CargaHistorica) { p.Casos[0].Episodios[1].Cierre = "2026-10-15" }},
		{"fecha inexistente", func(p *CargaHistorica) { p.Casos[0].Episodios[0].Cierre = "2026-02-30" }},
		{"estado incompatible", func(p *CargaHistorica) { p.Casos[0].Estado = EnIntervencion }},
		{"equipo incompatible", func(p *CargaHistorica) { p.EquipoID = "44444444-4444-4444-8444-444444444444" }},
		{"numero en no aceptado", func(p *CargaHistorica) {
			p.Casos[0].Estado = NoAceptada
			p.Casos[0].Aceptacion = nil
			p.Casos[0].Episodios = nil
			p.Casos[0].NoAceptacion = &NoAceptacion{Fecha: "2026-02-01", Motivo: "Se mantiene el apoyo en Servicios Sociales.", Subprograma: "RIESGO_SOCIAL"}
		}},
		{"fuera de año", func(p *CargaHistorica) {
			p.Casos[0].Aceptacion.FechaActa = "2025-02-01"
			p.Casos[0].Episodios = []EpisodioHistorico{{Apertura: "2025-02-01", Cierre: "2025-03-01", MotivoCierre: "OBJ_TOTAL"}}
			p.Casos[0].FechaDerivacion = "2025-01-01"
		}},
		{"vigencia desordenada", func(p *CargaHistorica) {
			p.Casos[0].Familia = []CambioFamilia{{Desde: "2026-04-01"}, {Desde: "2026-03-01"}}
		}},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			b, _ := json.Marshal(cargaCivil())
			var p CargaHistorica
			_ = json.Unmarshal(b, &p)
			tt.cambiar(&p)
			if err := p.Validar(ahora); err == nil {
				t.Fatal("fecha o estado incoherente admitido")
			}
		})
	}
	if err := cargaCivil().Validar(ahora); err != nil {
		t.Fatal(err)
	}
	p := cargaCivil()
	p.Casos = append(p.Casos, p.Casos[0])
	if err := p.Validar(ahora); !errors.Is(err, ErrNumeroDuplicado) {
		t.Fatal(err)
	}
	p = cargaCivil()
	p.Casos[0].Numero = ""
	if err := p.Validar(ahora); err != nil {
		t.Fatal("número desconocido", err)
	}
}
func TestNumeroHistoricoFormatoEquipoYAnio(t *testing.T) {
	for _, n := range []string{"08-00-26", "8-01-26", "08-001-26", "08-01-2026", "07-01-26", "08-01-25"} {
		if _, err := (NumeroHistorico{Numero: n}).Validar("08", 2026); !errors.Is(err, ErrNumeroHistorico) {
			t.Fatal(n, err)
		}
	}
	if n, err := (NumeroHistorico{Numero: "08-42-26"}).Validar("08", 2026); err != nil || n != 42 {
		t.Fatal(n, err)
	}
}

func TestCargaAbiertosDeCualquierAnioAnteriorYCerrados2026(t *testing.T) {
	ahora := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	for _, anio := range []int{2001, 2010, 2022, 2023, 2025, 2026} {
		fecha := time.Date(anio, 2, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		p := cargaCivil()
		c := &p.Casos[0]
		c.FechaDerivacion = fecha
		c.Aceptacion.FechaActa = fecha
		c.Episodios = []EpisodioHistorico{{Apertura: fecha}}
		c.Estado = EnIntervencion
		if err := p.Validar(ahora); err != nil {
			t.Fatal(anio, err)
		}
		if _, err := (NumeroHistorico{Numero: "08-31-" + fecha[2:4]}).Validar("08", anio); err != nil {
			t.Fatal(anio, err)
		}
		c.Estado = Cerrada
		c.Episodios[0].Cierre = "2026-06-20"
		c.Episodios[0].MotivoCierre = "OBJ_TOTAL"
		if err := p.Validar(ahora); err != nil {
			t.Fatal("cerrado2026", anio, err)
		}
		if anio < 2026 {
			c.Episodios[0].Cierre = fecha
			if err := p.Validar(ahora); err == nil {
				t.Fatal("cierre ajeno a2026 admitido", anio)
			}
		}
	}
}

func TestCargaHistoricaIdentidadesOriginalesYVigenciasPorColeccion(t *testing.T) {
	ahora := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	const id = "55555555-5555-4555-8555-555555555555"
	p := cargaCivil()
	p.Casos[0].Familia = []CambioFamilia{{Desde: "2026-02-01", Familia: &Familia{}}, {Desde: "2026-02-01", Adultos: &[]Adulto{}}}
	if err := p.Validar(ahora); err != nil {
		t.Fatal("vigencias de colecciones independientes", err)
	}
	p.Casos[0].Familia[1].Adultos = &[]Adulto{{ID: id, Orden: 1}, {ID: id, Orden: 2}}
	if err := p.Validar(ahora); !errors.Is(err, ErrFamilia) {
		t.Fatal("adultos duplicados", err)
	}
	p = cargaCivil()
	p.Casos[0].Menores = []CambioMenores{{Desde: "2026-02-01", Menores: &[]Menor{{ID: id, Orden: 1}, {ID: id, Orden: 2}}}}
	if err := p.Validar(ahora); !errors.Is(err, ErrMenores) {
		t.Fatal("menores duplicados", err)
	}
	p.Casos[0].Menores[0].Menores = &[]Menor{{ID: "identidad-invalida", Orden: 1}}
	if err := p.Validar(ahora); !errors.Is(err, ErrMenores) {
		t.Fatal("UUID inválido", err)
	}
}
