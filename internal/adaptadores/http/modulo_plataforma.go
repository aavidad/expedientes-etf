package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"io/fs"
)

type ModuloPlataforma struct {
	paquetes      entrada.Paquetes
	web           fs.FS
	idioma, tema  string
	avisos        []string
	comprobarBase func(context.Context) error
	sesion        *sesionHTTP
}

func (m ModuloPlataforma) Rutas() []Ruta {
	return rutasPlataforma(m.paquetes, m.web, m.idioma, m.tema, m.avisos, m.comprobarBase, m.sesion)
}
