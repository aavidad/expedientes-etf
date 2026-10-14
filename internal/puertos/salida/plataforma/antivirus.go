package plataforma

import "context"

// EstadoAntivirus no decide la política de publicación. Cada caso de uso
// rechaza o advierte según el tipo de subida; sólo Limpio acredita revisión.
type EstadoAntivirus uint8

const (
	AntivirusNoDisponible EstadoAntivirus = iota
	AntivirusLimpio
	AntivirusInfectado
)

// Antivirus recibe contenido en memoria y lo revisa sin persistirlo. Sirve
// tanto a los paquetes de Administración como a los anexos de derivación.
type Antivirus interface {
	Revisar(context.Context, []byte) (EstadoAntivirus, error)
}
