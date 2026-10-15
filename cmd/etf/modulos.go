package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/plataforma"
	"log/slog"
)

// dependenciasModulos contiene los recursos del arranque; cada módulo crea sus
// casos de uso y conectores en su fichero de composición.
type dependenciasModulos struct {
	base           *postgres.Base
	calculoMemoria *postgres.Base
	antivirus      *clamav.Antivirus
	disco          *ficheros.Paquetes
	publicacion    *postgres.PublicacionPaquetes
	instalacion    entrada.Instalacion
	cfg            plataforma.Configuracion
	registro       *slog.Logger
}

func componerModulos(d dependenciasModulos) []adaptadorHTTP.Modulo {
	return []adaptadorHTTP.Modulo{
		moduloRestauracion(d), moduloBandeja(d), moduloSalidas(d), moduloSMI(d), moduloUsuarios(d), moduloCatalogos(d), moduloUbicaciones(d), moduloInstalacion(d),

		moduloDerivaciones(d), moduloExpedientes(d), moduloLecturaMemoria(d), moduloNarrativas(d), moduloConsultaMemoria(d), moduloNavegacionMemoria(d), moduloDesgloseMemoria(d), moduloCierreMemoria(d), moduloMemoriaWord(d), moduloDocumentos(d), moduloActividad(d), moduloSPAF(d), moduloEvaluacion(d), moduloAnexo(d), moduloDecision(d), moduloPlantillas(d),

		moduloEquipo(d), moduloFI1(), moduloAnexo6(d), moduloTraslado(d), moduloControlCargas(d), moduloCarga(d), moduloConservacion(d), moduloAuditoria(d), moduloVoz(),
	}
}
