package catalogo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestIntegracionSMIPermisosVersionYAuditoria(t *testing.T) {
	b, ctx := base(t)
	servicio := app.NuevoSMI(&repo.Repositorio{}, b, postgres.NuevoRegistroAuditoriaTransaccional(b), time.Now)
	coordinacion := entrada.Actor{UsuarioID: actorID, Perfiles: []string{"coordinacion"}}
	filas, err := servicio.Consultar(ctx, coordinacion)
	if err != nil || len(filas) != 15 || filas[0].Anio != 2026 || filas[14].Anio != 2040 {
		t.Fatalf("filas iniciales: cantidad=%d error=%v", len(filas), err)
	}
	cambio := dominio.CambioSMI{Anio: 2028, ImporteMensualCentimos: 123456, VersionEsperada: 0}
	guardada, err := servicio.Guardar(ctx, coordinacion, cambio)
	if err != nil || guardada.Version != 1 || guardada.ImporteMensual != "1234.56" || guardada.ImporteAnual != "17283.84" {
		t.Fatalf("importe SMI y anual: %+v error=%v", guardada, err)
	}
	_, err = servicio.Guardar(ctx, coordinacion, cambio)
	var conflicto *dominio.Conflicto
	if !errors.As(err, &conflicto) || conflicto.Actual != 1 {
		t.Fatalf("conflicto optimista: %v", err)
	}
	profesional := entrada.Actor{UsuarioID: actorID, Perfiles: []string{"profesional_etf"}}
	if _, err := servicio.Consultar(ctx, profesional); !errors.Is(err, dominio.ErrPermiso) {
		t.Fatalf("permiso profesional: %v", err)
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{"profesional_etf"}}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var filas int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM catalogo.smi`).Scan(&filas); err != nil {
			return err
		}
		if filas != 0 {
			t.Fatalf("RLS expuso %d filas a profesional_etf", filas)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("SMI-OK 15 años, anual ×14, permiso coordinación, historial, conflicto y RLS")
}
