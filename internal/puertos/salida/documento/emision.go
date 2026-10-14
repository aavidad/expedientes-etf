package documento

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"time"
)

// FuenteDatos es el contrato neutro que implementará expediente. Debe leer
// dentro de la transacción recibida y derivar equipo y versiones en servidor.
// No admite datos ni pertenencia suministrados por el cliente HTTP.
type FuenteDatos interface {
	Obtener(context.Context, d.Actor, d.Solicitud) (d.Datos, error)
}
type Firmantes interface {
	Vigente(context.Context, string, time.Time) (d.Firmante, error)
}
type GeneradorDocumentos interface {
	Generar(context.Context, []byte, d.Relleno) ([]byte, error)
	Version() string
}
type Idiomas interface {
	Resolver(context.Context, string) (d.IdiomaPlantilla, error)
}
type FicherosPlantilla interface {
	Leer(context.Context, d.IdiomaPlantilla, d.Plantilla) ([]byte, error)
}
type Repositorio interface {
	Bloquear(context.Context, d.Actor, d.Solicitud) error
	Peticion(context.Context, d.Actor, string) (d.Emision, string, error)
	Activa(context.Context, string, string) (d.Plantilla, error)
	Ultima(context.Context, d.Solicitud) (d.Emision, error)
	Guardar(context.Context, d.Actor, d.Solicitud, string, d.Emision) error
}
type RegistroTransaccional interface {
	RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error)
}
type Reloj interface{ Ahora() time.Time }
type Identificadores interface{ Nuevo() (string, error) }

// Presentacion lee el paquete de idioma fijado y la referencia de catálogo.
type Presentacion interface {
	Cargar(context.Context, d.IdiomaPlantilla, p.ReferenciaCatalogo) (map[string]string, map[string]string, error)
}
