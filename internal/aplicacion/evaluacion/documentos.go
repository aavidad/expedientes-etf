package evaluacion

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	exp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"slices"
	"strings"
	"unicode"
)

type lecturaMenoresDocumento interface {
	ObtenerMenores(context.Context, string) (exp.FichaMenores, error)
}

// ParaDocumentos participa en la transacción del documento, sin abrir otra.
func (s *Servicio) ParaDocumentos(ctx context.Context, actor d.Actor, id string) (d.DatosDocumento, error) {
	if err := actor.Autorizar(false); err != nil {
		return d.DatosDocumento{}, err
	}
	if !d.IDValido(id) {
		return d.DatosDocumento{}, d.ErrPeticion
	}
	familia, err := s.Familias.ObtenerFamilia(ctx, id)
	if err != nil {
		return d.DatosDocumento{}, d.ErrNoEncontrada
	}
	if !slices.Contains(actor.Perfiles, "coordinacion") && familia.EquipoID != actor.EquipoID {
		return d.DatosDocumento{}, d.ErrAcceso
	}
	r, err := s.Repo.Consultar(ctx, id)
	if err != nil {
		return d.DatosDocumento{}, err
	}
	datos := d.DatosDocumento{Registro: r, Iniciales: map[string]string{}}
	for _, adulto := range familia.Adultos {
		datos.Iniciales[adulto.ID] = iniciales(adulto.Nombre + " " + adulto.Apellidos)
	}
	definiciones := d.Definiciones{}
	vistas := map[string]bool{}
	conMenores := false
	for _, aplicacion := range r.Escalas {
		conMenores = conMenores || aplicacion.Sujeto.Tipo == "menor"
		if vistas[aplicacion.VersionEscalaID] {
			continue
		}
		escala, err := s.Catalogo.Escala(ctx, aplicacion.VersionEscalaID)
		if err != nil {
			return d.DatosDocumento{}, err
		}
		vistas[escala.ID] = true
		definiciones.Escalas = append(definiciones.Escalas, escala)
	}
	if conMenores && s.MenoresDocumento != nil {
		menores, err := s.MenoresDocumento.ObtenerMenores(ctx, id)
		if err != nil {
			return d.DatosDocumento{}, err
		}
		for _, menor := range menores.Menores {
			datos.Iniciales[menor.ID] = iniciales(menor.Nombre + " " + menor.Apellidos)
		}
	}
	if len(r.Escalas) > 0 && s.SujetosDocumento != nil {
		nombres, err := s.SujetosDocumento.NombresSujetos(ctx, id)
		if err != nil {
			return d.DatosDocumento{}, err
		}
		for sujeto, nombre := range nombres {
			if nombre != "" {
				datos.Iniciales[sujeto] = iniciales(nombre)
			}
		}
	}
	datos.Registro.Definiciones = &definiciones
	return datos, nil
}

func iniciales(nombre string) string {
	var partes []string
	for _, palabra := range strings.Fields(nombre) {
		for _, letra := range palabra {
			if unicode.IsLetter(letra) {
				partes = append(partes, strings.ToUpper(string(letra))+".")
				break
			}
		}
	}
	return strings.Join(partes, " ")
}
