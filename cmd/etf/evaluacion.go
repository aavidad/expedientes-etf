package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/evaluacion/ficheros"
	http "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	ev "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/evaluacion"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion"
	"path/filepath"
	"time"
)

func moduloEvaluacion(d dependenciasModulos) http.ModuloEvaluacion {
	return http.ModuloEvaluacion{Servicio: servicioEvaluacion(d)}
}

func servicioEvaluacion(d dependenciasModulos) *evaluacion.Servicio {
	return &evaluacion.Servicio{Repo: &ev.Repositorio{}, Catalogo: ficheros.Catalogo{Ruta: filepath.Join(d.cfg.IdiomasRuta, d.cfg.Idioma, "catalogos", "evaluacion-v1.reglas")}, Familias: &exp.Repositorio{}, MenoresDocumento: &exp.Repositorio{}, SujetosDocumento: &exp.Repositorio{}, Adultos: exp.ConsultaAdultos{}, Tx: d.base, Auditoria: postgres.NuevoRegistroAuditoriaTransaccional(d.base), Ahora: time.Now}
}
