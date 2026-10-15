package catalogo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCierreVigenteExigeTransaccion(t *testing.T) {
	_, err := (&adaptador.Repositorio{}).CierreVigente(context.Background(), time.Now())
	if !errors.Is(err, postgres.ErrTransaccion) {
		t.Fatal(err)
	}
}

func TestIntegracionCierreD5VigenciaPermisosEInmutabilidad(t *testing.T) {
	b, ctx := base(t)
	r := &adaptador.Repositorio{}
	fecha := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, perfil := range []string{"coordinacion", "profesional_etf"} {
		t.Run(perfil, func(t *testing.T) {
			err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{perfil}}, func(ctx context.Context) error {
				catalogo, err := r.CierreVigente(ctx, fecha)
				if err != nil {
					return err
				}
				if catalogo.VersionID == "" || len(catalogo.Motivos) != 11 {
					t.Fatalf("motivos iniciales: %+v", catalogo)
				}
				for _, motivo := range catalogo.Motivos {
					if (motivo.Codigo == "DECLARACION_RIESGO" && motivo.SPAF != "DECLARACION_RIESGO") || (motivo.Codigo == "OTROS" && motivo.SPAF != "OTROS") {
						t.Fatalf("D5 equivalencia confirmada: %+v", motivo)
					}
					if motivo.AdmiteTexto != (motivo.Codigo == "OTROS") || motivo.SinConsentimiento != (motivo.Codigo == "SIN_CONSENTIMIENTO") {
						t.Fatalf("propiedades del motivo: %+v", motivo)
					}
					if motivo.Codigo == "SIN_CONSENTIMIENTO" && (motivo.Acta != "NO_ACEPTACION" || motivo.SPAF != "NO_COLABORACION") {
						t.Fatalf("equivalencia sin consentimiento: %+v", motivo)
					}
				}
				_, err = r.CierreVigente(ctx, fecha.AddDate(-1, 0, 0))
				if !errors.Is(err, dominio.ErrNoDisponible) {
					t.Fatalf("fecha anterior: %v", err)
				}
				version, err := r.Vigente(ctx, "asignacion_municipios", fecha)
				if err != nil {
					return err
				}
				geografia, err := r.Datos(ctx, version)
				if err != nil {
					return err
				}
				if len(geografia.Equipos) == 0 || len(geografia.Centros) == 0 || len(geografia.Municipios) == 0 {
					t.Fatal("catálogo geográfico de documentos vacío")
				}
				tx, err := postgres.DesdeContexto(ctx)
				if err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, "SAVEPOINT permiso_cierre"); err != nil {
					return err
				}
				_, err = tx.Exec(ctx, `UPDATE catalogo.motivo_cierre SET admite_texto=true WHERE codigo='OBJ_TOTAL'`)
				var fallo *pgconn.PgError
				if !errors.As(err, &fallo) || fallo.Code != "42501" {
					t.Fatalf("escritura de catálogo permitida a %s: %v", perfil, err)
				}
				if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT permiso_cierre; RELEASE SAVEPOINT permiso_cierre"); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, perfil := range []string{"", "consulta", "administracion", "sistema", "equipo"} {
		t.Run("denegado_"+perfil, func(t *testing.T) {
			identidad := plataforma.Identidad{}
			if perfil != "" {
				identidad = plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{perfil}}
			}
			err := b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
				tx, err := postgres.DesdeContexto(ctx)
				if err != nil {
					return err
				}
				var cantidad int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM catalogo.motivo_cierre`).Scan(&cantidad); err != nil {
					return err
				}
				if cantidad != 0 {
					t.Fatalf("RLS expuso %d motivos a %q", cantidad, perfil)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	tx := migrador(t, ctx)
	rechaza(t, ctx, tx, `UPDATE catalogo.motivo_cierre SET admite_texto=true WHERE codigo='OBJ_TOTAL'`, "P0001")
	rechaza(t, ctx, tx, `DELETE FROM catalogo.motivo_cierre WHERE codigo='OBJ_TOTAL'`, "P0001")
	rechaza(t, ctx, tx, `TRUNCATE catalogo.motivo_cierre`, "P0001")
	t.Log("CIERRE-D5-OK 11 motivos, vigencia, perfiles, equivalencia sin consentimiento e inmutabilidad")
}
