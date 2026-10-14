package ficheros

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
)

// AlmacenVersiones escribe rutas exclusivas: una limpieza nunca borra la
// versión de otro instalador, aunque ambos suban exactamente el mismo ZIP.
type AlmacenVersiones struct{ idiomas, temas string }

func NuevoAlmacenVersiones(idiomas, temas string) *AlmacenVersiones {
	return &AlmacenVersiones{idiomas: idiomas, temas: temas}
}

func (a *AlmacenVersiones) base(tipo string) string {
	if tipo == dominio.TipoIdioma {
		return a.idiomas
	}
	return a.temas
}

var nonceVersion = regexp.MustCompile(`^[0-9a-f]{32}$`)

func rutaVersionValida(v dominio.VersionInstalada) bool {
	p := strings.Split(v.Ruta, "/")
	return v.Valida() && len(p) == 6 && p[0] == ".versiones" && p[1] == v.Codigo && p[2] == v.Version && p[3] == v.Huella && nonceVersion.MatchString(p[4]) && p[5] == v.Codigo
}

func (a *AlmacenVersiones) Guardar(ctx context.Context, c dominio.ContenidoInstalacion) (v dominio.VersionInstalada, err error) {
	v = dominio.VersionInstalada{Tipo: c.Tipo, Codigo: c.Codigo, Version: c.Version, Huella: c.Huella}
	if !v.Valida() {
		return v, dominio.ErrPaqueteInvalido
	}
	var aleatorio [16]byte
	if _, err = rand.Read(aleatorio[:]); err != nil {
		return v, err
	}
	v.Ruta = path.Join(".versiones", v.Codigo, v.Version, v.Huella, hex.EncodeToString(aleatorio[:]), v.Codigo)
	raiz, err := os.OpenRoot(a.base(v.Tipo))
	if err != nil {
		return v, dominio.ErrNoDisponible
	}
	defer raiz.Close()
	if err = raiz.MkdirAll(path.Dir(v.Ruta), 0750); err != nil {
		return v, dominio.ErrNoDisponible
	}
	if err = raiz.Mkdir(v.Ruta, 0750); err != nil {
		return v, dominio.ErrNoDisponible
	}
	defer func() {
		if err != nil {
			if limpiar := a.Eliminar(context.WithoutCancel(ctx), v); limpiar != nil {
				err = dominio.ErrNoDisponible
			}
		}
	}()
	carpeta, err := raiz.OpenRoot(v.Ruta)
	if err != nil {
		return v, dominio.ErrNoDisponible
	}
	defer carpeta.Close()
	if c.Tipo == dominio.TipoIdioma {
		for _, d := range []string{"textos", "catalogos", "plantillas"} {
			if err = carpeta.MkdirAll(d, 0750); err != nil {
				return v, dominio.ErrNoDisponible
			}
		}
	}
	for nombre, datos := range c.Archivos {
		if err = ctx.Err(); err != nil {
			return v, err
		}
		if !rutaZIPValida(strings.TrimSuffix(nombre, "/")) {
			return v, dominio.ErrPaqueteInvalido
		}
		if strings.HasSuffix(nombre, "/") {
			if err = carpeta.MkdirAll(strings.TrimSuffix(nombre, "/"), 0750); err != nil {
				return v, dominio.ErrNoDisponible
			}
			continue
		}
		if err = carpeta.MkdirAll(path.Dir(nombre), 0750); err != nil {
			return v, dominio.ErrNoDisponible
		}
		f, fallo := carpeta.OpenFile(nombre, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0440)
		if fallo != nil {
			return v, dominio.ErrNoDisponible
		}
		_, fallo = f.Write(datos)
		if fallo == nil {
			fallo = f.Sync()
		}
		cierre := f.Close()
		if fallo != nil || cierre != nil {
			return v, dominio.ErrNoDisponible
		}
		lectura, fallo := carpeta.Open(nombre)
		if fallo != nil {
			return v, dominio.ErrNoDisponible
		}
		b, fallo := io.ReadAll(io.LimitReader(lectura, int64(len(datos))+1))
		cierre = lectura.Close()
		if fallo != nil || cierre != nil || !bytes.Equal(datos, b) {
			return v, dominio.ErrNoDisponible
		}
	}
	// Sincroniza y sella todos los directorios después de comprobar los bytes.
	err = fs.WalkDir(carpeta.FS(), ".", func(nombre string, d fs.DirEntry, fallo error) error {
		if fallo != nil {
			return fallo
		}
		if !d.IsDir() {
			return nil
		}
		f, fallo := carpeta.Open(nombre)
		if fallo != nil {
			return fallo
		}
		fallo = f.Sync()
		cierre := f.Close()
		if fallo != nil {
			return fallo
		}
		if cierre != nil {
			return cierre
		}
		return carpeta.Chmod(nombre, 0550)
	})
	if err != nil {
		return v, dominio.ErrNoDisponible
	}
	// Los padres también deben llegar al volumen antes de confirmar PostgreSQL.
	for dir := path.Dir(v.Ruta); ; dir = path.Dir(dir) {
		f, fallo := raiz.Open(dir)
		if fallo != nil {
			return v, dominio.ErrNoDisponible
		}
		fallo = f.Sync()
		cierre := f.Close()
		if fallo != nil || cierre != nil {
			return v, dominio.ErrNoDisponible
		}
		if dir == "." {
			break
		}
	}
	return v, nil
}

func (a *AlmacenVersiones) Eliminar(ctx context.Context, v dominio.VersionInstalada) error {
	if !rutaVersionValida(v) {
		return dominio.ErrPaqueteInvalido
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	raiz, err := os.OpenRoot(a.base(v.Tipo))
	if err != nil {
		return err
	}
	defer raiz.Close()
	carpeta, err := raiz.OpenRoot(v.Ruta)
	if os.IsNotExist(err) {
		return raiz.RemoveAll(path.Dir(v.Ruta))
	}
	if err != nil {
		return err
	}
	err = fs.WalkDir(carpeta.FS(), ".", func(nombre string, d fs.DirEntry, fallo error) error {
		if fallo != nil {
			return fallo
		}
		if d.Type()&os.ModeSymlink != 0 {
			return dominio.ErrPaqueteInvalido
		}
		if d.IsDir() {
			return carpeta.Chmod(nombre, 0750)
		}
		return nil
	})
	carpeta.Close()
	if err != nil {
		return err
	}
	return raiz.RemoveAll(path.Dir(v.Ruta))
}

func (a *AlmacenVersiones) Huerfanas(ctx context.Context, antes time.Time) ([]dominio.VersionInstalada, error) {
	versiones := []dominio.VersionInstalada{}
	for _, tipo := range []string{dominio.TipoIdioma, dominio.TipoTema} {
		raiz, err := os.OpenRoot(a.base(tipo))
		if err != nil {
			return nil, err
		}
		err = fs.WalkDir(raiz.FS(), ".versiones", func(nombre string, d fs.DirEntry, err error) error {
			if os.IsNotExist(err) && nombre == ".versiones" {
				return nil
			}
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 || !d.IsDir() {
				return dominio.ErrPaqueteInvalido
			}
			p := strings.Split(nombre, "/")
			if len(p) == 5 {
				v := dominio.VersionInstalada{Tipo: tipo, Codigo: p[1], Version: p[2], Huella: p[3], Ruta: path.Join(nombre, p[1])}
				if !rutaVersionValida(v) {
					return dominio.ErrPaqueteInvalido
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				if info.ModTime().Before(antes) {
					versiones = append(versiones, v)
				}
				return fs.SkipDir
			}
			return nil
		})
		raiz.Close()
		if err != nil {
			return nil, err
		}
	}
	return versiones, nil
}

var _ salida.AlmacenVersiones = (*AlmacenVersiones)(nil)
