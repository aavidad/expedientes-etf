package documento_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestRESP1ReferenciasNuevasYOriginalesInmutables(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	b, err := postgres.Nueva(t.Context(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	traducciones := traduccionesEN(t)
	err = b.Ejecutar(t.Context(), plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		for _, idioma := range []string{"es", "en"} {
			base := filepath.Join("../../../../idiomas", idioma, "plantillas")
			contenido, err := os.ReadFile(filepath.Join(base, "respuestas-2026-10-14.json"))
			if err != nil {
				return err
			}
			var cambios map[string]struct{ Previa, Nueva string }
			if err = json.Unmarshal(contenido, &cambios); err != nil {
				return err
			}
			for tipo, c := range cambios {
				for _, version := range []string{c.Previa, c.Nueva} {
					archivo, err := os.ReadFile(filepath.Join(base, tipo, "v"+version+".docx"))
					if err != nil {
						return err
					}
					var huella string
					if err = tx.QueryRow(ctx, `SELECT encode(huella_sha256,'hex') FROM documento.plantilla WHERE tipo_documento=$1 AND idioma=$2 AND version=$3`, tipo, idioma, version).Scan(&huella); err != nil {
						return err
					}
					if huella != d.Huella(archivo) {
						t.Fatalf("versión %s/%s/%s alterada", tipo, idioma, version)
					}
				}
				var activa string
				var administrativa bool
				if err = tx.QueryRow(ctx, `SELECT p.version,p.subida_por IS NOT NULL FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id WHERE r.tipo_documento=$1 AND r.idioma=$2`, tipo, idioma).Scan(&activa, &administrativa); err != nil {
					return err
				}
				esperada := c.Nueva
				if traduccion, ok := traducciones[tipo]; idioma == "en" && ok {
					esperada = traduccion.Nueva
				}
				if !administrativa && activa != esperada {
					t.Fatalf("referencia inicial %s/%s: %s", tipo, idioma, activa)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("RESP1-PLANTILLAS-OK: diez referencias nuevas, huellas verificadas y originales conservados")
}
