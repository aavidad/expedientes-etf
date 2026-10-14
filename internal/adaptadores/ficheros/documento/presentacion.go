package documento

import (
	"context"
	"encoding/json"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
)

func (f Plantillas) ResolverCatalogo(ctx context.Context, codigo string) (p.ReferenciaCatalogo, error) {
	if !codigoIdioma.MatchString(codigo) || f.Paquetes == nil {
		return p.ReferenciaCatalogo{}, d.ErrNoDisponible
	}
	refs, err := f.Paquetes.Referencias(ctx, "catalogo")
	if err != nil {
		return p.ReferenciaCatalogo{}, d.ErrNoDisponible
	}
	for _, ref := range refs {
		if ref.Codigo == codigo {
			return p.ReferenciaCatalogo{Codigo: codigo, Version: int(ref.Revision), Contenido: ref.Version, Ruta: ref.Ruta, Huella: ref.Huella}, nil
		}
	}
	return p.ReferenciaCatalogo{Codigo: codigo}, nil
}

func (f Plantillas) Cargar(ctx context.Context, idioma d.IdiomaPlantilla, ref p.ReferenciaCatalogo) (map[string]string, map[string]string, error) {
	if ctx.Err() != nil || !codigoIdioma.MatchString(ref.Codigo) || !fs.ValidPath(idioma.Ruta) || (idioma.Ruta != ref.Codigo && (!strings.HasPrefix(idioma.Ruta, ".versiones/"+ref.Codigo+"/") || !strings.HasSuffix(idioma.Ruta, "/"+ref.Codigo))) {
		return nil, nil, d.ErrNoDisponible
	}
	root, err := os.OpenRoot(f.Raiz)
	if err != nil {
		return nil, nil, d.ErrNoDisponible
	}
	defer root.Close()
	leer := func(nombre string) ([]byte, error) {
		archivo, err := root.Open(nombre)
		if err != nil {
			return nil, d.ErrNoDisponible
		}
		defer archivo.Close()
		info, err := archivo.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return nil, d.ErrNoDisponible
		}
		b, err := io.ReadAll(io.LimitReader(archivo, (1<<20)+1))
		if err != nil || len(b) > 1<<20 {
			return nil, d.ErrNoDisponible
		}
		return b, nil
	}
	coleccion := func(carpeta string) (map[string]string, error) {
		dir, err := root.Open(carpeta)
		if err != nil {
			return nil, d.ErrNoDisponible
		}
		defer dir.Close()
		archivos, err := dir.ReadDir(129)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, d.ErrNoDisponible
		}
		if len(archivos) > 128 {
			return nil, d.ErrNoDisponible
		}
		m := map[string]string{}
		total := 0
		for _, archivo := range archivos {
			if !strings.HasSuffix(archivo.Name(), ".json") {
				continue
			}
			b, err := leer(path.Join(carpeta, archivo.Name()))
			if err != nil {
				return nil, err
			}
			total += len(b)
			if total > 4<<20 {
				return nil, d.ErrNoDisponible
			}
			var parte map[string]string
			if json.Unmarshal(b, &parte) != nil {
				return nil, d.ErrNoDisponible
			}
			for k, v := range parte {
				if _, ok := m[k]; ok {
					return nil, d.ErrNoDisponible
				}
				m[k] = v
			}
		}
		return m, nil
	}
	textos, err := coleccion(path.Join(idioma.Ruta, "textos"))
	if err != nil {
		return nil, nil, err
	}
	var nombres map[string]string
	if ref.Contenido == "" {
		nombres, err = coleccion(path.Join(idioma.Ruta, "catalogos"))
	} else {
		esperado := path.Join(ref.Codigo, "catalogos", ".versiones", ref.Contenido+".json")
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(ref.Contenido) || ref.Ruta != esperado {
			return nil, nil, d.ErrNoDisponible
		}
		var b []byte
		b, err = leer(esperado)
		if err == nil {
			if d.Huella(b) != ref.Huella || json.Unmarshal(b, &nombres) != nil {
				err = d.ErrNoDisponible
			}
		}
	}
	if err != nil {
		return nil, nil, d.ErrNoDisponible
	}
	return textos, nombres, nil
}
