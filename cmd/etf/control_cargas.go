package main

import (
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	r "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

func moduloControlCargas(d dependenciasModulos) h.ModuloControlCargas {
	registro := postgres.NuevoRegistroAuditoria(d.base)
	return h.ModuloControlCargas{Servicio: app.Nuevo(app.Dependencias{CargasHistoricas: &r.Repositorio{}, Transacciones: d.base, Registro: registro, RegistroTransaccional: registro, Reloj: relojIdentidad{}})}
}
