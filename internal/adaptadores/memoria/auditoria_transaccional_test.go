package memoria

import (
	"context"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type transacciones struct{ fallar bool }

func (t transacciones) Ejecutar(ctx context.Context, _ plataforma.Identidad, op func(context.Context) error) error {
	if err := op(ctx); err != nil {
		return err
	}
	if t.fallar {
		return errors.New("commit.fallido")
	}
	return nil
}
func TestDobleNoAuditaTransaccionesRevertidas(t *testing.T) {
	for _, fallar := range []bool{false, true} {
		a := NuevaAuditoriaTransaccional(transacciones{fallar})
		err := a.Ejecutar(context.Background(), plataforma.Identidad{}, func(ctx context.Context) error {
			_, err := a.Confirmar(ctx, dominio.Datos{OcurridoEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), UsuarioID: "sistema", Accion: "catalogo.publicar", Entidad: "catalogo", Resultado: "ok"})
			return err
		})
		if (err != nil) != fallar {
			t.Fatal(err)
		}
		n := len(a.Eventos())
		if (n == 0) != fallar {
			t.Fatal(n)
		}
	}
}
