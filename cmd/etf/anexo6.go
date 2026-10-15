package main

import (
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repositorio "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetes "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloAnexo6(d dependenciasModulos) h.ModuloAnexo6 {
	return h.ModuloAnexo6{Opciones: catalogo.ConsultaDestinosAnexo6{ConsultaDecision: catalogo.ConsultaDecision{Repositorio: &repositorio.Repositorio{}, Textos: paquete.NuevaPublicacion(d.disco, &paquetes.Referencias{}), Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoria(d.base), Ahora: time.Now}}}
}
