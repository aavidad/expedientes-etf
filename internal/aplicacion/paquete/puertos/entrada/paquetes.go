package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type Idioma = dominio.Idioma
type Tema = dominio.Tema
type Recurso = dominio.Recurso

// Paquetes ofrece sólo lectura de los datos necesarios para arrancar la interfaz.
type Paquetes interface {
	Idiomas(context.Context) ([]Idioma, error)
	Textos(context.Context, string) (map[string]string, error)
	Tema(context.Context, string) (Tema, error)
	Temas(context.Context) ([]Tema, error)
	RecursoTema(context.Context, string, string) (Recurso, error)
	Tokens(context.Context, string) (map[string]string, error)
}
