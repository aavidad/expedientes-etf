package expediente

import (
	"errors"
	"math/big"
	"slices"
	"strings"
	"time"
)

var ErrFamilia = errors.New("expediente.familia_invalida")

type Familia struct {
	Direccion        string  `json:"direccion"`
	Telefono         string  `json:"telefono_contacto"`
	Email            string  `json:"email_contacto"`
	Composicion      *string `json:"composicion_familiar"`
	ComposicionOtras string  `json:"composicion_otras_texto"`
	Numerosa         *bool   `json:"familia_numerosa"`
	Vivienda         *string `json:"vivienda"`
	Economia         *string `json:"situacion_economica"`
	Ingresos         *int64  `json:"ingresos_anuales_centimos"`
	Tramo            *string `json:"tramo_smi"`
}

type Adulto struct {
	ID           string  `json:"id"`
	Orden        int     `json:"orden"`
	Nombre       string  `json:"nombre"`
	Apellidos    string  `json:"apellidos"`
	Papel        string  `json:"papel"`
	Sexo         string  `json:"sexo"`
	Nacimiento   *string `json:"fecha_nacimiento"`
	Edad         *int    `json:"edad_declarada"`
	Educacion    *string `json:"nivel_educativo"`
	Trabajo      *string `json:"situacion_laboral"`
	TrabajoTexto string  `json:"situacion_laboral_texto"`
	Tutor        *bool   `json:"es_tutor"`
	Telefono     string  `json:"telefono"`
	Email        string  `json:"email"`
	OtrosDatos   string  `json:"otros_datos"`
}

// AdultoEstadistico excluye nombres, contactos y narrativas de la historia.
type AdultoEstadistico struct {
	ID         string  `json:"id"`
	Papel      string  `json:"papel"`
	Sexo       string  `json:"sexo"`
	Nacimiento *string `json:"fecha_nacimiento"`
	Edad       *int    `json:"edad_declarada"`
	Educacion  *string `json:"nivel_educativo"`
	Trabajo    *string `json:"situacion_laboral"`
	Tutor      *bool   `json:"es_tutor"`
}

type FamiliaEstadistica struct {
	Composicion *string `json:"composicion_familiar"`
	Numerosa    *bool   `json:"familia_numerosa"`
	Vivienda    *string `json:"vivienda"`
	Economia    *string `json:"situacion_economica"`
	Ingresos    *int64  `json:"ingresos_anuales_centimos"`
	Tramo       *string `json:"tramo_smi"`
}

type CambioFamilia struct {
	Version int       `json:"version"`
	Desde   string    `json:"vigente_desde"`
	Familia *Familia  `json:"familia,omitempty"`
	Adultos *[]Adulto `json:"adultos,omitempty"`
}
type ResultadoFamilia struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type FichaFamilia struct {
	ID         string    `json:"id"`
	EquipoID   string    `json:"equipo_id"`
	Estado     string    `json:"estado"`
	Version    int       `json:"version"`
	Aceptacion time.Time `json:"fecha_aceptacion"`
	Familia    Familia   `json:"familia"`
	Adultos    []Adulto  `json:"adultos"`
}
type Vigencia struct {
	Desde    string  `json:"desde"`
	Hasta    *string `json:"hasta"`
	Revision int     `json:"revision"`
}
type AdultoConsulta struct {
	ID           string   `json:"id"`
	ExpedienteID string   `json:"expediente_id"`
	Papel        string   `json:"papel"`
	Sexo         string   `json:"sexo"`
	Vigencia     Vigencia `json:"vigencia"`
}
type HistoriaFamilia struct {
	Familia         FamiliaEstadistica  `json:"familia"`
	Adultos         []AdultoEstadistico `json:"adultos"`
	VigenciaFamilia *Vigencia           `json:"vigencia_familia"`
	VigenciaAdultos *Vigencia           `json:"vigencia_adultos"`
	TramoCalculado  *string             `json:"tramo_calculado"`
	AnioSMI         int                 `json:"anio_smi"`
	VersionSMI      *int                `json:"version_smi"`
}
type TramoFamilia struct {
	Codigo string `json:"codigo"`
	Desde  int64  `json:"desde"`
	Hasta  *int64 `json:"hasta"`
}
type CatalogosFamilia struct {
	Opciones map[string][]string `json:"opciones"`
	Tramos   []TramoFamilia      `json:"tramos"`
}

func (a Actor) AutorizarFamilia(escritura bool) error {
	if !IDValido(a.UsuarioID) {
		return ErrAcceso
	}
	if !escritura && slices.Contains(a.Perfiles, "coordinacion") {
		return nil
	}
	if slices.Contains(a.Perfiles, "profesional_etf") && IDValido(a.EquipoID) {
		return nil
	}
	return ErrAcceso
}

func (c CatalogosFamilia) Admite(lista string, v *string) bool {
	return v == nil || slices.Contains(c.Opciones[lista], *v)
}
func textoFamilia(s string, max int) bool { return len(s) <= max && !strings.ContainsRune(s, '\x00') }
func (p CambioFamilia) Validar(c CatalogosFamilia, aceptacion, ahora time.Time) error {
	fecha, err := time.Parse("2006-01-02", p.Desde)
	if err != nil || fecha.Before(aceptacion) || p.Desde > FechaCivil(ahora) || p.Version < 1 || (p.Familia == nil) == (p.Adultos == nil) {
		return ErrFamilia
	}
	if f := p.Familia; f != nil {
		if !c.Admite("composicion_familiar", f.Composicion) || !c.Admite("vivienda", f.Vivienda) || !c.Admite("situacion_economica", f.Economia) || !c.Admite("tramo_ingresos_smi", f.Tramo) || f.Ingresos != nil && (*f.Ingresos < 0 || f.Tramo != nil) {
			return ErrFamilia
		}
		if !textoFamilia(f.Direccion, 2000) || !textoFamilia(f.Telefono, 100) || !textoFamilia(f.Email, 254) || !textoFamilia(f.ComposicionOtras, 4000) {
			return ErrFamilia
		}
		if f.ComposicionOtras != "" && (f.Composicion == nil || *f.Composicion != "OTRAS") {
			return ErrFamilia
		}
	}
	if p.Adultos != nil {
		if len(*p.Adultos) > 100 {
			return ErrFamilia
		}
		ids, ordenes := map[string]bool{}, map[int]bool{}
		for _, a := range *p.Adultos {
			if a.ID != "" && (!IDValido(a.ID) || ids[a.ID]) || a.Orden < 1 || a.Orden > 100 || ordenes[a.Orden] {
				return ErrFamilia
			}
			ids[a.ID], ordenes[a.Orden] = true, true
			if a.Papel == "" || a.Sexo == "" || !c.Admite("papel_parental", &a.Papel) || !c.Admite("sexo", &a.Sexo) || !c.Admite("nivel_educativo", a.Educacion) || !c.Admite("situacion_laboral", a.Trabajo) {
				return ErrFamilia
			}
			if a.Nacimiento != nil {
				n, e := time.Parse("2006-01-02", *a.Nacimiento)
				if e != nil || n.After(fecha) || n.Year() < 1 || a.Edad != nil {
					return ErrFamilia
				}
			}
			if a.Edad != nil && (*a.Edad < 18 || *a.Edad > 130) {
				return ErrFamilia
			}
			if a.TrabajoTexto != "" && (a.Trabajo == nil || *a.Trabajo != "OTRAS") {
				return ErrFamilia
			}
			for _, s := range []string{a.Nombre, a.Apellidos, a.Telefono, a.Email} {
				if !textoFamilia(s, 254) {
					return ErrFamilia
				}
			}
			if !textoFamilia(a.TrabajoTexto, 4000) || !textoFamilia(a.OtrosDatos, 4000) {
				return ErrFamilia
			}
		}
	}
	return nil
}
func (f Familia) Estadistica() FamiliaEstadistica {
	return FamiliaEstadistica{f.Composicion, f.Numerosa, f.Vivienda, f.Economia, f.Ingresos, f.Tramo}
}
func (a Adulto) Estadistica() AdultoEstadistico {
	return AdultoEstadistico{a.ID, a.Papel, a.Sexo, a.Nacimiento, a.Edad, a.Educacion, a.Trabajo, a.Tutor}
}

// CalcularTramo compara proporciones exactas, sin desbordamientos ni flotantes.
func (c CatalogosFamilia) CalcularTramo(ingresos, smi int64) *string {
	if ingresos < 0 || smi <= 0 {
		return nil
	}
	valor := new(big.Int).Mul(big.NewInt(ingresos), big.NewInt(100))
	for _, t := range c.Tramos {
		desde := new(big.Int).Mul(big.NewInt(smi), big.NewInt(t.Desde))
		if valor.Cmp(desde) < 0 {
			continue
		}
		if t.Hasta != nil && valor.Cmp(new(big.Int).Mul(big.NewInt(smi), big.NewInt(*t.Hasta))) >= 0 {
			continue
		}
		codigo := t.Codigo
		return &codigo
	}
	return nil
}
