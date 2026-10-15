package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloUbicaciones(d dependenciasModulos) adaptadorHTTP.ModuloUbicaciones {
	return adaptadorHTTP.ModuloUbicaciones{Editor: catalogo.NuevaEdicion(&catalogoPostgres.Repositorio{}, paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{}), d.base, postgres.NuevoRegistroAuditoriaTransaccional(d.base), time.Now)}
}
