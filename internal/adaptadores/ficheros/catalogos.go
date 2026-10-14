package ficheros

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"time"
)

var claveCatalogo = regexp.MustCompile(`^[a-z][a-z0-9_.]*\.[A-Za-z0-9_-]{1,80}$`)
var huellaCatalogo = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validarCatalogo(m map[string]string) error {
	if m == nil || len(m) > 10000 {
		return dominio.ErrPublicacion
	}
	for k, v := range m {
		if !claveCatalogo.MatchString(k) || strings.TrimSpace(v) == "" || len(v) > 2000 {
			return dominio.ErrPublicacion
		}
	}
	return nil
}
func (p *Paquetes) LeerCatalogo(ctx context.Context, r dominio.ReferenciaCatalogo) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := abrirDirectorio(p.idiomas, r.Codigo, "/catalogos")
	if err != nil {
		return nil, dominio.ErrPublicacion
	}
	defer root.Close()
	m := map[string]string{}
	if r.Contenido == "" {
		d, err := root.Open(".")
		if err != nil {
			return nil, dominio.ErrPublicacion
		}
		defer d.Close()
		entradas, err := d.ReadDir(-1)
		if err != nil || len(entradas) > 128 {
			return nil, dominio.ErrPublicacion
		}
		for _, e := range entradas {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			var parte map[string]string
			if err := leerJSON(root, e.Name(), &parte); err != nil {
				return nil, dominio.ErrPublicacion
			}
			for k, v := range parte {
				if _, ok := m[k]; ok {
					return nil, dominio.ErrPublicacion
				}
				m[k] = v
			}
		}
	} else {
		if !huellaCatalogo.MatchString(r.Huella) || r.Contenido != r.Huella || r.Ruta != r.Codigo+"/catalogos/.versiones/"+r.Huella+".json" {
			return nil, dominio.ErrPublicacion
		}
		nombre := ".versiones/" + r.Huella + ".json"
		info, err := root.Lstat(nombre)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return nil, dominio.ErrPublicacion
		}
		b, err := root.ReadFile(nombre)
		if err != nil {
			return nil, dominio.ErrPublicacion
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != r.Huella {
			return nil, dominio.ErrPublicacion
		}
		if err := decodificarJSON(b, &m); err != nil {
			return nil, dominio.ErrPublicacion
		}
	}
	if err := validarCatalogo(m); err != nil {
		return nil, err
	}
	return m, nil
}
func (p *Paquetes) EscribirCatalogo(ctx context.Context, codigo string, m map[string]string) (dominio.ReferenciaCatalogo, error) {
	if err := ctx.Err(); err != nil {
		return dominio.ReferenciaCatalogo{}, err
	}
	if err := validarCatalogo(m); err != nil {
		return dominio.ReferenciaCatalogo{}, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > 1<<20 {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	r := dominio.ReferenciaCatalogo{Codigo: codigo, Contenido: huella, Huella: huella, Ruta: codigo + "/catalogos/.versiones/" + huella + ".json"}
	root, err := abrirDirectorio(p.idiomas, codigo, "/catalogos")
	if err != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	defer root.Close()
	if err := root.Mkdir(".versiones", 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	info, err := root.Lstat(".versiones")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	temporal := ".versiones/." + hex.EncodeToString(aleatorio)
	f, err := root.OpenFile(temporal, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	defer root.Remove(temporal)
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	cierre := f.Close()
	if err != nil || cierre != nil {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	// Enlace exclusivo: la ruta definitiva aparece completa y nunca se sustituye.
	if err := root.Link(temporal, ".versiones/"+huella+".json"); err != nil && !errors.Is(err, fs.ErrExist) {
		return dominio.ReferenciaCatalogo{}, dominio.ErrPublicacion
	}
	if _, err := p.LeerCatalogo(ctx, r); err != nil {
		return dominio.ReferenciaCatalogo{}, err
	}
	return r, nil
}

func (p *Paquetes) LimpiarCatalogo(ctx context.Context, codigo string, conservadas map[string]bool, antes time.Time) error {
	root, err := abrirDirectorio(p.idiomas, codigo, "/catalogos")
	if err != nil {
		return dominio.ErrPublicacion
	}
	defer root.Close()
	versiones, err := root.OpenRoot(".versiones")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return dominio.ErrPublicacion
	}
	defer versiones.Close()
	dir, err := versiones.Open(".")
	if err != nil {
		return dominio.ErrPublicacion
	}
	defer dir.Close()
	entradas, err := dir.ReadDir(-1)
	if err != nil || len(entradas) > 4096 {
		return dominio.ErrPublicacion
	}
	for _, e := range entradas {
		if err := ctx.Err(); err != nil {
			return err
		}
		nombre := e.Name()
		huella := strings.TrimSuffix(nombre, ".json")
		if !strings.HasSuffix(nombre, ".json") || !huellaCatalogo.MatchString(huella) || conservadas[huella] {
			continue
		}
		info, err := versiones.Lstat(nombre)
		if err != nil {
			return dominio.ErrPublicacion
		}
		if info.Mode().IsRegular() && info.ModTime().Before(antes) {
			if err := versiones.Remove(nombre); err != nil {
				return dominio.ErrPublicacion
			}
		}
	}
	return nil
}
