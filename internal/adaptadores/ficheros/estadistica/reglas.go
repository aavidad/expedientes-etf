package estadistica

import (
	"context"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	"io"
	"os"
	"strconv"
	"time"
)

// Reglas conserva las versiones numéricas de distribución, como el catálogo
// de familia. Cada cambio publica otro fichero; no sustituye el anterior.
type Reglas struct {
	Raiz string
	SMI  salida.SMIMemoria
}

var _ salida.ReglasMemoria = Reglas{}

type versionReglas struct {
	Version    string       `json:"version"`
	Desde      time.Time    `json:"vigente_desde"`
	Registrado time.Time    `json:"registrado_en"`
	Parametros d.Parametros `json:"parametros"`
}

func (r Reglas) ReglasMemoria(ctx context.Context, anio int, conocido time.Time) (d.Historia[d.Parametros], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raiz, err := os.OpenRoot(r.Raiz)
	if err != nil {
		return nil, d.ErrReglas
	}
	defer raiz.Close()
	f, err := raiz.Open("indice.json")
	if err != nil {
		return nil, d.ErrReglas
	}
	defer f.Close()
	var archivos []string
	decoder := json.NewDecoder(io.LimitReader(f, 16385))
	if decoder.Decode(&archivos) != nil || decoder.Decode(new(any)) != io.EOF || len(archivos) > 100 {
		return nil, d.ErrReglas
	}
	historia := d.Historia[d.Parametros]{}
	ids := map[string]bool{}
	for _, archivo := range archivos {
		// Sólo nombres de versiones del catálogo, sin rutas ni extensiones libres.
		if len(archivo) > 64 || archivo == "" {
			return nil, d.ErrReglas
		}
		for _, c := range archivo {
			if !(c >= '0' && c <= '9' || c == '.' || c == 'v') {
				return nil, d.ErrReglas
			}
		}
		file, err := raiz.Open(archivo + ".json")
		if err != nil {
			return nil, d.ErrReglas
		}
		stat, err := file.Stat()
		if err != nil || !stat.Mode().IsRegular() || stat.Size() > 1<<20 {
			file.Close()
			return nil, d.ErrReglas
		}
		var v versionReglas
		dec := json.NewDecoder(io.LimitReader(file, (1<<20)+1))
		dec.DisallowUnknownFields()
		err = dec.Decode(&v)
		fin := dec.Decode(new(any))
		file.Close()
		if err != nil || fin != io.EOF || v.Version == "" || ids[v.Version] || v.Desde.IsZero() || v.Registrado.IsZero() {
			return nil, d.ErrReglas
		}
		ids[v.Version] = true
		// Se reconstruye la vigencia con las versiones conocidas, conservando las
		// publicaciones antiguas cuando se reproduce una emisión ya confirmada.
		if v.Registrado.After(conocido) {
			continue
		}
		historia = append(historia, d.Revision[d.Parametros]{ID: v.Version, Vigencia: d.Vigencia{Desde: v.Desde}, RegistradoEn: v.Registrado, Dato: d.Dato[d.Parametros]{Conocido: true, Valor: v.Parametros}})
	}
	// El orden de índice es causal, con vigencias estrictamente crecientes.
	for i := 1; i < len(historia); i++ {
		if !historia[i].Vigencia.Desde.After(historia[i-1].Vigencia.Desde) {
			return nil, d.ErrReglas
		}
		historia[i-1].Vigencia.Hasta = historia[i].Vigencia.Desde
	}
	if r.SMI != nil {
		smi, err := r.SMI.SMI(ctx)
		if err != nil {
			return nil, err
		}
		encontrado := false
		for _, s := range smi {
			if s.Anio != anio {
				continue
			}
			encontrado = true
			for i := range historia {
				historia[i].ID += ":" + strconv.Itoa(s.Version)
				if s.ImporteAnualCentimos != nil {
					historia[i].Dato.Valor.SMIAnualCentimos = *s.ImporteAnualCentimos
				} else {
					historia[i].Dato.Valor.TramosSMI = nil
				}
			}
		}
		if !encontrado {
			for i := range historia {
				historia[i].Dato.Valor.TramosSMI = nil
			}
		}
	} else {
		for i := range historia {
			historia[i].Dato.Valor.TramosSMI = nil
		}
	}
	return historia, nil
}
