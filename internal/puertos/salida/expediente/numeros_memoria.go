package expediente

import "context"

// NumerosMemoria resuelve de una vez los números, con la RLS del llamador.
type NumerosMemoria interface {
	NumerosMemoria(context.Context, []string) (map[string]string, error)
}
