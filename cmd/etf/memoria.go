package main

import (
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	plantillas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documento "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	persistencia "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	documentoApp "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

func moduloNarrativas(d dependenciasModulos) adaptadorHTTP.ModuloNarrativas {
	return adaptadorHTTP.ModuloNarrativas{Configuracion: adaptadorHTTP.ConfiguracionNarrativas{Servicio: app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: &persistencia.Repositorio{}, Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}})}}
}

func moduloMemoriaWord(d dependenciasModulos) adaptadorHTTP.ModuloMemoriaWord {
	f := plantillas.Plantillas{Raiz: d.cfg.IdiomasRuta, Paquetes: d.publicacion, Registro: d.registro}
	fuente := fuenteConCierreMemoria()
	svc := documentoApp.NuevoMemoria(documentoApp.DependenciasMemoria{Repositorio: documento.Repositorio{}, Fuente: fuente, Idiomas: f, Ficheros: f, Textos: f, Motor: word.Motor{}, Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{}})
	return adaptadorHTTP.ModuloMemoriaWord{Configuracion: adaptadorHTTP.ConfiguracionMemoriaWord{Servicio: svc}}
}

func fuenteCierreMemoria() app.FuenteMemoria {
	return app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}, Reglas: reglas.Reglas{Raiz: "datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: &persistencia.Repositorio{}}
}
func registroAvisosMemoria(registro audit.RegistroTransaccional) app.RegistroConAvisos {
	return app.RegistroConAvisos{Registro: registro, Repositorio: &persistencia.Repositorio{}, Fuente: fuenteCierreMemoria(), Equipos: &expediente.LecturaEstadistica{}}
}
func moduloCierreMemoria(d dependenciasModulos) adaptadorHTTP.ModuloCierreMemoria {
	return adaptadorHTTP.ModuloCierreMemoria{Servicio: app.NuevoCierre(app.DependenciasCierre{Repositorio: &persistencia.Repositorio{}, Fuente: fuenteCierreMemoria(), Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}})}
}

func fuenteConCierreMemoria() app.FuenteConCierre {
	return app.FuenteConCierre{Repositorio: &persistencia.Repositorio{}, Fuente: fuenteCierreMemoria()}
}

func moduloLecturaMemoria(d dependenciasModulos) adaptadorHTTP.ModuloLecturaMemoria {
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuenteConCierreMemoria(), CifrasPublicadas: &persistencia.CifrasMemoria{}, Catalogos: &catalogo.Repositorio{}, Transacciones: d.base, Auditoria: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}})
	return adaptadorHTTP.ModuloLecturaMemoria{Configuracion: adaptadorHTTP.ConfiguracionLecturaMemoria{Servicio: svc, Equipos: svc, Consulta: moduloConsultaMemoria(d).Servicio}}
}
