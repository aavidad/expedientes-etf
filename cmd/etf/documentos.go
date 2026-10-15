package main

import (
	firmantesDocumento "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	registroManual "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/registro_manual"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	plantillasDocumento "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documentoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expedientePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	expedienteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"time"
)

func moduloDocumentos(d dependenciasModulos) adaptadorHTTP.ModuloDocumentos {
	plantillas := plantillasDocumento.Plantillas{Raiz: d.cfg.IdiomasRuta, Paquetes: d.publicacion, Registro: d.registro}
	registroDocumento := postgres.NuevoRegistroAuditoria(d.base)
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		d.registro.Warn("documento.no_disponible")
		return adaptadorHTTP.ModuloDocumentos{}
	}
	servicioDocumentos := documento.Nuevo(documento.Dependencias{
		Datos:               documento.DatosEntrada{Evaluaciones: servicioEvaluacion(d), Salidas: configurarAceptaciones(d.base, d.cfg.IdiomasRuta, d.cfg.Idioma).SalidasServicio, Expedientes: expedienteApp.ConsultaDocumentos{Repositorio: &expedientePostgres.Repositorio{}}, Catalogos: &catalogoPostgres.Repositorio{}, Referencias: &paquetePostgres.Referencias{}, Presentacion: plantillas, Reloj: relojExpediente{zona: zona}},
		RepositorioRegistro: documentoPostgres.Repositorio{}, RegistroSalida: registroManual.Registro{},
		Repositorio: documentoPostgres.Repositorio{}, Firmantes: firmantesDocumento.Firmantes{Catalogo: &catalogoPostgres.Repositorio{}},
		Motor: word.Motor{}, Idiomas: plantillas, Ficheros: plantillas, Transacciones: d.base,
		Registro: registroDocumento, RegistroTransaccional: registroDocumento, Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{},
	})
	calculoSPAF := estadistica.NuevoSPAF(estadistica.DependenciasSPAF{Lectura: &expedientePostgres.LecturaEstadistica{Reglas: &catalogoPostgres.Repositorio{}}, Transacciones: d.base, Auditoria: registroDocumento, Reloj: relojIdentidad{}})
	servicioSPAFWord := documento.NuevoEmisionSPAF(documento.DependenciasEmisionSPAF{Calculo: calculoSPAF, Repositorio: documentoPostgres.Repositorio{}, Catalogos: &catalogoPostgres.Repositorio{}, Referencias: plantillas, Presentacion: plantillas, Idiomas: plantillas, Ficheros: plantillas, Motor: word.MotorSPAF{}, Transacciones: d.base, Auditoria: registroDocumento, Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{}})
	administracion := documento.NuevaAdministracionPlantillas(documento.DependenciasPlantillas{
		Repositorio: documentoPostgres.Repositorio{}, Gestion: documentoPostgres.Repositorio{}, Almacen: plantillas, Validador: word.ValidadorPlantillas{}, Antivirus: d.antivirus,
		Idiomas: plantillas, Ficheros: plantillas, Transacciones: d.base, Registro: registroDocumento,
		RegistroTransaccional: registroDocumento, Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{},
	})
	return adaptadorHTTP.ModuloDocumentos{Configuracion: adaptadorHTTP.ConfiguracionDocumentos{Servicio: servicioDocumentos, Registros: servicioDocumentos, Plantillas: administracion, GestionPlantillas: administracion, SPAF: servicioSPAFWord}}
}
