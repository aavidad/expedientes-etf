package expediente_test

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	documentos "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"os"
	"slices"
	"strings"
	"testing"
)

type registroAuditoriaBandejaFallido struct{ registroFallido }

func (registroAuditoriaBandejaFallido) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, nil
}

func TestBandejaEquipoRLSBusquedaEnmascaradaHistorialYAuditoria(t *testing.T) {
	if os.Getenv("ETF_BANDEJA_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_BANDEJA_POSTGRES_INTEGRACION=1")
	}
	b, ctx, deps, a, p := prepararAceptacion(t)
	derivacion := subir(t, ctx, deps, a)
	p = conMenores(p, derivacion)
	aceptada, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, derivacion.ID, "bandeja-aceptar", p)
	if err != nil {
		t.Fatal(err)
	}
	lectura := aplicacion.LecturaEquipo{Avisos: &adaptador.Repositorio{}, Asignacion: deps.Catalogo, Repositorio: &adaptador.Repositorio{}, Documentos: documentos.Repositorio{}, Historial: postgres.HistorialExpediente{}, Registro: postgres.NuevoRegistroAuditoria(b), Transacciones: b, Reloj: reloj{}, Capacidades: []string{"documentos", "historial"}}
	profesional := d.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	lista, err := lectura.ListarEquipo(ctx, profesional, d.FiltroEquipo{Busqueda: aceptada.Numero, Limite: 50})
	if err != nil || len(lista.Filas) != 1 {
		t.Fatal(lista, err)
	}
	if strings.Contains(lista.Filas[0].Familia, "Ortega") || lista.Filas[0].Familia == "" {
		t.Fatal("listado sin enmascarar", lista)
	}
	if !slices.Contains(lista.Filas[0].Avisos, "equipo.aviso.memoria") {
		t.Fatal("avisos de F2.02 ausentes", lista.Filas[0].Avisos)
	}
	f, err := lectura.AbrirFichaEquipo(ctx, profesional, derivacion.ID)
	if err != nil || len(f.Hitos) != 1 || len(f.Historial) == 0 {
		t.Fatal(f, err)
	}
	if !strings.Contains(f.Familia, "Ortega") || f.Hitos[0].Tipo != "hito" {
		t.Fatal(f)
	}
	if _, err := lectura.AbrirFichaEquipo(ctx, profesional, "no-es-uuid"); !errors.Is(err, d.ErrPeticion) {
		t.Fatal(err)
	}
	for _, perfiles := range [][]string{{"profesional_etf"}, {"consulta"}, {"administracion"}, {"coordinacion", "profesional_etf"}, {}} {
		ajeno := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: perfiles, EquipoID: "22222222-2222-4222-8222-222222222222"}
		err = b.Ejecutar(ctx, ajeno, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var filas int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.expediente WHERE id=$1`, derivacion.ID).Scan(&filas); err != nil {
				return err
			}
			if filas != 0 && !slices.Contains(perfiles, "coordinacion") {
				t.Fatal("RLS permite equipo ajeno", perfiles)
			}
			return tx.QueryRow(ctx, `SELECT count(*) FROM auditoria.historial_expediente($1)`, derivacion.ID).Scan(&filas)
		})
		if err == nil {
			t.Fatal("historial permite perfil/equipo ajeno", perfiles)
		}
	}
	mixto := d.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{"coordinacion", "profesional_etf"}, EquipoID: "22222222-2222-4222-8222-222222222222"}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: mixto.UsuarioID, Perfiles: mixto.Perfiles, EquipoID: mixto.EquipoID}, func(ctx context.Context) error {
		datos, err := (&adaptador.Repositorio{}).DatosAvisosEquipo(ctx, []string{derivacion.ID}, mixto.EquipoID, d.FechaCivil(reloj{}.Ahora()))
		if len(datos) != 0 {
			t.Fatal("el puerto de avisos permite un expediente ajeno", datos)
		}
		return err
	})
	if !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal("el puerto de avisos no respeta el equipo del perfil doble", err)
	}
	if lista, err := lectura.ListarEquipo(ctx, mixto, d.FiltroEquipo{Busqueda: aceptada.Numero, Limite: 50}); err != nil || len(lista.Filas) != 0 {
		t.Fatal("bandeja mixta ajena", lista, err)
	}
	if _, err := lectura.AbrirFichaEquipo(ctx, a, derivacion.ID); err != nil {
		t.Fatal("coordinación sin equipo", err)
	}
	fallida := lectura
	fallida.Registro = registroAuditoriaBandejaFallido{}
	if salida, err := fallida.AbrirFichaEquipo(ctx, profesional, derivacion.ID); !errors.Is(err, d.ErrNoDisponible) || salida.ID != "" {
		t.Fatal("datos sin auditar", salida, err)
	}
	t.Log("Bandeja: iniciales, ficha, hitos, historial, RLS ajena y fallo de auditoría comprobados")
}
