package postgres

import (
	"context"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"os"
	"testing"
)

// El ensayo SEG4 proporciona una revisión abierta con datos inventados.
func TestIntegracionRevisionRestauracionLecturaAuditada(t *testing.T) {
	id := os.Getenv("ETF_SEG4_ADMIN")
	if id == "" {
		t.Skip("ETF_SEG4_ADMIN")
	}
	b, ctx := baseIntegracion(t)
	actor := plataforma.Identidad{UsuarioID: id, Perfiles: []string{"administracion"}}
	if err := b.Ejecutar(ctx, actor, func(ctx context.Context) error {
		r, err := (&Restauracion{}).Leer(ctx)
		if err != nil {
			return err
		}
		if len(r.Cambios) < 4 || r.CopiaEn.IsZero() {
			t.Fatal("revisión incompleta")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	usuarios := aplicacion.NuevaAdministracionUsuarios(&Perfiles{}, b, NuevoRegistroAuditoria(b), relojPersistencia{})
	servicio := aplicacion.NuevaRevisionRestauracion(usuarios, &Restauracion{})
	a := entrada.Acceso{Estado: dominio.Activo, Usuario: dominio.Usuario{ID: id, Estado: dominio.Activo, Perfiles: []dominio.Perfil{dominio.Administracion}, RevisionRestauracion: true}}
	if _, err := servicio.Leer(ctx, a); err != nil {
		t.Fatal("lectura auditada", err)
	}
}
