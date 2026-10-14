package auditoria_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

// Doble de pruebas, sin adaptador de producción ni dependencia de PostgreSQL.
type registroMemoria struct {
	mu      sync.Mutex
	eventos []dominio.Evento
	err     error
}

var _ salida.Registro = (*registroMemoria)(nil)

func (r *registroMemoria) Confirmar(ctx context.Context, d dominio.Datos) (dominio.Evento, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return dominio.Evento{}, err
	}
	if r.err != nil {
		return dominio.Evento{}, r.err
	}
	anterior := dominio.Inicio()
	if len(r.eventos) > 0 {
		anterior = r.eventos[len(r.eventos)-1].Punto()
	}
	e, err := dominio.Encadenar(anterior.ID+1, d, anterior)
	if err != nil {
		return dominio.Evento{}, err
	}
	r.eventos = append(r.eventos, e)
	e.Perfiles = append([]string{}, e.Perfiles...)
	return e, nil
}

func datos() dominio.Datos {
	return dominio.Datos{OcurridoEn: time.Date(2026, 10, 13, 8, 42, 11, 0, time.UTC), UsuarioID: "sistema", Perfiles: []string{"administracion"}, Accion: "auditoria.exportar", Entidad: "auditoria", Resultado: "ok"}
}

type registroFunc func(context.Context, dominio.Datos) (dominio.Evento, error)

func (f registroFunc) Confirmar(ctx context.Context, d dominio.Datos) (dominio.Evento, error) {
	return f(ctx, d)
}

func TestConfirmaAntesDelPrimerByte(t *testing.T) {
	r := &registroMemoria{}
	s := aplicacion.NuevoRegistro(r)
	entregas := 0
	e, err := s.RegistrarAntesDeEntregar(context.Background(), datos(), func(context.Context) error {
		if len(r.eventos) != 1 {
			t.Fatal("entrega antes de confirmar")
		}
		entregas++
		return nil
	})
	if err != nil || entregas != 1 || e.ID != 1 {
		t.Fatalf("evento: %+v, error: %v", e, err)
	}
	// Los resultados del caso de uso tampoco comparten las colecciones guardadas.
	e.Perfiles[0] = "consulta"
	if err := dominio.Verificar(r.eventos, dominio.Inicio(), r.eventos[0].Punto()); err != nil {
		t.Fatal(err)
	}
}

func TestRegistroFallidoNoEntregaNiFiltraLaCausa(t *testing.T) {
	causa := errors.New("fichero_privado_inaccesible")
	r := &registroMemoria{err: causa}
	e, err := aplicacion.NuevoRegistro(r).RegistrarAntesDeEntregar(context.Background(), datos(), func(context.Context) error { t.Fatal("entregó sin registro"); return nil })
	if !errors.Is(err, causa) || err.Error() != "auditoria.registro_fallido" || e.ID != 0 || len(r.eventos) != 0 {
		t.Fatalf("%+v, %v", e, err)
	}
}

func TestCancelacionAntesYDespuesDeConfirmar(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	r := &registroMemoria{}
	noEntregar := func(context.Context) error { t.Fatal("entrega cancelada"); return nil }
	if _, err := aplicacion.NuevoRegistro(r).RegistrarAntesDeEntregar(ctx, datos(), noEntregar); !errors.Is(err, context.Canceled) || len(r.eventos) != 0 {
		t.Fatal(err)
	}
	ctx, cancelar = context.WithCancel(context.Background())
	f := registroFunc(func(ctx context.Context, d dominio.Datos) (dominio.Evento, error) {
		e, err := r.Confirmar(ctx, d)
		cancelar()
		return e, err
	})
	e, err := aplicacion.NuevoRegistro(f).RegistrarAntesDeEntregar(ctx, datos(), noEntregar)
	if !errors.Is(err, context.Canceled) || len(r.eventos) != 1 || e.ID != 1 {
		t.Fatal(err)
	}
}

func TestFalloDeEntregaConservaEvento(t *testing.T) {
	r := &registroMemoria{}
	causa := errors.New("destino_cerrado")
	e, err := aplicacion.NuevoRegistro(r).RegistrarAntesDeEntregar(context.Background(), datos(), func(context.Context) error { return causa })
	if !errors.Is(err, causa) || err.Error() != "auditoria.entrega_fallida" || e.ID != 1 || len(r.eventos) != 1 {
		t.Fatal(err)
	}
}

func TestEntradaYConfirmacionInvalidasNoEntregan(t *testing.T) {
	noEntregar := func(context.Context) error { t.Fatal("entrega inválida"); return nil }
	r := &registroMemoria{}
	d := datos()
	d.UsuarioID = "Inés Moreno"
	if _, err := aplicacion.NuevoRegistro(r).RegistrarAntesDeEntregar(context.Background(), d, noEntregar); err == nil || len(r.eventos) != 0 {
		t.Fatal("datos inválidos")
	}
	for _, s := range []*aplicacion.Registro{nil, aplicacion.NuevoRegistro(nil)} {
		if _, err := s.RegistrarAntesDeEntregar(context.Background(), datos(), noEntregar); err == nil {
			t.Fatal("sin registro")
		}
	}
	if _, err := aplicacion.NuevoRegistro(r).RegistrarAntesDeEntregar(context.Background(), datos(), nil); err == nil || len(r.eventos) != 0 {
		t.Fatal("sin entrega")
	}
	for nombre, cambiar := range map[string]func(*dominio.Evento){
		"vacio":     func(e *dominio.Evento) { *e = dominio.Evento{} },
		"metadatos": func(e *dominio.Evento) { e.Accion = "expediente.listar" },
		"huella":    func(e *dominio.Evento) { e.Huella = dominio.HuellaInicial },
		"metadatos_recalculados": func(e *dominio.Evento) {
			d := datos()
			d.Accion = "expediente.listar"
			*e, _ = dominio.Encadenar(1, d, dominio.Inicio())
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			f := registroFunc(func(ctx context.Context, d dominio.Datos) (dominio.Evento, error) {
				e, _ := dominio.Encadenar(1, d, dominio.Inicio())
				cambiar(&e)
				return e, nil
			})
			if _, err := aplicacion.NuevoRegistro(f).RegistrarAntesDeEntregar(context.Background(), datos(), noEntregar); err == nil || err.Error() != "auditoria.confirmacion_invalida" {
				t.Fatal(err)
			}
		})
	}
}

func TestConfirmacionesConcurrentesFormanUnaCadena(t *testing.T) {
	r := &registroMemoria{}
	s := aplicacion.NuevoRegistro(r)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.RegistrarAntesDeEntregar(context.Background(), datos(), func(context.Context) error { return nil }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(r.eventos) != 64 {
		t.Fatalf("eventos: %d", len(r.eventos))
	}
	if err := dominio.Verificar(r.eventos, dominio.Inicio(), r.eventos[63].Punto()); err != nil {
		t.Fatal(err)
	}
}
