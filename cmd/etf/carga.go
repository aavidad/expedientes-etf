package main

import (
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetes "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	"time"
)

func moduloCarga(d dependenciasModulos) h.ModuloCarga {
	return h.ModuloCarga{Opciones: catalogo.ConsultaDecision{Repositorio: &repo.Repositorio{}, Textos: paquete.NuevaPublicacion(d.disco, &paquetes.Referencias{}), Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoriaTransaccional(d.base), Ahora: time.Now}}
}
