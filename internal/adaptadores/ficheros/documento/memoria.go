package documento

import (
	"context"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

func (f Plantillas) LeerMemoria(ctx context.Context, idioma d.IdiomaPlantilla, p d.Plantilla) (d.PresentacionMemoria, error) {
	if err := ctx.Err(); err != nil {
		return d.PresentacionMemoria{}, err
	}
	if p.Tipo != "memoria" || !codigoIdioma.MatchString(p.Idioma) || !version.MatchString(p.Version) || !fs.ValidPath(idioma.Ruta) || (idioma.Ruta != p.Idioma && (!strings.HasPrefix(idioma.Ruta, ".versiones/"+p.Idioma+"/") || !strings.HasSuffix(idioma.Ruta, "/"+p.Idioma))) {
		return d.PresentacionMemoria{}, d.ErrPlantilla
	}
	raiz, err := os.OpenRoot(f.Raiz)
	if err != nil {
		return d.PresentacionMemoria{}, d.ErrPlantilla
	}
	defer raiz.Close()
	file, err := raiz.Open(path.Join(idioma.Ruta, "plantillas", "memoria", "v"+p.Version+".json"))
	if err != nil {
		return d.PresentacionMemoria{}, d.ErrPlantilla
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > 1<<20 {
		return d.PresentacionMemoria{}, d.ErrPlantilla
	}
	var textos d.PresentacionMemoria
	decoder := json.NewDecoder(io.LimitReader(file, (1<<20)+1))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&textos) != nil || decoder.Decode(new(any)) != io.EOF {
		return d.PresentacionMemoria{}, d.ErrPlantilla
	}
	return textos, nil
}
