package main

import (
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documento "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloEquipo(d dependenciasModulos) h.ModuloEquipo {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return h.ModuloEquipo{}
	}
	return h.ModuloEquipo{Servicio: &app.LecturaEquipo{Avisos: &expediente.Repositorio{}, Catalogos: paquete.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{}), IdiomaPredeterminado: d.cfg.Idioma, Asignacion: &catalogo.Repositorio{}, Catalogo: &catalogo.Repositorio{}, Repositorio: &expediente.Repositorio{}, Documentos: documento.Repositorio{}, Historial: postgres.HistorialExpediente{}, Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojExpediente{zona: zona}, Capacidades: []string{"familia", "menores", "intervencion", "consentimiento", "evaluacion", "documentos", "cierre", "salidas", "historial", "seguimiento"}}}
}
