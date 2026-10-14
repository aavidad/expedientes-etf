package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"os"
	"path"
)

func (f Plantillas) Guardar(ctx context.Context, p d.Plantilla, b []byte) error {
	if !d.PlantillaPublicada(p) || len(b) > d.MaximoPlantilla || len(b) == 0 || d.Huella(b) != p.Huella {
		return d.ErrPlantilla
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	raiz, err := os.OpenRoot(f.Raiz)
	if err != nil {
		return d.ErrNoDisponible
	}
	defer raiz.Close()
	ruta := path.Join(p.Idioma, p.Ruta)
	if err := raiz.MkdirAll(path.Dir(ruta), 0700); err != nil {
		return d.ErrNoDisponible
	}
	archivo, err := raiz.OpenFile(ruta, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
	if err != nil {
		return d.ErrNoDisponible
	}
	ok := false
	defer func() {
		_ = archivo.Close()
		if !ok {
			_ = raiz.Remove(ruta)
		}
	}()
	if _, err := archivo.Write(b); err != nil {
		return d.ErrNoDisponible
	}
	if err := archivo.Sync(); err != nil {
		return d.ErrNoDisponible
	}
	if err := archivo.Close(); err != nil {
		return d.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Sincroniza cada directorio creado, incluida su entrada en el padre.
	for padre := path.Dir(ruta); ; padre = path.Dir(padre) {
		dir, err := raiz.Open(padre)
		if err != nil {
			return d.ErrNoDisponible
		}
		err = dir.Sync()
		cerrado := dir.Close()
		if err != nil || cerrado != nil {
			return d.ErrNoDisponible
		}
		if padre == "." {
			break
		}
	}
	ok = true
	return nil
}
func (f Plantillas) Eliminar(ctx context.Context, p d.Plantilla) error {
	if !d.PlantillaPublicada(p) {
		return d.ErrPlantilla
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	raiz, err := os.OpenRoot(f.Raiz)
	if err != nil {
		return d.ErrNoDisponible
	}
	defer raiz.Close()
	if err := raiz.Remove(path.Join(p.Idioma, p.Ruta)); err != nil && !os.IsNotExist(err) {
		return d.ErrNoDisponible
	}
	return nil
}
