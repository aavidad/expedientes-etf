package main

import (
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repositorio "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

func moduloBandeja(d dependenciasModulos) adaptadorHTTP.ModuloBandeja {
	return adaptadorHTTP.ModuloBandeja{Servicio: expediente.Nuevo(expediente.Dependencias{
		Bandeja: &repositorio.Repositorio{}, Catalogo: &catalogo.Repositorio{}, Transacciones: d.base,
		Registro: postgres.NuevoRegistroAuditoria(d.base), RegistroTransaccional: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{},
		NombresMunicipios: nombres.NuevosMunicipios(d.cfg.IdiomasRuta, d.cfg.Idioma),
	})}
}
