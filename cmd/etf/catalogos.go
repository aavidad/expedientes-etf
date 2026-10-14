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

func moduloCatalogos(d dependenciasModulos) adaptadorHTTP.ModuloCatalogos {
	cfg := adaptadorHTTP.ConfiguracionCatalogos{
		Consulta: catalogo.NuevaConsulta(&catalogoPostgres.Repositorio{}, d.base),
		Edicion:  catalogo.NuevaEdicion(&catalogoPostgres.Repositorio{}, paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{}), d.base, postgres.NuevoRegistroAuditoriaTransaccional(d.base), time.Now),
	}
	return adaptadorHTTP.ModuloCatalogos{Configuracion: cfg}
}
