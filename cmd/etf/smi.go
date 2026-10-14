package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"time"
)

func moduloSMI(d dependenciasModulos) adaptadorHTTP.ModuloSMI {
	return adaptadorHTTP.ModuloSMI{Configuracion: adaptadorHTTP.ConfiguracionSMI{Servicio: catalogo.NuevoSMI(&catalogoPostgres.Repositorio{}, d.base, postgres.NuevoRegistroAuditoriaTransaccional(d.base), time.Now)}}
}
