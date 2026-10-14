package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type Idioma = dominio.Idioma
type Tema = dominio.Tema
type Recurso = dominio.Recurso

type Textos interface {
	Idiomas(context.Context) ([]Idioma, error)
	Textos(context.Context, string) (map[string]string, error)
}

type Temas interface {
	Tema(context.Context, string) (Tema, error)
	Temas(context.Context) ([]Tema, error)
	RecursoTema(context.Context, string, string) (Recurso, error)
	Tokens(context.Context, string) (map[string]string, error)
}
