package main

import (
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetes "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloCarga(d dependenciasModulos) h.ModuloCarga {
	return h.ModuloCarga{Actas: &expediente.ServicioActas{Validador: lectura.Validador{}, Antivirus: d.antivirus, Lector: lectura.NuevoLector(), Registro: postgres.NuevoRegistroAuditoria(d.base), Reloj: relojIdentidad{}}, Opciones: catalogo.ConsultaDecision{Repositorio: &repo.Repositorio{}, Textos: paquete.NuevaPublicacion(d.disco, &paquetes.Referencias{}), Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoriaTransaccional(d.base), Ahora: time.Now}}
}
