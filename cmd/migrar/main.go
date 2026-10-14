// migrar es una orden de despliegue; nunca se importa desde la aplicación.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aavidad/expedientes-etf/internal/plataforma/migracion"
)

func main() {
	registro := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ruta := os.Getenv("ETF_MIGRACIONES_RUTA")
	if ruta == "" {
		ruta = "migraciones"
	}
	plan, err := migracion.LeerPlan(os.DirFS(ruta))
	if err == nil {
		ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancelar()
		limite, cancelarLimite := context.WithTimeout(ctx, 10*time.Minute)
		defer cancelarLimite()
		err = migracion.Ejecutar(limite, migracion.Configuracion{Host: os.Getenv("ETF_BD_HOST"), Puerto: os.Getenv("ETF_BD_PUERTO"), Base: os.Getenv("ETF_BD_NOMBRE"), Contrasena: os.Getenv("ETF_MIGRADOR_CONTRASENA")}, plan, os.Stdout)
	}
	if err != nil {
		// LeerPlan y Ejecutar devuelven claves; los errores de disco no se imprimen.
		clave := "migracion.error"
		switch err.Error() {
		case "migracion.huella", "migracion.historial", "migracion.rol", "migracion.dependencia", "migracion.retroceso", "migracion.ciclo", "migracion.dependencia_ausente", "migracion.estado_previo", "migracion.configuracion":
			clave = err.Error()
		}
		registro.Error(clave)
		os.Exit(1)
	}
	registro.Info("migracion.fin")
}
