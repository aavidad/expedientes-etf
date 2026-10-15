package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repositorio "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	catalogoApp "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	paqueteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
)

func moduloBandeja(d dependenciasModulos) adaptadorHTTP.ModuloBandeja {
	return adaptadorHTTP.ModuloBandeja{Servicio: expediente.Nuevo(expediente.Dependencias{
		Bandeja: &repositorio.Repositorio{}, Catalogo: &catalogo.Repositorio{}, Transacciones: d.base,
		Registro: postgres.NuevoRegistroAuditoria(d.base), RegistroTransaccional: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{},
		NombresMunicipios: catalogoApp.NombresUbicaciones{Publicacion: paqueteApp.NuevaPublicacion(d.disco, &paquetePostgres.Referencias{}), Idioma: d.cfg.Idioma},
	})}
}
