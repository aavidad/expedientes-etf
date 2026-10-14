package ficheros

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"

	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
)

// Paquetes limita las lecturas a raíces independientes, nunca al repositorio.
type Paquetes struct {
	idiomas, temas string
	reserva        fs.FS
}

var _ salida.Textos = (*Paquetes)(nil)
var _ salida.Temas = (*Paquetes)(nil)

func NuevosPaquetes(idiomas, temas string) *Paquetes {
	return NuevosPaquetesConReserva(idiomas, temas, nil)
}

var identificador = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var claveTexto = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)

func abrirDirectorio(base, nombre, sufijo string) (*os.Root, error) {
	if !identificador.MatchString(nombre) {
		return nil, fs.ErrInvalid
	}
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	return raiz.OpenRoot(nombre + sufijo)
}

func leerJSON(raiz *os.Root, nombre string, destino any) error {
	info, err := raiz.Lstat(nombre)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return fs.ErrInvalid
	}
	f, err := raiz.Open(nombre)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(b) > 1<<20 {
		return fs.ErrInvalid
	}
	return decodificarJSON(b, destino)
}

func (p *Paquetes) Textos(ctx context.Context, codigo string) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	idioma, err := p.leerIdioma(codigo)
	if err != nil {
		return nil, err
	}
	_ = idioma
	raiz, err := abrirDirectorio(p.idiomas, codigo, "/textos")
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	directorio, err := raiz.Open(".")
	if err != nil {
		return nil, err
	}
	defer directorio.Close()
	entradas, err := directorio.ReadDir(65)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entradas) > 64 {
		return nil, fs.ErrInvalid
	}
	textos := make(map[string]string)
	tamano := 0
	for _, entrada := range entradas {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(entrada.Name(), ".json") {
			continue
		}
		var parte map[string]string
		if err := leerJSON(raiz, entrada.Name(), &parte); err != nil {
			return nil, err
		}
		if parte == nil {
			return nil, fs.ErrInvalid
		}
		for clave, valor := range parte {
			if !claveTexto.MatchString(clave) {
				return nil, fs.ErrInvalid
			}
			if _, existe := textos[clave]; existe {
				return nil, fs.ErrInvalid
			}
			tamano += len(clave) + len(valor)
			if tamano > 2<<20 {
				return nil, fs.ErrInvalid
			}
			textos[clave] = valor
		}
	}
	if codigo == dominio.IdiomaPredeterminado {
		for _, clave := range []string{"comun.titulo", "comun.error_carga", "comun.reintentar"} {
			if strings.TrimSpace(textos[clave]) == "" {
				return nil, fs.ErrInvalid
			}
		}
	}
	return textos, nil
}

// Cada consulta relee el disco: una sustitución de directorio se observa sin reinicio.
func (p *Paquetes) leerIdioma(codigo string) (dominio.Idioma, error) {
	var idioma dominio.Idioma
	if !dominio.CodigoIdiomaValido(codigo) {
		return idioma, fs.ErrInvalid
	}
	raiz, err := abrirDirectorio(p.idiomas, codigo, "")
	if err != nil {
		return idioma, err
	}
	defer raiz.Close()
	if err := leerJSON(raiz, "idioma.json", &idioma); err != nil {
		return idioma, err
	}
	if !idioma.Valido() || idioma.Codigo != codigo {
		return idioma, fs.ErrInvalid
	}
	for _, nombre := range []string{"textos", "catalogos", "plantillas"} {
		info, err := raiz.Lstat(nombre)
		if err != nil {
			return idioma, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return idioma, fs.ErrInvalid
		}
	}
	return idioma, nil
}

func listarDirectorios(base string) ([]string, error) {
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	dir, err := raiz.Open(".")
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entradas, err := dir.ReadDir(129)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entradas) > 128 {
		return nil, fs.ErrInvalid
	}
	nombres := []string{}
	for _, entrada := range entradas {
		if entrada.IsDir() {
			nombres = append(nombres, entrada.Name())
		}
	}
	sort.Strings(nombres)
	return nombres, nil
}

func (p *Paquetes) Idiomas(ctx context.Context) ([]dominio.Idioma, error) {
	nombres, err := listarDirectorios(p.idiomas)
	if err != nil {
		return nil, err
	}
	idiomas := []dominio.Idioma{}
	for _, nombre := range nombres {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		idioma, err := p.leerIdioma(nombre)
		if err != nil {
			continue
		}
		if _, err := p.Textos(ctx, nombre); err == nil {
			idioma.Predeterminado = idioma.Codigo == dominio.IdiomaPredeterminado
			idiomas = append(idiomas, idioma)
		}
	}
	return idiomas, nil
}

// Rechaza claves repetidas incluso dentro del manifiesto: evita interpretaciones distintas.
func decodificarJSON(b []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	if err := objetoJSON(d); err != nil {
		return err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return fs.ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(destino)
}
func objetoJSON(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimitador, esDelimitador := token.(json.Delim)
	if !esDelimitador {
		return nil
	}
	if delimitador == '{' {
		vistas := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok := token.(string)
			if !ok || vistas[clave] {
				return fs.ErrInvalid
			}
			vistas[clave] = true
			if err := objetoJSON(d); err != nil {
				return err
			}
		}
	} else if delimitador == '[' {
		for d.More() {
			if err := objetoJSON(d); err != nil {
				return err
			}
		}
	} else {
		return fs.ErrInvalid
	}
	_, err = d.Token()
	return err
}

// La copia del predeterminado se inyecta desde composición; el adaptador no importa recursos.
func NuevosPaquetesConReserva(idiomas, temas string, reserva fs.FS) *Paquetes {
	return &Paquetes{idiomas: idiomas, temas: temas, reserva: reserva}
}
