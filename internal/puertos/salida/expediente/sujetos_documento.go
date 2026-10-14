package expediente

import "context"

// SujetosDocumento permite identificar resultados ya anotados, también cuando
// la persona dejó de figurar en la composición vigente. El consumidor sólo
// conserva iniciales y audita la lectura en su transacción de emisión.
type SujetosDocumento interface {
	NombresSujetos(context.Context, string) (map[string]string, error)
}
