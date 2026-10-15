package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria"
	"time"
)

func moduloAuditoria(d dependenciasModulos) h.ModuloAuditoria {
	return h.ModuloAuditoria{Configuracion: h.ConfiguracionAuditoria{Servicio: auditoria.NuevoExportador(postgres.LecturaExportacionAuditoria{}, d.base, ficheros.NuevasExportacionesAuditoria(), postgres.NuevoRegistroAuditoria(d.base), time.Now), Textos: ficheros.NuevosActivos(d.disco, d.publicacion), Registro: d.registro}}
}
