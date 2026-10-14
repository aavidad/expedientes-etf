package auditoria

import (
	"context"
	"time"
)

type ApunteExpediente struct {
	ID        string
	Accion    string
	Fecha     time.Time
	Autor     string
	Resultado string
}

type HistorialExpediente interface {
	HistorialExpediente(context.Context, string) ([]ApunteExpediente, error)
}
