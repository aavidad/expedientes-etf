package expediente_test

import (
	"context"
	"encoding/json"
	"errors"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"testing"
	"time"
)

type lectorSinMunicipio struct{ original salida.LectorAnexo }

func (l lectorSinMunicipio) Leer(ctx context.Context, formato string, b []byte) (dominio.Extraccion, error) {
	e, err := l.original.Leer(ctx, formato, b)
	for i := range e.Campos {
		if e.Campos[i].Codigo == "familia.direccion" || strings.HasPrefix(e.Campos[i].Codigo, "valoracion.") {
			e.Campos[i].Valor = ""
		}
	}
	return e, err
}

func TestIntegracionBandejaEnmascaradoFiltrosRLSYAuditoria(t *testing.T) {
	b, ctx, deps, a, p := prepararDecision(t)
	deps.Bandeja = &adaptador.Repositorio{}
	servicio := aplicacion.Nuevo(deps)
	sinMunicipio := deps
	sinMunicipio.Lector = lectorSinMunicipio{original: deps.Lector}
	sinPropuesta := subir(t, ctx, sinMunicipio, a)
	if sinPropuesta.Datos.PropuestaMunicipio.Candidatos != nil {
		t.Fatal("se esperaba propuesta sin candidatos")
	}
	recibida := subir(t, ctx, deps, a)
	espera := subir(t, ctx, deps, a)
	rechazada := subir(t, ctx, deps, a)
	aceptada := subir(t, ctx, deps, a)
	hoy := dominio.FechaCivil(deps.Reloj.Ahora())
	if _, err := servicio.DejarEnEspera(ctx, a, espera.ID, "bandeja-espera", dominio.Espera{Version: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.NoAceptar(ctx, a, rechazada.ID, "bandeja-rechazo", dominio.NoAceptacion{Version: 1, Fecha: hoy, Motivo: "La familia reside fuera del ámbito del Programa.", Subprograma: "RIESGO_SOCIAL"}); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Aceptar(ctx, a, aceptada.ID, "bandeja-aceptar", conMenores(p, aceptada)); err != nil {
		t.Fatal(err)
	}
	filtro := dominio.FiltroBandeja{Busqueda: "Ortega", Limite: 100}
	r, err := servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, f := range r.Filas {
		ids[f.ID] = true
		if f.Subprograma != "RIESGO_SOCIAL" {
			t.Fatalf("subprograma del Anexo 3: %q", f.Subprograma)
		}
		if f.Iniciales != "O. S." {
			t.Fatalf("iniciales=%q", f.Iniciales)
		}
	}
	for _, id := range []string{sinPropuesta.ID, recibida.ID, espera.ID, rechazada.ID, aceptada.ID} {
		if !ids[id] {
			t.Fatalf("falta estado: %s", id)
		}
	}
	contenido, _ := json.Marshal(r.Filas)
	for _, prohibido := range []string{"Ortega", "Lucía", "telefono", "direccion", "descripcion"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("dato expuesto: %s", prohibido)
		}
	}
	filtro.Estado = "aceptada"
	filtro.EquipoID = p.EquipoID
	filtro.MunicipioID = p.MunicipioID
	r, err = servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil || r.Total < 1 {
		t.Fatal(r, err)
	}
	filtro.Desplazamiento = 100000
	r, err = servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil || len(r.Filas) != 0 || r.Total < 1 {
		t.Fatal("total de página vacía", r, err)
	}
	filtro = dominio.FiltroBandeja{Busqueda: "Ortega", Limite: 1}
	r, err = servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil || len(r.Filas) != 1 || r.Total < 4 {
		t.Fatal("paginación", r, err)
	}
	filtro.Busqueda = "%"
	r, err = servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil || r.Total != 0 {
		t.Fatal("comodín de búsqueda", r, err)
	}
	filtro = dominio.FiltroBandeja{Desde: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), Limite: 20}
	r, err = servicio.ConsultarBandeja(ctx, a, filtro)
	if err != nil || r.Total != 0 {
		t.Fatal("fecha", r, err)
	}
	for _, perfil := range []string{"administracion", "consulta", "profesional_etf", ""} {
		otro := a
		otro.Perfiles = []string{perfil}
		if perfil == "" {
			otro.Perfiles = nil
		}
		if _, err := servicio.ConsultarBandeja(ctx, otro, dominio.FiltroBandeja{Limite: 20}); !errors.Is(err, dominio.ErrAcceso) {
			t.Fatalf("perfil %q: %v", perfil, err)
		}
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: otro.Perfiles}, func(ctx context.Context) error {
			r, err := deps.Bandeja.ConsultarBandeja(ctx, dominio.FiltroBandeja{Limite: 20}, "")
			if err == nil && r.Total != 0 {
				t.Fatalf("RLS perfil %q: %d", perfil, r.Total)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	deps.RegistroTransaccional = registroFallido{}
	if r, err := aplicacion.Nuevo(deps).ConsultarBandeja(ctx, a, dominio.FiltroBandeja{Limite: 20}); !errors.Is(err, dominio.ErrNoDisponible) || r.Filas != nil {
		t.Fatal("datos entregados sin auditoría", r, err)
	}
}
