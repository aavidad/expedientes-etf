package evaluacion

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
)

type Repositorio interface {
	Consultar(context.Context, string) (d.Registro, error)
	Version(context.Context, string, string) (int, error)
	Guardar(context.Context, d.Cambio) error
	Repetir(context.Context, string, string, string, string) (d.Cambio, bool, error)
	Recordar(context.Context, string, string, string, string, d.Cambio) error
}
type Catalogo interface {
	Escala(context.Context, string) (d.Escala, error)
	Instrumento(context.Context, string) (bool, error)
}

// ConsultaMenores se inyectará cuando F2.01b publique su contrato. Hasta entonces
// se rechazan sujetos menores, sin guardar identificadores sin pertenencia.
type ConsultaMenores interface {
	Pertenece(context.Context, string, string, string) (bool, error)
}

// ConsultaProfesionales verifica adscripción al equipo en la fecha aplicada.
// No se sustituye por la pertenencia actual de la sesión.
type ConsultaProfesionales interface {
	Pertenece(context.Context, string, string, string) (bool, error)
}

// CatalogoPublicado expone únicamente las definiciones inmutables disponibles.
type CatalogoPublicado interface {
	Definiciones(context.Context) (d.Definiciones, error)
}
