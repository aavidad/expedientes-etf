package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"os"
	"path/filepath"
	"testing"
)

func TestPresentacionFijaIdiomaActivoYCatalogoPublicado(t *testing.T) {
	root := t.TempDir()
	escribir := func(ruta, contenido string) {
		t.Helper()
		nombre := filepath.Join(root, ruta)
		if err := os.MkdirAll(filepath.Dir(nombre), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(nombre, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir("es/textos/documento.json", `{"documento.marca_seleccion":"X"}`)
	ruta := ".versiones/es/2.0.0/0123456789/es"
	escribir(ruta+"/textos/documento.json", `{"documento.marca_seleccion":"✓"}`)
	escribir(ruta+"/catalogos/ubicaciones.json", `{"centro.ALHAMA":"Centro de Alhama"}`)
	f := Plantillas{Raiz: root}
	textos, nombres, err := f.Cargar(context.Background(), d.IdiomaPlantilla{Ruta: ruta, Version: "2.0.0"}, p.ReferenciaCatalogo{Codigo: "es"})
	if err != nil || textos["documento.marca_seleccion"] != "✓" || nombres["centro.ALHAMA"] != "Centro de Alhama" {
		t.Fatal("se leyó distribución en vez del paquete activo", err)
	}
	b := `{"centro.ALHAMA":"Centro comarcal de Alhama"}`
	huella := d.Huella([]byte(b))
	publicado := "es/catalogos/.versiones/" + huella + ".json"
	escribir(publicado, b)
	ref := p.ReferenciaCatalogo{Codigo: "es", Contenido: huella, Huella: huella, Ruta: publicado}
	_, nombres, err = f.Cargar(t.Context(), d.IdiomaPlantilla{Ruta: ruta}, ref)
	if err != nil || nombres["centro.ALHAMA"] != "Centro comarcal de Alhama" {
		t.Fatal("referencia de catálogo ignorada", err)
	}
	ref.Huella = d.Huella([]byte("alterado"))
	if _, _, err = f.Cargar(t.Context(), d.IdiomaPlantilla{Ruta: ruta}, ref); err == nil {
		t.Fatal("catálogo alterado aceptado")
	}
	if _, _, err = f.Cargar(t.Context(), d.IdiomaPlantilla{Ruta: "../fuera"}, p.ReferenciaCatalogo{Codigo: "es"}); err == nil {
		t.Fatal("lectura no confinada")
	}
}
