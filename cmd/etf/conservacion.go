package main

import (
	f "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	c "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	r "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

func moduloConservacion(d dependenciasModulos) h.ModuloConservacion {
	return h.ModuloConservacion{Servicio: &app.ServicioConservacion{Repositorio: &r.Repositorio{}, Catalogo: &c.Repositorio{}, Defecto: f.Conservacion{Ruta: "datos/catalogos/conservacion.json"}, Reloj: relojIdentidad{}, Transacciones: d.base, Registro: postgres.NuevoRegistroAuditoria(d.base)}}
}
