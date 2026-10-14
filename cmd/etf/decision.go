package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documentoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloDecision(d dependenciasModulos) adaptadorHTTP.ModuloDecision {
	registro := registroAvisosMemoria(postgres.NuevoRegistroAuditoriaTransaccional(d.base))
	return adaptadorHTTP.ModuloDecision{
		Opciones:  catalogo.ConsultaDecision{Repositorio: &catalogoPostgres.Repositorio{}, Textos: paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{}), Transacciones: d.base, Registro: registro, Ahora: time.Now},
		Emisiones: documento.ConsultaDecision{Idiomas: paquete.NuevaConsulta(ficheros.NuevosActivos(d.disco, d.publicacion), ficheros.NuevosActivos(d.disco, d.publicacion)), Repositorio: documentoPostgres.Repositorio{}, Registros: documentoPostgres.Repositorio{}, Transacciones: d.base, Auditoria: registro, Ahora: time.Now},
	}
}
