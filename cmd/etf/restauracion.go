package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
)

func moduloRestauracion(d dependenciasModulos) adaptadorHTTP.ModuloRestauracion {
	usuarios := identidad.NuevaAdministracionUsuarios(&postgres.Perfiles{}, d.base, postgres.NuevoRegistroAuditoria(d.base), relojIdentidad{})
	return adaptadorHTTP.ModuloRestauracion{Servicio: identidad.NuevaRevisionRestauracion(usuarios, &postgres.Restauracion{}), Catalogos: moduloCatalogos(d).Configuracion.Consulta}
}
