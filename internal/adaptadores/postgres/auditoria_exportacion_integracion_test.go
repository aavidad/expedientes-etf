package postgres

import (
	"bytes"
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"testing"
	"time"
)

func TestIntegracionExportacionFlujoAuditoria(t *testing.T) {
	b, ctx := baseIntegracion(t)
	repo := LecturaExportacionAuditoria{}
	registro := NuevoRegistroAuditoria(b)
	usuario := "aaabbbcc-1234-4234-8234-123456789abc"
	// Fechas exclusivas del ensayo; fuera del resto de fixtures.
	f := auditoria.FiltroExportacion{Desde: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)}
	var cadena []auditoria.Evento
	for i := 0; i < 260; i++ {
		e, err := registro.Confirmar(ctx, auditoria.Datos{OcurridoEn: f.Desde.Add(time.Duration(i) * time.Second), UsuarioID: usuario, Perfiles: []string{"administracion"}, Accion: "auditoria.consultar", Entidad: "auditoria", Resultado: "ok"})
		if err != nil {
			t.Fatal(err)
		}
		cadena = append(cadena, e)
	}
	for _, perfil := range []string{"administracion", "coordinacion", "profesional_etf", "consulta", ""} {
		t.Run(perfil, func(t *testing.T) {
			actor := plataforma.Identidad{}
			if perfil != "" {
				actor.UsuarioID = usuario
				actor.Perfiles = []string{perfil}
			}
			err := b.Ejecutar(ctx, actor, func(ctx context.Context) error {
				tramo, err := repo.Delimitar(ctx, f)
				if err != nil {
					return err
				}
				if tramo.Final != cadena[len(cadena)-1].Punto() {
					t.Fatal("sello final incorrecto")
				}
				despues := tramo.Anterior.ID
				total := 0
				for despues < tramo.Final.ID {
					pagina, err := repo.Pagina(ctx, tramo, despues)
					if err != nil {
						return err
					}
					if len(pagina) == 0 || len(pagina) > 256 {
						t.Fatal("página sin límite")
					}
					for _, e := range pagina {
						total++
						despues = e.ID
					}
				}
				if total != 260 {
					t.Fatalf("eventos=%d", total)
				}
				return nil
			})
			if perfil == "administracion" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != "auditoria.permiso_denegado" {
				t.Fatalf("permiso=%s err=%v", perfil, err)
			}
			// La página también comprueba perfil si se intenta saltar la delimitación.
			err = b.Ejecutar(ctx, actor, func(ctx context.Context) error {
				_, err := repo.Pagina(ctx, auditoria.TramoExportacion{Final: cadena[len(cadena)-1].Punto()}, 0)
				return err
			})
			if perfil != "administracion" && (err == nil || err.Error() != "auditoria.permiso_denegado") {
				t.Fatalf("página expuesta: %v", err)
			}
		})
	}
	servicio := app.NuevoExportador(repo, b, ficheros.NuevasExportacionesAuditoria(), registro, time.Now)
	descarga, err := servicio.Exportar(ctx, auditoria.ActorExportacion{UsuarioID: usuario, Perfiles: []string{"administracion"}}, f, "csv", "auditoria.instrucciones_verificacion")
	if err != nil {
		t.Fatalf("exportación completa: %v", err)
	}
	contenido, err := io.ReadAll(descarga.Contenido)
	_ = descarga.Contenido.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ficheros.VerificarPaqueteAuditoria(bytes.NewReader(contenido), int64(len(contenido))); err != nil {
		t.Fatal(err)
	}
	var fuerza, rls bool
	if err := b.pool.QueryRow(ctx, `SELECT relrowsecurity,relforcerowsecurity FROM pg_class WHERE oid='auditoria.evento'::regclass`).Scan(&rls, &fuerza); err != nil || !rls || !fuerza {
		t.Fatalf("RLS=%v FORCE=%v err=%v", rls, fuerza, err)
	}
	if _, err := b.pool.Exec(ctx, `SELECT id FROM auditoria.evento LIMIT 1`); err == nil {
		t.Fatal("lectura directa del registro permitida")
	}
	if _, err := b.pool.Exec(ctx, `SELECT auditoria.pagina_exportacion(0,1)`); err == nil {
		t.Fatal("contexto de Administración filtrado entre transacciones")
	}
}
