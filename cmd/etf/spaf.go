package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
)

func moduloSPAF(d dependenciasModulos) adaptadorHTTP.ModuloSPAF {
	// La proyección histórica deja explícitos los datos todavía no recogidos.
	s := estadistica.NuevoSPAF(estadistica.DependenciasSPAF{Lectura: &expediente.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}, Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}, Catalogo: &catalogo.Repositorio{}, Etiquetas: paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{})})
	cfg := adaptadorHTTP.ConfiguracionSPAF{Servicio: s, Opciones: s}
	return adaptadorHTTP.ModuloSPAF{Configuracion: cfg}
}
