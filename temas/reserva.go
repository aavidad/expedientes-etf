// Package temas conserva una copia de datos del tema predeterminado para recuperación.
package temas

import (
	"embed"
	"io/fs"
)

//go:embed expedientes-etf/tema.json expedientes-etf/tokens.json expedientes-etf/fuentes/*.woff2 expedientes-etf/iconos/*.png
var reserva embed.FS

func Predeterminado() fs.FS { return reserva }
