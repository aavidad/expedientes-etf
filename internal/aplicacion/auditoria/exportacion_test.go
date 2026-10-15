package auditoria_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	app "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	s "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type txExportacion struct{ llamadas int }

func (t *txExportacion) Ejecutar(ctx context.Context, a plataforma.Identidad, op func(context.Context) error) error {
	t.llamadas++
	return op(ctx)
}

type lecturaExportacion struct{ eventos []d.Evento }

func (l lecturaExportacion) Delimitar(context.Context, d.FiltroExportacion) (d.TramoExportacion, error) {
	return d.TramoExportacion{Anterior: d.Inicio(), Final: l.eventos[len(l.eventos)-1].Punto()}, nil
}
func (l lecturaExportacion) Pagina(_ context.Context, _ d.TramoExportacion, id uint64) ([]d.Evento, error) {
	if id == 0 {
		return l.eventos, nil
	}
	return nil, nil
}

type archivoExportacion struct {
	io.Reader
	cerrado bool
}

func (a *archivoExportacion) Close() error  { a.cerrado = true; return nil }
func (a *archivoExportacion) Tamano() int64 { return 1 }
func (a *archivoExportacion) Filas() int    { return 1 }

type paquetesExportacion struct{ archivo *archivoExportacion }

func (p paquetesExportacion) Preparar(_ context.Context, _ d.TramoExportacion, _ d.FiltroExportacion, _, _ string, recorrer func(func(d.Evento) error) error) (s.ArchivoExportacion, error) {
	err := recorrer(func(d.Evento) error { return nil })
	if err != nil {
		return nil, err
	}
	return p.archivo, nil
}
func TestExportadorDeniegaAntesDeLeerYExigeAuditoria(t *testing.T) {
	e, err := d.Encadenar(1, datos(), d.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"consulta", "coordinacion", "profesional_etf", "administracion"} {
		t.Run(perfil, func(t *testing.T) {
			tx := &txExportacion{}
			a := &archivoExportacion{Reader: strings.NewReader("x")}
			registro := &registroMemoria{err: errors.New("fallo")}
			servicio := app.NuevoExportador(lecturaExportacion{[]d.Evento{e}}, tx, paquetesExportacion{a}, registro, time.Now)
			f := d.FiltroExportacion{Desde: e.OcurridoEn.Add(-time.Hour), Hasta: e.OcurridoEn.Add(time.Hour)}
			resultado, err := servicio.Exportar(context.Background(), d.ActorExportacion{UsuarioID: "12345678-1234-4234-8234-123456789abc", Perfiles: []string{perfil}}, f, "json", "")
			if err == nil || resultado.Contenido != nil {
				t.Fatal("entrega sin auditoría")
			}
			if perfil != "administracion" {
				if tx.llamadas != 0 {
					t.Fatal("lee sin permiso")
				}
			} else if !a.cerrado {
				t.Fatal("no limpia archivo tras fallo de auditoría")
			}
		})
	}
}
func TestExportadorRechazaCadenaRota(t *testing.T) {
	e, _ := d.Encadenar(1, datos(), d.Inicio())
	e.Accion = "auditoria.alterada"
	a := &archivoExportacion{Reader: strings.NewReader("x")}
	registro := &registroMemoria{}
	s := app.NuevoExportador(lecturaExportacion{[]d.Evento{e}}, &txExportacion{}, paquetesExportacion{a}, registro, time.Now)
	_, err := s.Exportar(context.Background(), d.ActorExportacion{UsuarioID: "12345678-1234-4234-8234-123456789abc", Perfiles: []string{"administracion"}}, d.FiltroExportacion{Desde: e.OcurridoEn.Add(-time.Hour), Hasta: e.OcurridoEn.Add(time.Hour)}, "csv", "")
	if len(registro.eventos) != 1 || registro.eventos[0].Resultado != "error" {
		t.Fatal("fallo sin auditoría")
	}
	if err == nil || err.Error() != "auditoria.cadena_alterada" {
		t.Fatal(err)
	}
}
