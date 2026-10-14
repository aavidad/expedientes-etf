package main

import (
	documentos "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	"path/filepath"
	"time"
	_ "time/tzdata"

	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repositorio "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

func configurarAceptaciones(base *postgres.Base, idiomasRuta, idioma string) adaptadorHTTP.ConfiguracionAceptaciones {
	r := &repositorio.Repositorio{}
	c := &catalogo.Repositorio{}
	auditoria := registroAvisosMemoria(postgres.NuevoRegistroAuditoria(base))
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return adaptadorHTTP.ConfiguracionAceptaciones{}
	}
	servicio := expediente.Nuevo(expediente.Dependencias{
		Conservacion: &expediente.ServicioConservacion{Repositorio: r, Catalogo: c, Defecto: ficheros.Conservacion{Ruta: "datos/catalogos/conservacion.json"}}, Avisos: r, CargasHistoricas: r, Traslados: r, Seguimientos: r, SalidasServicio: r, Identificadores: identificadoresDerivacion{}, EmisionesFinalizacion: documentos.Repositorio{}, Cierres: r, ConsultaCierreSPM: expediente.ConsultaCierreSPM{Repositorio: r, Salidas: r},
		Repositorio: r, Propuestas: r, MotivosCierre: c, Aceptaciones: r, Decisiones: r, Hitos: r, Numeracion: c, Catalogo: c, Usuarios: postgres.NuevosUsuarios(base), Transacciones: base,
		Intervencion: r, CatalogosIntervencion: c,
		Menores: r, CatalogosMenores: ficheros.CatalogosMenores{Ruta: filepath.Join(idiomasRuta, idioma, "catalogos", "menores.json")},
		Familias: r, SMI: c, CatalogosFamilia: ficheros.CatalogosFamilia{Listas: filepath.Join(idiomasRuta, idioma, "catalogos", "listas.json"), Familia: filepath.Join(idiomasRuta, idioma, "catalogos", "familia.json"), Parametros: "datos/catalogos/familia.json"},
		Registro: auditoria, RegistroTransaccional: auditoria, Reloj: relojExpediente{zona: zona},
	})
	return adaptadorHTTP.ConfiguracionAceptaciones{Avisos: servicio, CargasHistoricas: servicio, TrasladoSeguimiento: servicio, Cierres: servicio, SalidasServicio: servicio, Servicio: servicio, Decisiones: servicio, Familia: servicio, Menores: servicio, Intervencion: servicio, Propuestas: servicio, Hitos: servicio}
}

type relojExpediente struct{ zona *time.Location }

func (r relojExpediente) Ahora() time.Time { return time.Now().In(r.zona) }
