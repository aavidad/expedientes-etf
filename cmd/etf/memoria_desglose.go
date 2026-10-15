package main

import (
	http "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
)

func moduloDesgloseMemoria(d dependenciasModulos) http.ModuloDesgloseMemoria {
	return http.ModuloDesgloseMemoria{Servicio: moduloNavegacionMemoria(d).Servicio.(entrada.DesgloseCifraMemoria)}
}
