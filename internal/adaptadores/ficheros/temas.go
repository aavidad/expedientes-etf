package ficheros

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

func tokenValido(clave, valor string) bool { return dominio.TokenValido(clave, valor) }

func (p *Paquetes) leerTemaDisco(ctx context.Context, nombre string) (dominio.Tema, map[string]string, error) {
	var tema dominio.Tema
	if !dominio.NombreTemaValido(nombre) {
		return tema, nil, fs.ErrInvalid
	}
	raiz, err := abrirDirectorio(p.temas, nombre, "")
	if err != nil {
		return tema, nil, err
	}
	defer raiz.Close()
	if err := leerJSON(raiz, "tema.json", &tema); err != nil {
		return tema, nil, err
	}
	if !tema.Valido() || tema.Identificador != nombre {
		return tema, nil, fs.ErrInvalid
	}
	var tokens map[string]string
	if err := leerJSON(raiz, "tokens.json", &tokens); err != nil {
		return tema, nil, err
	}
	if len(tokens) == 0 || len(tokens) > len(dominio.TokensRequeridos()) {
		return tema, nil, fs.ErrInvalid
	}
	for clave, valor := range tokens {
		if !tokenValido(clave, valor) {
			return tema, nil, fs.ErrInvalid
		}
	}
	// Recorre únicamente el paquete abierto: no permite enlaces ni ficheros activos.
	entradas, total := 0, int64(0)
	err = fs.WalkDir(raiz.FS(), ".", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entradas++
		if entradas > 1000 {
			return fs.ErrInvalid
		}
		if ruta == "." {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fs.ErrInvalid
		}
		if d.IsDir() {
			if ruta != "fuentes" && ruta != "iconos" {
				return fs.ErrInvalid
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fs.ErrInvalid
		}
		total += info.Size()
		if total > 20<<20 {
			return fs.ErrInvalid
		}
		if ruta == "tema.json" || ruta == "tokens.json" {
			return nil
		}
		_, err = leerRecurso(raiz, ruta)
		return err
	})
	if err != nil {
		return tema, nil, err
	}
	for _, ruta := range tema.Fuentes {
		if _, err := leerRecurso(raiz, ruta); err != nil {
			return tema, nil, err
		}
	}
	return tema, tokens, nil
}

func (p *Paquetes) leerTema(ctx context.Context, nombre string) (dominio.Tema, map[string]string, error) {
	tema, tokens, err := p.leerTemaDisco(ctx, nombre)
	if err == nil && (nombre != dominio.TemaPredeterminado || dominio.TokensValidos(tokens, tema.Tipo)) {
		return tema, tokens, nil
	}
	if nombre != dominio.TemaPredeterminado || p.reserva == nil || ctx.Err() != nil {
		if err == nil {
			err = fs.ErrInvalid
		}
		return tema, nil, err
	}
	var copia dominio.Tema
	b, errorLectura := fs.ReadFile(p.reserva, nombre+"/tema.json")
	if errorLectura != nil {
		return copia, nil, errorLectura
	}
	if err := decodificarJSON(b, &copia); err != nil {
		return copia, nil, err
	}
	b, errorLectura = fs.ReadFile(p.reserva, nombre+"/tokens.json")
	if errorLectura != nil {
		return copia, nil, errorLectura
	}
	if err := decodificarJSON(b, &tokens); err != nil {
		return copia, nil, err
	}
	if !copia.Valido() || copia.Identificador != nombre || !dominio.TokensValidos(tokens, copia.Tipo) {
		return copia, nil, fs.ErrInvalid
	}
	for _, ruta := range copia.Fuentes {
		b, err := fs.ReadFile(p.reserva, nombre+"/"+ruta)
		if err != nil {
			return copia, nil, err
		}
		if _, err := validarRecurso(b, path.Ext(ruta)); err != nil {
			return copia, nil, err
		}
	}
	return copia, tokens, nil
}

func leerRecurso(raiz *os.Root, ruta string) (dominio.Recurso, error) {
	var recurso dominio.Recurso
	if !fs.ValidPath(ruta) || strings.Contains(ruta, "\\") || strings.Contains(ruta, "/.") || strings.HasPrefix(ruta, ".") {
		return recurso, fs.ErrInvalid
	}
	extension := path.Ext(ruta)
	if extension != ".png" && extension != ".woff2" {
		return recurso, fs.ErrInvalid
	}
	info, err := raiz.Lstat(ruta)
	if err != nil {
		return recurso, err
	}
	if !info.Mode().IsRegular() || info.Size() > 8<<20 {
		return recurso, fs.ErrInvalid
	}
	f, err := raiz.Open(ruta)
	if err != nil {
		return recurso, err
	}
	defer f.Close()
	datos, err := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if err != nil {
		return recurso, err
	}
	return validarRecurso(datos, extension)
}

func validarRecurso(datos []byte, extension string) (dominio.Recurso, error) {
	var recurso dominio.Recurso
	if len(datos) > 8<<20 {
		return recurso, fs.ErrInvalid
	}
	if extension == ".png" {
		cfg, err := png.DecodeConfig(bytes.NewReader(datos))
		if err != nil || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width*cfg.Height > 16<<20 {
			return recurso, fs.ErrInvalid
		}
		if _, err := png.Decode(bytes.NewReader(datos)); err != nil {
			return recurso, fs.ErrInvalid
		}
		recurso.Tipo = "image/png"
	} else if extension == ".woff2" {
		if !woff2Valida(datos) {
			return recurso, fs.ErrInvalid
		}
		recurso.Tipo = "font/woff2"
	} else {
		return recurso, fs.ErrInvalid
	}
	recurso.Contenido = datos
	return recurso, nil
}

func (p *Paquetes) Tokens(ctx context.Context, nombre string) (map[string]string, error) {
	_, tokens, err := p.leerTema(ctx, nombre)
	return tokens, err
}
func (p *Paquetes) Temas(ctx context.Context) ([]dominio.Tema, error) {
	nombres, err := listarDirectorios(p.temas)
	if err != nil && p.reserva == nil {
		return nil, err
	}
	temas := []dominio.Tema{}
	defecto := false
	for _, nombre := range nombres {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tema, _, err := p.leerTema(ctx, nombre)
		if err == nil {
			temas = append(temas, tema)
			if nombre == dominio.TemaPredeterminado {
				defecto = true
			}
		}
	}
	if !defecto && p.reserva != nil {
		tema, _, err := p.leerTema(ctx, dominio.TemaPredeterminado)
		if err != nil {
			return nil, err
		}
		temas = append(temas, tema)
	}
	return temas, nil
}
func (p *Paquetes) RecursoTema(ctx context.Context, nombre, ruta string) (dominio.Recurso, error) {
	if _, _, err := p.leerTema(ctx, nombre); err != nil {
		return dominio.Recurso{}, err
	}
	if nombre == dominio.TemaPredeterminado && p.reserva != nil {
		tema, tokens, err := p.leerTemaDisco(ctx, nombre)
		if err != nil || !dominio.TokensValidos(tokens, tema.Tipo) {
			if !fs.ValidPath(ruta) || strings.Contains(ruta, "\\") {
				return dominio.Recurso{}, fs.ErrInvalid
			}
			b, err := fs.ReadFile(p.reserva, nombre+"/"+ruta)
			if err != nil {
				return dominio.Recurso{}, err
			}
			return validarRecurso(b, path.Ext(ruta))
		}
	}
	raiz, err := abrirDirectorio(p.temas, nombre, "")
	if err != nil {
		return dominio.Recurso{}, err
	}
	defer raiz.Close()
	return leerRecurso(raiz, ruta)
}

func (p *Paquetes) Tema(ctx context.Context, nombre string) (dominio.Tema, error) {
	tema, _, err := p.leerTema(ctx, nombre)
	return tema, err
}
