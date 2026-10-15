package ficheros

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type publicacionFija []dominio.Referencia

func (p publicacionFija) Activar(context.Context, dominio.Actor, dominio.VersionInstalada, int64) (dominio.Referencia, error) {
	return dominio.Referencia{}, nil
}
func (p publicacionFija) Referencias(_ context.Context, tipo string) ([]dominio.Referencia, error) {
	if tipo != dominio.TipoIdioma {
		return nil, nil
	}
	return p, nil
}
func (p publicacionFija) Registrada(context.Context, dominio.VersionInstalada) (bool, error) {
	return true, nil
}

func TestTextosDeVersionPublicadaSeCompletanConLaDistribucion(t *testing.T) {
	base := t.TempDir()
	huella := strings.Repeat("a", 64)
	ruta := ".versiones/es/2.0.0/" + huella + "/" + strings.Repeat("b", 32) + "/es"
	escribir := func(relativa, contenido string) {
		t.Helper()
		nombre := filepath.Join(base, filepath.FromSlash(relativa))
		if err := os.MkdirAll(filepath.Dir(nombre), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(nombre, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, raiz := range []string{"es", ruta} {
		for _, d := range []string{"catalogos", "plantillas"} {
			if err := os.MkdirAll(filepath.Join(base, filepath.FromSlash(raiz), d), 0700); err != nil {
				t.Fatal(err)
			}
		}
	}
	escribir("es/idioma.json", `{"codigo":"es","nombre":"Castellano","version":"3.0.0"}`)
	escribir("es/textos/comun.json", `{"comun.titulo":"Distribución","comun.error_carga":"No se pudo cargar","comun.reintentar":"Reintentar","consulta.nueva":"Clave nueva"}`)
	escribir(ruta+"/idioma.json", `{"codigo":"es","nombre":"Castellano","version":"2.0.0"}`)
	escribir(ruta+"/textos/comun.json", `{"comun.titulo":"Publicado","comun.error_carga":"Fallo publicado","comun.reintentar":"Otra vez"}`)
	ref := dominio.Referencia{VersionInstalada: dominio.VersionInstalada{Tipo: dominio.TipoIdioma, Codigo: "es", Version: "2.0.0", Huella: huella, Ruta: ruta}, Revision: 1}
	activos := NuevosActivos(NuevosPaquetes(base, t.TempDir()), publicacionFija{ref})
	textos, err := activos.Textos(context.Background(), "es")
	if err != nil {
		t.Fatal(err)
	}
	if textos["comun.titulo"] != "Publicado" || textos["comun.reintentar"] != "Otra vez" {
		t.Fatal("la distribución sustituyó textos publicados", textos)
	}
	if textos["consulta.nueva"] != "Clave nueva" {
		t.Fatal("falta la clave nueva de la distribución", textos)
	}
	// Una clave movida de fichero en la distribución no se duplica.
	escribir("es/textos/consulta.json", `{"consulta.movida":"Movida"}`)
	escribir(ruta+"/textos/consulta.json", `{"consulta.anterior":"Anterior"}`)
	if textos, err = activos.Textos(context.Background(), "es"); err != nil || textos["consulta.movida"] != "Movida" || textos["consulta.anterior"] != "Anterior" {
		t.Fatal("clave movida de fichero", textos, err)
	}
	// Claves repetidas en la distribución siguen siendo un error.
	escribir("es/textos/repetida.json", `{"consulta.nueva":"Otra"}`)
	if _, err = activos.Textos(context.Background(), "es"); err == nil {
		t.Fatal("clave repetida en la distribución aceptada")
	}
	if err := os.Remove(filepath.Join(base, "es", "textos", "repetida.json")); err != nil {
		t.Fatal(err)
	}
	// Sin distribución legible, la versión publicada sigue sirviéndose sola.
	if err := os.RemoveAll(filepath.Join(base, "es", "textos")); err != nil {
		t.Fatal(err)
	}
	textos, err = activos.Textos(context.Background(), "es")
	if err != nil || textos["comun.titulo"] != "Publicado" || textos["consulta.nueva"] != "" {
		t.Fatal("versión publicada no disponible sin distribución", textos, err)
	}
}
