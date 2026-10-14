package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/hoja"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

func moduloExpedientes(d dependenciasModulos) adaptadorHTTP.ModuloExpedientes {
	registro := registroAvisosMemoria(postgres.NuevoRegistroAuditoria(d.base))
	c := configurarAceptaciones(d.base, d.cfg.IdiomasRuta, d.cfg.Idioma)
	c.Hoja = adaptadorHTTP.ConfiguracionHoja{Servicio: &app.ServicioHoja{Historicos: c.CargasHistoricas, Lector: hoja.Lector{}, Antivirus: d.antivirus, Reloj: relojIdentidad{}}, Textos: d.disco, Idioma: d.cfg.Idioma}
	c.SPM = app.Nuevo(app.Dependencias{Repositorio: &repo.Repositorio{}, SalidasSPM: &repo.Repositorio{}, Transacciones: d.base, Registro: registro, RegistroTransaccional: registro, Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{}})
	return adaptadorHTTP.ModuloExpedientes{Configuracion: c}
}
