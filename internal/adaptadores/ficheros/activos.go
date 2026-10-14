package ficheros

import (
	"context"
	"path"
	"path/filepath"
	"sort"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
)

// Activos conserva los paquetes de distribución como reserva inicial. Para
// cada código publicado sólo abre la ruta confirmada; no explora .versiones.
type Activos struct {
	inicial     *Paquetes
	referencias salida.Publicacion
}

func NuevosActivos(p *Paquetes, r salida.Publicacion) *Activos { return &Activos{p, r} }

func (a *Activos) resolver(ctx context.Context, tipo, codigo string) (*Paquetes, error) {
	refs, err := a.referencias.Referencias(ctx, tipo)
	if err != nil {
		return nil, err
	}
	for _, r := range refs {
		if r.Codigo == codigo {
			if !rutaVersionValida(r.VersionInstalada) {
				return nil, dominio.ErrNoDisponible
			}
			p := *a.inicial
			if tipo == dominio.TipoIdioma {
				p.idiomas = filepath.Join(p.idiomas, filepath.FromSlash(path.Dir(r.Ruta)))
			} else {
				p.temas = filepath.Join(p.temas, filepath.FromSlash(path.Dir(r.Ruta)))
				p.reserva = nil
			}
			return &p, nil
		}
	}
	return a.inicial, nil
}

func (a *Activos) Textos(ctx context.Context, codigo string) (map[string]string, error) {
	p, err := a.resolver(ctx, dominio.TipoIdioma, codigo)
	if err != nil {
		return nil, err
	}
	return p.Textos(ctx, codigo)
}
func (a *Activos) Tema(ctx context.Context, codigo string) (dominio.Tema, error) {
	p, err := a.resolver(ctx, dominio.TipoTema, codigo)
	if err != nil {
		return dominio.Tema{}, err
	}
	return p.Tema(ctx, codigo)
}
func (a *Activos) Tokens(ctx context.Context, codigo string) (map[string]string, error) {
	p, err := a.resolver(ctx, dominio.TipoTema, codigo)
	if err != nil {
		return nil, err
	}
	return p.Tokens(ctx, codigo)
}
func (a *Activos) RecursoTema(ctx context.Context, codigo, ruta string) (dominio.Recurso, error) {
	p, err := a.resolver(ctx, dominio.TipoTema, codigo)
	if err != nil {
		return dominio.Recurso{}, err
	}
	return p.RecursoTema(ctx, codigo, ruta)
}

func (a *Activos) Idiomas(ctx context.Context) ([]dominio.Idioma, error) {
	refs, err := a.referencias.Referencias(ctx, dominio.TipoIdioma)
	if err != nil {
		return nil, err
	}
	base, err := a.inicial.Idiomas(ctx)
	if err != nil {
		return nil, err
	}
	porCodigo := map[string]dominio.Idioma{}
	for _, v := range base {
		porCodigo[v.Codigo] = v
	}
	for _, r := range refs {
		delete(porCodigo, r.Codigo)
		if !rutaVersionValida(r.VersionInstalada) {
			return nil, dominio.ErrNoDisponible
		}
		p := NuevosPaquetes(filepath.Join(a.inicial.idiomas, filepath.FromSlash(path.Dir(r.Ruta))), a.inicial.temas)
		v, err := p.leerIdioma(r.Codigo)
		if err != nil {
			return nil, dominio.ErrNoDisponible
		}
		if _, err = p.Textos(ctx, r.Codigo); err != nil {
			return nil, dominio.ErrNoDisponible
		}
		v.Predeterminado = v.Codigo == dominio.IdiomaPredeterminado
		porCodigo[v.Codigo] = v
	}
	resultado := []dominio.Idioma{}
	for _, v := range porCodigo {
		resultado = append(resultado, v)
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Codigo < resultado[j].Codigo })
	return resultado, nil
}

func (a *Activos) Temas(ctx context.Context) ([]dominio.Tema, error) {
	refs, err := a.referencias.Referencias(ctx, dominio.TipoTema)
	if err != nil {
		return nil, err
	}
	base, err := a.inicial.Temas(ctx)
	if err != nil {
		return nil, err
	}
	porCodigo := map[string]dominio.Tema{}
	for _, v := range base {
		porCodigo[v.Identificador] = v
	}
	for _, r := range refs {
		delete(porCodigo, r.Codigo)
		if !rutaVersionValida(r.VersionInstalada) {
			return nil, dominio.ErrNoDisponible
		}
		p := NuevosPaquetes(a.inicial.idiomas, filepath.Join(a.inicial.temas, filepath.FromSlash(path.Dir(r.Ruta))))
		v, err := p.Tema(ctx, r.Codigo)
		if err != nil {
			return nil, dominio.ErrNoDisponible
		}
		porCodigo[v.Identificador] = v
	}
	resultado := []dominio.Tema{}
	for _, v := range porCodigo {
		resultado = append(resultado, v)
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Identificador < resultado[j].Identificador })
	return resultado, nil
}
