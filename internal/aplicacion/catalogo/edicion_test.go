package catalogo

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type repoEdicion struct{ publicaciones int }

func (*repoEdicion) Ultima(context.Context, string) (dominio.Datos, error) {
	return dominio.Datos{}, nil
}
func (r *repoEdicion) Publicar(context.Context, string, dominio.Cambio) (dominio.Version, error) {
	r.publicaciones++
	return dominio.Version{ID: "00000000-0000-4000-8000-000000000013"}, nil
}
func (*repoEdicion) Firmantes(context.Context) ([]dominio.Firmante, error) { return nil, nil }
func (*repoEdicion) GuardarFirmante(context.Context, dominio.Cambio) (dominio.Firmante, error) {
	return dominio.Firmante{}, nil
}

type paqueteEdicion struct{ fallo error }

func (*paqueteEdicion) Catalogos(context.Context) (paquetes.Catalogos, error) {
	return paquetes.Catalogos{}, nil
}
func (*paqueteEdicion) Preparar(context.Context, string, map[string]string, map[string]int) (paquetes.Preparada, error) {
	return paquetes.Preparada{}, nil
}
func (p *paqueteEdicion) Activar(context.Context, paquetes.Preparada) error { return p.fallo }

type txEdicion struct{ llamadas int }

func (tx *txEdicion) Ejecutar(ctx context.Context, _ plataforma.Identidad, op func(context.Context) error) error {
	tx.llamadas++
	return op(ctx)
}

type registroEdicion struct{ fallo error }

func (r registroEdicion) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, r.fallo
}
func TestEdicionNoEscribeSinPermisoNiTrasConflicto(t *testing.T) {
	r := &repoEdicion{}
	tx := &txEdicion{}
	p := &paqueteEdicion{fallo: &paquetes.ConflictoPublicacion{Actual: 2, Codigo: "es"}}
	e := NuevaEdicion(r, p, tx, registroEdicion{}, time.Now)
	c := entrada.Cambio{VigenteDesde: time.Now(), Plazo: &dominio.Plazo{Clave: "consentimiento", Valor: 30, Unidad: "dias"}}
	if _, err := e.Guardar(context.Background(), entrada.Actor{UsuarioID: "persona", Perfiles: []string{"equipo"}}, "plazos", c); !errors.Is(err, dominio.ErrPermiso) {
		t.Fatal(err)
	}
	if tx.llamadas != 0 {
		t.Fatal("transacción sin permiso")
	}
	_, err := e.Guardar(context.Background(), entrada.Actor{UsuarioID: "persona", Perfiles: []string{"administracion"}}, "plazos", c)
	var conflicto *paquetes.ConflictoPublicacion
	if !errors.As(err, &conflicto) || r.publicaciones != 0 {
		t.Fatal(err, r.publicaciones)
	}
}

func (*paqueteEdicion) TextosVersion(context.Context, string, string) (map[string]string, error) {
	return nil, nil
}

func (*paqueteEdicion) LimpiarHuerfanos(context.Context, time.Time) error { return nil }
