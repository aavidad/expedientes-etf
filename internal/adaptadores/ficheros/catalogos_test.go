package ficheros

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCatalogoInmutableConcurrenteYHuella(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "es", "catalogos"), 0700); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(dir, "")
	ctx := context.Background()
	m := map[string]string{"cargo.COORDINACION": "Coordinación provincial"}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := p.EscribirCatalogo(ctx, "es", m)
			if err != nil {
				t.Error(err)
				return
			}
			v, err := p.LeerCatalogo(ctx, r)
			if err != nil || v["cargo.COORDINACION"] != m["cargo.COORDINACION"] {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	r, err := p.EscribirCatalogo(ctx, "es", m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, r.Ruta), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.LeerCatalogo(ctx, r); err == nil {
		t.Fatal("huella adulterada admitida")
	}
	if _, err := p.LeerCatalogo(ctx, dominio.ReferenciaCatalogo{Codigo: "../es"}); err == nil {
		t.Fatal("salida de raíz")
	}
}

func TestLimpiezaConservaRegistradasYPreparacionesRecientes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "es", "catalogos"), 0700); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(dir, "")
	activa, err := p.EscribirCatalogo(ctx, "es", map[string]string{"cargo.COORDINACION": "Coordinación"})
	if err != nil {
		t.Fatal(err)
	}
	huerfana, err := p.EscribirCatalogo(ctx, "es", map[string]string{"cargo.COORDINACION": "Coordinación provincial"})
	if err != nil {
		t.Fatal(err)
	}
	reciente, err := p.EscribirCatalogo(ctx, "es", map[string]string{"cargo.COORDINACION": "Coordinación del Programa"})
	if err != nil {
		t.Fatal(err)
	}
	antigua := time.Now().Add(-48 * time.Hour)
	for _, r := range []dominio.ReferenciaCatalogo{activa, huerfana} {
		if err := os.Chtimes(filepath.Join(dir, r.Ruta), antigua, antigua); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.LimpiarCatalogo(ctx, "es", map[string]bool{activa.Huella: true}, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, r := range []dominio.ReferenciaCatalogo{activa, reciente} {
		if _, err := p.LeerCatalogo(ctx, r); err != nil {
			t.Fatal("versión conservada eliminada", err)
		}
	}
	if _, err := p.LeerCatalogo(ctx, huerfana); err == nil {
		t.Fatal("huérfano antiguo conservado")
	}
}
