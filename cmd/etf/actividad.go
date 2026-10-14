package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividadPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/actividad"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloActividad(d dependenciasModulos) adaptadorHTTP.ModuloActividad {
	return adaptadorHTTP.ModuloActividad{Configuracion: adaptadorHTTP.ConfiguracionActividad{Servicio: actividad.Nuevo(&actividadPostgres.Repositorio{}, d.base, postgres.NuevoRegistroAuditoriaTransaccional(d.base), time.Now, actividad.Opciones{Equipos: actividadPostgres.Equipos{Catalogo: &catalogoPostgres.Repositorio{}, Ahora: time.Now}, Etiquetas: paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{})})}}
}
