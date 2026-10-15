package documento

import (
	"bytes"
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

func TestPlantillaDeDistribucionComoReservaDelIdiomaActivo(t *testing.T) {
	b, err := os.ReadFile("../../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	raiz := t.TempDir()
	ruta := ".versiones/es/2.0.0/" + d.Huella(b) + "/01234567890123456789012345678901/es"
	f := Plantillas{Raiz: raiz}
	activo := d.IdiomaPlantilla{Ruta: ruta, Version: "2.0.0"}
	plantilla := d.Plantilla{Idioma: "es", Tipo: d.ActaAceptacion, Version: "1.1.9", Ruta: "plantillas/entrada/aceptacion/v1.1.9.docx", Huella: d.Huella(b)}
	escribir := func(relativa string, contenido []byte) {
		t.Helper()
		nombre := filepath.Join(raiz, filepath.FromSlash(relativa))
		if err := os.MkdirAll(filepath.Dir(nombre), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(nombre, contenido, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// La base con otra huella no sirve como reserva.
	escribir("es/"+plantilla.Ruta, []byte("otra plantilla"))
	if _, err := f.Leer(t.Context(), activo, plantilla); err == nil {
		t.Fatal("reserva con huella distinta aceptada")
	}
	escribir("es/"+plantilla.Ruta, b)
	if _, err := f.Leer(t.Context(), activo, plantilla); err != nil {
		t.Fatal("versión nueva de la distribución no disponible", err)
	}
	// Una copia alterada en el paquete activo no impide usar la verificada, pero se avisa.
	escribir(ruta+"/"+plantilla.Ruta, []byte("alterada"))
	var avisos bytes.Buffer
	f.Registro = slog.New(slog.NewJSONHandler(&avisos, nil))
	if leida, err := f.Leer(t.Context(), activo, plantilla); err != nil || d.Huella(leida) != plantilla.Huella {
		t.Fatal("no se verificó la huella", err)
	}
	if !strings.Contains(avisos.String(), "plantilla.huella_paquete_activo") {
		t.Fatal("sin aviso de copia alterada", avisos.String())
	}
}

func TestPresentacionMemoriaDeDistribucionComoReserva(t *testing.T) {
	b, err := os.ReadFile("../../../../idiomas/es/plantillas/memoria/v1.0.0.json")
	if err != nil {
		t.Fatal(err)
	}
	raiz := t.TempDir()
	activo := d.IdiomaPlantilla{Ruta: ".versiones/es/2.0.0/h/n/es"}
	plantilla := d.Plantilla{Idioma: "es", Tipo: "memoria", Version: "1.0.0"}
	if _, err := (Plantillas{Raiz: raiz}).LeerMemoria(t.Context(), activo, plantilla); err == nil {
		t.Fatal("presentación inexistente aceptada")
	}
	nombre := filepath.Join(raiz, "es", "plantillas", "memoria", "v1.0.0.json")
	if err := os.MkdirAll(filepath.Dir(nombre), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nombre, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Plantillas{Raiz: raiz}).LeerMemoria(t.Context(), activo, plantilla); err != nil {
		t.Fatal("sin reserva de la distribución", err)
	}
}
