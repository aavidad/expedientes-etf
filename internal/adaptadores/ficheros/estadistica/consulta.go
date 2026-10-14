package estadistica

import (
	"context"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
)

type TextosCifras struct{ Raiz string }

func (s TextosCifras) LeerTextosCifras(ctx context.Context, ruta string) (map[string]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	idioma := path.Base(ruta)
	if !fs.ValidPath(ruta) || !regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`).MatchString(idioma) || (ruta != idioma && !strings.HasPrefix(ruta, ".versiones/"+idioma+"/")) {
		return nil, d.ErrEntrada
	}
	raiz, err := os.OpenRoot(s.Raiz)
	if err != nil {
		return nil, d.ErrDisponibleMemoria
	}
	defer raiz.Close()
	f, err := raiz.Open(ruta + "/textos/consulta_memoria.json")
	if err != nil {
		return nil, d.ErrDisponibleMemoria
	}
	defer f.Close()
	var r map[string]string
	dec := json.NewDecoder(io.LimitReader(f, 16385))
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF {
		return nil, d.ErrDisponibleMemoria
	}
	for _, k := range []string{"estadistica.consulta_fecha", "estadistica.consulta_provisional", "estadistica.consulta_definitiva"} {
		if r[k] == "" {
			return nil, d.ErrDisponibleMemoria
		}
	}
	return r, nil
}
