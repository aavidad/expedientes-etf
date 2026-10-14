package paquete

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type validadorPreparacion struct{}

func (validadorPreparacion) Validar(context.Context, string, []byte) (dominio.ContenidoInstalacion, error) {
	return dominio.ContenidoInstalacion{Tipo: "idioma", Codigo: "fr", Version: "1.0.0"}, nil
}

type antivirusPreparacion struct {
	estado plataforma.EstadoAntivirus
	fallo  error
}

func (a antivirusPreparacion) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return a.estado, a.fallo
}

type registroPreparacion struct{ fallo error }

func (r registroPreparacion) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, r.fallo
}

func TestPreparacionAntivirusObligatorioAuditoriaYCaducidad(t *testing.T) {
	a := dominio.Actor{UsuarioID: "00000000-0000-4000-8000-000000000012", Perfiles: []string{"administracion"}}
	for _, caso := range []struct {
		nombre                     string
		estado                     plataforma.EstadoAntivirus
		fallo, auditoria, esperado error
	}{
		{"infectado", plataforma.AntivirusInfectado, nil, nil, dominio.ErrInfectado},
		{"indisponible", plataforma.AntivirusNoDisponible, nil, nil, dominio.ErrAntivirus},
		{"error_aunque_limpio", plataforma.AntivirusLimpio, errors.New("antivirus.no_disponible"), nil, dominio.ErrAntivirus},
		{"auditoria_caida", plataforma.AntivirusLimpio, nil, errors.New("auditoria.persistencia_fallida"), dominio.ErrNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			i := NuevoInstalador(validadorPreparacion{}, antivirusPreparacion{caso.estado, caso.fallo}, nil, nil, registroPreparacion{caso.auditoria}, time.Now)
			p, err := i.Preparar(context.Background(), a, "idioma", "fr", nil)
			if !errors.Is(err, caso.esperado) || p.Ticket != "" || len(i.pendientes) != 0 || len(i.plazas) != 0 {
				t.Fatal(p, err)
			}
		})
	}
	ahora := time.Now()
	i := NuevoInstalador(validadorPreparacion{}, antivirusPreparacion{estado: plataforma.AntivirusLimpio}, nil, nil, registroPreparacion{}, func() time.Time { return ahora })
	uno, err := i.Preparar(context.Background(), a, "idioma", "fr", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = i.Preparar(context.Background(), a, "idioma", "fr", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = i.Preparar(context.Background(), a, "idioma", "fr", nil); !errors.Is(err, dominio.ErrOcupado) {
		t.Fatal(err)
	}
	otro := a
	otro.UsuarioID = "00000000-0000-4000-8000-000000000013"
	if _, err = i.Activar(context.Background(), otro, uno.Ticket, 0); !errors.Is(err, dominio.ErrPreparacion) {
		t.Fatal(err)
	}
	if err = i.Descartar(context.Background(), a, uno.Ticket); err != nil {
		t.Fatal(err)
	}
	if len(i.plazas) != 1 {
		t.Fatal("descarte no libera plaza")
	}
	ahora = ahora.Add(11 * time.Minute)
	i.expirar()
	if len(i.plazas) != 0 || len(i.pendientes) != 0 {
		t.Fatal("caducidad")
	}
	if _, err = i.Activar(context.Background(), a, uno.Ticket, 0); !errors.Is(err, dominio.ErrPreparacion) {
		t.Fatal(err)
	}
}
