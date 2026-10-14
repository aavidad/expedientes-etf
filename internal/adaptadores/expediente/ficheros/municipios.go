package ficheros

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"regexp"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

type Municipios struct{ ruta, idioma string }

var _ salida.NombresMunicipios = (*Municipios)(nil)

func NuevosMunicipios(ruta, idioma string) *Municipios { return &Municipios{ruta, idioma} }

func (m *Municipios) Nombres(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`).MatchString(m.idioma) {
		return nil, dominio.ErrNoDisponible
	}
	raiz, err := os.OpenRoot(m.ruta)
	if err != nil {
		return nil, dominio.ErrNoDisponible
	}
	defer raiz.Close()
	nombres := make(map[string]string)
	for _, archivo := range []string{"ubicaciones.json", "variantes_municipios.json"} {
		parte, err := leerNombres(raiz, m.idioma+"/catalogos/"+archivo)
		if err != nil {
			return nil, err
		}
		for clave, nombre := range parte {
			if _, repetida := nombres[clave]; repetida {
				return nil, dominio.ErrNoDisponible
			}
			nombres[clave] = nombre
		}
	}
	return nombres, ctx.Err()
}

func leerNombres(raiz *os.Root, archivo string) (map[string]string, error) {
	f, err := raiz.Open(archivo)
	if err != nil {
		return nil, dominio.ErrNoDisponible
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, dominio.ErrNoDisponible
	}
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return nil, dominio.ErrNoDisponible
	}
	var parte map[string]string
	if json.Unmarshal(b, &parte) != nil || len(parte) == 0 {
		return nil, dominio.ErrNoDisponible
	}
	return parte, nil
}

// NombresEn carga etiquetas de catálogo sin ampliar permisos de Administración.
func (m *Municipios) NombresEn(ctx context.Context, idioma string) (map[string]string, error) {
	n := NuevosMunicipios(m.ruta, idioma)
	nombres, err := n.Nombres(ctx)
	if err != nil {
		return nil, err
	}
	raiz, err := os.OpenRoot(m.ruta)
	if err != nil {
		return nil, dominio.ErrNoDisponible
	}
	defer raiz.Close()
	listas, err := leerNombres(raiz, idioma+"/catalogos/listas.json")
	if err != nil {
		return nil, err
	}
	for k, v := range listas {
		nombres[k] = v
	}
	return nombres, nil
}
