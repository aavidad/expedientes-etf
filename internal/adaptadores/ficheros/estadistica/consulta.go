package estadistica

import (
	"context"
	"encoding/json"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	paquete "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
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
	var r map[string]string
	for i, capa := range paquete.CapasTextos(ruta) {
		parte, err := leerConsulta(raiz, capa)
		if i > 0 && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, d.ErrDisponibleMemoria
		}
		if r == nil {
			r = parte
		} else {
			paquete.CompletarTextos(r, parte)
		}
	}
	for _, k := range []string{"estadistica.consulta_fecha", "estadistica.consulta_provisional", "estadistica.consulta_definitiva"} {
		if r[k] == "" {
			return nil, d.ErrDisponibleMemoria
		}
	}
	return r, nil
}

func leerConsulta(raiz *os.Root, carpeta string) (map[string]string, error) {
	f, err := raiz.Open(carpeta + "/textos/consulta_memoria.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r map[string]string
	dec := json.NewDecoder(io.LimitReader(f, 16385))
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF || r == nil {
		return nil, d.ErrDisponibleMemoria
	}
	return r, nil
}
