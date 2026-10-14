package main

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"log/slog"
	"time"
)

// El vencimiento se comprueba al arrancar y cada minuto. Sólo cambia seguimientos
// cuyo plazo ya terminó; varias instancias pueden ejecutarlo sin duplicar actos.
func iniciarVencimientoSeguimientos(ctx context.Context, base *postgres.Base, registro *slog.Logger) {
	s := app.Nuevo(app.Dependencias{Seguimientos: &repo.Repositorio{}, Transacciones: base, RegistroTransaccional: postgres.NuevoRegistroAuditoria(base), Reloj: relojIdentidad{}})
	go func() {
		vencer := func() {
			limite, fin := context.WithTimeout(ctx, 25*time.Second)
			defer fin()
			if err := s.VencerSeguimientos(limite); err != nil && ctx.Err() == nil {
				registro.Warn("expediente.seguimiento_vencimiento_pendiente")
			}
		}
		vencer()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				vencer()
			}
		}
	}()
}
