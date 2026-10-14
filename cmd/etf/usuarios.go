package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
)

func moduloUsuarios(d dependenciasModulos) adaptadorHTTP.ModuloUsuarios {
	return adaptadorHTTP.ModuloUsuarios{Configuracion: adaptadorHTTP.ConfiguracionUsuarios{Servicio: identidad.NuevaAdministracionUsuarios(&postgres.Perfiles{}, d.base, postgres.NuevoRegistroAuditoria(d.base), relojIdentidad{})}}
}
