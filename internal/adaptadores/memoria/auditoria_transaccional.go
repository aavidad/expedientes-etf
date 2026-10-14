package memoria

import (
	"context"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"sync"
)

// AuditoriaTransaccional es el doble provisional de F0.08b. Prepara eventos
// dentro del contexto de negocio y sólo los incorpora si la base confirma.
// No es persistente ni cubre una caída del proceso después del COMMIT.
type AuditoriaTransaccional struct {
	mu      sync.Mutex
	base    plataforma.Transacciones
	eventos []dominio.Evento
}
type pendientes struct {
	eventos  []dominio.Evento
	anterior dominio.Punto
}
type claveAuditoria struct{}

func NuevaAuditoriaTransaccional(base plataforma.Transacciones) *AuditoriaTransaccional {
	return &AuditoriaTransaccional{base: base}
}
func (a *AuditoriaTransaccional) Ejecutar(ctx context.Context, i plataforma.Identidad, op func(context.Context) error) error {
	if ctx.Value(claveAuditoria{}) != nil {
		return errors.New("auditoria.transaccion_anidada")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p := &pendientes{anterior: dominio.Inicio()}
	if len(a.eventos) > 0 {
		p.anterior = a.eventos[len(a.eventos)-1].Punto()
	}
	err := a.base.Ejecutar(ctx, i, func(ctx context.Context) error { return op(context.WithValue(ctx, claveAuditoria{}, p)) })
	if err == nil {
		a.eventos = append(a.eventos, p.eventos...)
	}
	return err
}
func (a *AuditoriaTransaccional) Confirmar(ctx context.Context, d dominio.Datos) (dominio.Evento, error) {
	p, ok := ctx.Value(claveAuditoria{}).(*pendientes)
	if !ok {
		return dominio.Evento{}, errors.New("auditoria.transaccion_requerida")
	}
	if err := ctx.Err(); err != nil {
		return dominio.Evento{}, err
	}
	d.Perfiles = append([]string{}, d.Perfiles...)
	e, err := dominio.Encadenar(p.anterior.ID+1, d, p.anterior)
	if err != nil {
		return dominio.Evento{}, err
	}
	p.eventos = append(p.eventos, e)
	p.anterior = e.Punto()
	e.Perfiles = append([]string{}, e.Perfiles...)
	return e, nil
}
func (a *AuditoriaTransaccional) Eventos() []dominio.Evento {
	a.mu.Lock()
	defer a.mu.Unlock()
	eventos := append([]dominio.Evento{}, a.eventos...)
	for i := range eventos {
		eventos[i].Perfiles = append([]string{}, eventos[i].Perfiles...)
	}
	return eventos
}
