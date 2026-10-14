// Package web contiene únicamente la compilación de Vite; los textos y temas
// permanecen en disco. Ejecutar make web antes de construir el binario final.
package web

import "embed"

//go:embed all:dist
var Compilacion embed.FS
