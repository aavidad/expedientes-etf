package documento

import (
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"os"
	"path/filepath"
	"testing"
)

type referenciasPrueba struct{ refs []p.Referencia }

func (r referenciasPrueba) Referencias(context.Context, string) ([]p.Referencia, error) {
	return r.refs, nil
}
func TestPlantillasIdiomaActivoHuellaYAislamiento(t *testing.T) {
	b, err := os.ReadFile("../../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	raiz := t.TempDir()
	ruta := ".versiones/es/2.0.0/" + d.Huella(b) + "/01234567890123456789012345678901/es"
	f := Plantillas{Raiz: raiz, Paquetes: referenciasPrueba{[]p.Referencia{{VersionInstalada: p.VersionInstalada{Tipo: "idioma", Codigo: "es", Ruta: ruta, Version: "2.0.0"}}}}}
	p := d.Plantilla{Idioma: "es", Tipo: d.ActaAceptacion, Version: "1.0.0", Ruta: "plantillas/entrada/aceptacion/v1.0.0.docx", Huella: d.Huella(b)}
	activo, err := f.Resolver(t.Context(), "es")
	if err != nil || activo.Ruta != ruta {
		t.Fatal(activo, err)
	}
	if _, err := f.Leer(t.Context(), activo, p); !errors.Is(err, d.ErrPlantilla) {
		t.Fatal("sirvió reserva aunque hay idioma activo", err)
	}
	nombre := filepath.Join(raiz, filepath.FromSlash(ruta), filepath.FromSlash(p.Ruta))
	if err = os.MkdirAll(filepath.Dir(nombre), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(nombre, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Leer(t.Context(), activo, p); err != nil {
		t.Fatal(err)
	}
	p.Huella = d.Huella([]byte("alterada"))
	if _, err = f.Leer(t.Context(), activo, p); !errors.Is(err, d.ErrHuella) {
		t.Fatal(err)
	}
	p.Huella = d.Huella(b)
	if _, err = f.Leer(t.Context(), d.IdiomaPlantilla{Ruta: "../es"}, p); !errors.Is(err, d.ErrPlantilla) {
		t.Fatal(err)
	}
	externo := t.TempDir()
	if err = os.WriteFile(filepath.Join(externo, "plantilla.docx"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(nombre); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(externo, "plantilla.docx"), nombre); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Leer(t.Context(), activo, p); !errors.Is(err, d.ErrPlantilla) {
		t.Fatal("enlace fuera de raíz", err)
	}
}
