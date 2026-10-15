package main

import (
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"os"
)

func moduloVoz() adaptadorHTTP.ModuloVoz {
	return adaptadorHTTP.ModuloVoz{Audios: os.DirFS("bin/voz"), Documentos: os.DirFS("docs")}
}
