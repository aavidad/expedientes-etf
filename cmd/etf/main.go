package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"github.com/aavidad/expedientes-etf/internal/plataforma"
	"github.com/aavidad/expedientes-etf/temas"
	"github.com/aavidad/expedientes-etf/web"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--lector-anexo-pdf" {
		if err := lectura.EjecutarPDF(os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		return
	}
	registro := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := ejecutar(registro); err != nil {
		// El error de red puede contener direcciones; no se vuelca su valor.
		registro.Error("servidor.error")
		os.Exit(1)
	}
}

func ejecutar(registro *slog.Logger) error {
	cfg := plataforma.LeerConfiguracion(os.Getenv)
	for _, variable := range cfg.Avisos {
		registro.Warn("configuracion.reserva", "variable", variable)
	}
	disco := ficheros.NuevosPaquetesConReserva(cfg.IdiomasRuta, cfg.TemasRuta, temas.Predeterminado())
	avisos := append([]string(nil), cfg.Avisos...)
	// pgx consulta también el entorno libpq. Se retira antes de crear el pool
	// o cualquier goroutine: sólo ETF_* configura esta aplicación.
	base := &postgres.Base{}
	err := limpiarEntornoPostgres()
	if err == nil {
		base, err = postgres.Nueva(context.Background(), os.Getenv)
	}
	defer base.Cerrar()
	calculoMemoria, _ := postgres.NuevaCalculoMemoria(context.Background(), os.Getenv)
	defer calculoMemoria.Cerrar()
	if err != nil {
		registro.Warn("postgres.configuracion_invalida")
	} else if err := base.Comprobar(context.Background()); err != nil {
		registro.Warn("postgres.no_disponible")
	}
	publicacion := postgres.NuevaPublicacionPaquetes(base)
	activos := ficheros.NuevosActivos(disco, publicacion)
	consulta := paquete.NuevaConsulta(activos, activos)
	antivirus, errAntivirus := clamav.Nuevo(clamav.Configuracion{Direccion: os.Getenv("ETF_CLAMAV_DIRECCION"), TiempoLimite: 45 * time.Second, MaximoBytes: 20 << 20})
	if errAntivirus != nil {
		registro.Warn("antivirus.configuracion_invalida")
	}
	instalador := paquete.NuevoInstalador(disco, antivirus, ficheros.NuevoAlmacenVersiones(cfg.IdiomasRuta, cfg.TemasRuta), publicacion, postgres.NuevoRegistroAuditoria(base), time.Now)
	if _, err := consulta.Textos(context.Background(), cfg.Idioma); err != nil {
		registro.Warn("idiomas.no_disponibles")
		avisos = append(avisos, "idiomas.no_disponibles")
		cfg.Idioma = "es"
	}
	if _, err := consulta.Tokens(context.Background(), cfg.Tema); err != nil {
		registro.Warn("tema.no_disponible")
		avisos = append(avisos, "tema.no_disponible")
		cfg.Tema = "expedientes-etf"
	}
	interfaz, err := fs.Sub(web.Compilacion, "dist")
	if err != nil {
		return err
	}
	sesiones, falloIdentidad := configurarIdentidad(base, os.Getenv)
	if falloIdentidad != nil {
		registro.Warn("identidad.configuracion_invalida")
	}
	opciones := adaptadorHTTP.Opciones{Sesion: sesiones, Modulos: componerModulos(dependenciasModulos{base: base, calculoMemoria: calculoMemoria, antivirus: antivirus, disco: disco, publicacion: publicacion, instalacion: instalador, cfg: cfg, registro: registro})}
	handler, err := adaptadorHTTP.NuevoValidado(consulta, interfaz, cfg.Idioma, cfg.Tema, avisos, registro, base.Comprobar, opciones)
	if err != nil {
		return err
	}
	servidor := &http.Server{
		Addr:              cfg.HTTPDireccion,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       90 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(slog.NewJSONHandler(os.Stderr, nil), slog.LevelError),
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	iniciarVencimientoSeguimientos(ctx, base, registro)
	go func() {
		limpiar := func() {
			limite, fin := context.WithTimeout(ctx, 20*time.Second)
			defer fin()
			if err := instalador.Limpiar(limite); err != nil && ctx.Err() == nil {
				registro.Warn("paquete.limpieza_pendiente")
			}
		}
		limpiar()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				limpiar()
			}
		}
	}()
	terminado := make(chan struct{})
	defer close(terminado)
	cierreTerminado := make(chan struct{})
	go func() {
		defer close(cierreTerminado)
		select {
		case <-ctx.Done():
			cierre, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelar()
			if err := servidor.Shutdown(cierre); err != nil {
				registro.Warn("servidor.cierre_incompleto")
			}
		case <-terminado:
		}
	}()
	registro.Info("servidor.inicio")
	err = servidor.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		<-cierreTerminado
		return nil
	}
	return err
}

func limpiarEntornoPostgres() error {
	for _, variable := range os.Environ() {
		nombre, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(nombre, "PG") {
			if err := os.Unsetenv(nombre); err != nil {
				return postgres.ErrConfiguracion
			}
		}
	}
	return nil
}
