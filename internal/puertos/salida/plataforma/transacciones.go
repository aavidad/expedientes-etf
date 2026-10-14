package plataforma

import "context"

// Identidad contiene exclusivamente valores de la sesión validada por el
// servidor. EquipoID es único; el valor cero representa una operación sin
// identidad y debe quedar sin acceso por las políticas RLS.
type Identidad struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

// Transacciones confirma la operación sólo si termina sin error. El contexto
// entregado a la operación debe pasarse a todos sus repositorios y a auditoría:
// comparten la misma transacción. No se entrega el resultado antes de confirmar.
// Las llamadas anidadas se rechazan; no hay reintentos automáticos de escrituras.
type Transacciones interface {
	Ejecutar(context.Context, Identidad, func(context.Context) error) error
}
