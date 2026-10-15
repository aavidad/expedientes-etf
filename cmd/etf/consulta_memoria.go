package main

import (
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	plantillas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	http "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documento "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	estadistica "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	docApp "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
)

func moduloConsultaMemoria(d dependenciasModulos) http.ModuloConsultaMemoria {
	f := plantillas.Plantillas{Raiz: d.cfg.IdiomasRuta, Paquetes: d.publicacion, Registro: d.registro}
	motor := app.EjecutorConsulta{Transacciones: d.calculoMemoria, Fuente: app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{CalculoInterno: true}, Reglas: reglas.Reglas{Raiz: "datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{CalculoInterno: true}}}
	return http.ModuloConsultaMemoria{Servicio: app.NuevoConsulta(app.DependenciasConsulta{Reglas: &catalogo.Repositorio{}, Cerradas: &estadistica.Repositorio{}, Calculo: motor, Word: docApp.WordCifras{Repositorio: documento.Repositorio{}, Idiomas: f, Ficheros: f, Textos: f, Motor: word.Motor{}, Metadatos: reglas.TextosCifras{Raiz: d.cfg.IdiomasRuta}}, Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}})}
}
func moduloNavegacionMemoria(d dependenciasModulos) http.ModuloNavegacionMemoria {
	return http.ModuloNavegacionMemoria{Servicio: app.ServicioNavegacion{Cierres: &estadistica.Repositorio{}, Fuente: app.FuenteConCierre{Repositorio: &estadistica.Repositorio{}, Fuente: fuenteCierreMemoria()}, Numeros: &expediente.LecturaEstadistica{}, Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}}}
}
