package memoria

import (
	"context"
	"encoding/json"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"sync"
	"time"
)

type borrador struct {
	datos        []byte
	usuario      string
	hasta        time.Time
	temporizador *time.Timer
}
type Borradores struct {
	mu       sync.Mutex
	datos    map[string]borrador
	duracion time.Duration
	maximo   int
}

func NuevosBorradores(duracion time.Duration, maximo int) *Borradores {
	return &Borradores{datos: make(map[string]borrador), duracion: duracion, maximo: maximo}
}
func (b *Borradores) Poner(ctx context.Context, d dominio.Derivacion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	datos, err := json.Marshal(d)
	if err != nil {
		return dominio.ErrNoDisponible
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.datos) >= b.maximo {
		clear(datos)
		return dominio.ErrNoDisponible
	}
	id := d.Lectura.ID
	temporizador := time.AfterFunc(b.duracion, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if anterior, ok := b.datos[id]; ok {
			clear(anterior.datos)
			delete(b.datos, id)
		}
	})
	b.datos[id] = borrador{datos: datos, usuario: d.Lectura.LeidaPor, hasta: time.Now().Add(b.duracion), temporizador: temporizador}
	return nil
}
func (b *Borradores) Obtener(ctx context.Context, usuario, id string) (dominio.Derivacion, error) {
	if err := ctx.Err(); err != nil {
		return dominio.Derivacion{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	d, ok := b.datos[id]
	if !ok || d.usuario != usuario || !time.Now().Before(d.hasta) {
		return dominio.Derivacion{}, dominio.ErrNoEncontrada
	}
	var resultado dominio.Derivacion
	if err := json.Unmarshal(d.datos, &resultado); err != nil {
		return resultado, dominio.ErrNoDisponible
	}
	return resultado, nil
}
func (b *Borradores) Descartar(ctx context.Context, usuario, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if d, ok := b.datos[id]; ok && d.usuario == usuario {
		d.temporizador.Stop()
		clear(d.datos)
		delete(b.datos, id)
	}
	return nil
}
