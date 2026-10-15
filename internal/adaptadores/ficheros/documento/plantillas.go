package documento

import (
	"context"
	"errors"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
)

type ReferenciasIdioma interface {
	Referencias(context.Context, string) ([]p.Referencia, error)
}
type Plantillas struct {
	Raiz     string
	Paquetes ReferenciasIdioma
	// Registro recibe avisos técnicos sin datos personales; nil usa el predeterminado.
	Registro *slog.Logger
}

var codigoIdioma = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
var version = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func (f Plantillas) Resolver(ctx context.Context, idioma string) (d.IdiomaPlantilla, error) {
	if len(idioma) > 63 || !codigoIdioma.MatchString(idioma) || f.Paquetes == nil {
		return d.IdiomaPlantilla{}, d.ErrPlantilla
	}
	refs, err := f.Paquetes.Referencias(ctx, p.TipoIdioma)
	if err != nil {
		return d.IdiomaPlantilla{}, d.ErrNoDisponible
	}
	for _, r := range refs {
		if r.Codigo == idioma {
			if r.Tipo != p.TipoIdioma || !fs.ValidPath(r.Ruta) || !strings.HasPrefix(r.Ruta, ".versiones/"+idioma+"/") || !strings.HasSuffix(r.Ruta, "/"+idioma) {
				return d.IdiomaPlantilla{}, d.ErrPlantilla
			}
			return d.IdiomaPlantilla{Ruta: r.Ruta, Version: r.Version}, nil
		}
	}
	// Como paquete.Activos: los idiomas de distribución son la reserva inicial.
	return d.IdiomaPlantilla{Ruta: idioma, Version: "distribucion"}, nil
}
func (f Plantillas) Leer(ctx context.Context, idioma d.IdiomaPlantilla, p d.Plantilla) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	publicada := d.PlantillaPublicada(p)
	if !codigoIdioma.MatchString(p.Idioma) || !fs.ValidPath(idioma.Ruta) || (idioma.Ruta != p.Idioma && (!strings.HasPrefix(idioma.Ruta, ".versiones/"+p.Idioma+"/") || !strings.HasSuffix(idioma.Ruta, "/"+p.Idioma))) || !version.MatchString(p.Version) || !d.TipoPlantilla(p.Tipo) || (!publicada && p.Ruta != path.Join("plantillas", p.Tipo, "v"+p.Version+".docx")) {
		return nil, d.ErrPlantilla
	}
	raiz, err := os.OpenRoot(f.Raiz)
	if err != nil {
		return nil, d.ErrPlantilla
	}
	defer raiz.Close()
	if publicada {
		return leerPlantilla(raiz, path.Join(p.Idioma, p.Ruta), p.Huella)
	}
	// Una versión de idioma publicada antes de una actualización no trae las
	// versiones nuevas de plantillas de la distribución: se buscan después en
	// la base del mismo idioma, siempre con la huella que espera la base de datos.
	b, err := leerPlantilla(raiz, path.Join(idioma.Ruta, p.Ruta), p.Huella)
	if err != nil && idioma.Ruta != p.Idioma {
		if reserva, e := leerPlantilla(raiz, path.Join(p.Idioma, p.Ruta), p.Huella); e == nil {
			if errors.Is(err, d.ErrHuella) {
				// Una copia alterada en el paquete activo debe investigarse aunque se sirva la verificada.
				registro := f.Registro
				if registro == nil {
					registro = slog.Default()
				}
				registro.WarnContext(ctx, "plantilla.huella_paquete_activo", "idioma", p.Idioma, "tipo", p.Tipo, "version", p.Version, "paquete", idioma.Version)
			}
			return reserva, nil
		}
	}
	return b, err
}

func leerPlantilla(raiz *os.Root, ruta, huella string) ([]byte, error) {
	archivo, err := raiz.Open(ruta)
	if err != nil {
		return nil, d.ErrPlantilla
	}
	defer archivo.Close()
	info, err := archivo.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 10<<20 {
		return nil, d.ErrPlantilla
	}
	b, err := io.ReadAll(io.LimitReader(archivo, (10<<20)+1))
	if err != nil || len(b) > 10<<20 {
		clear(b)
		return nil, d.ErrPlantilla
	}
	if d.Huella(b) != huella {
		clear(b)
		return nil, d.ErrHuella
	}
	return b, nil
}
