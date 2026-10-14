package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
)

func moduloInstalacion(d dependenciasModulos) adaptadorHTTP.ModuloInstalacion {
	return adaptadorHTTP.ModuloInstalacion{Configuracion: adaptadorHTTP.ConfiguracionInstalacion{Instalacion: d.instalacion}}
}
