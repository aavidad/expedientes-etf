package expediente

import (
	"errors"
	"time"
)

var ErrMenores = errors.New("expediente.menores_invalidos")

type Menor struct {
	// Sólo el adaptador marca identidades que conservan la lectura inicial;
	// un nombre vacío guardado después no se rellena desde otro menor del anexo.
	DatosDesdeAnexo        bool    `json:"-"`
	ID                     string  `json:"id"`
	Orden                  int     `json:"orden"`
	Nombre                 string  `json:"nombre"`
	Apellidos              string  `json:"apellidos"`
	Sexo                   string  `json:"sexo"`
	Nacimiento             *string `json:"fecha_nacimiento"`
	Escolarizacion         *string `json:"nivel_escolarizacion"`
	CentroEducativo        string  `json:"centro_educativo"`
	Subprograma            string  `json:"subprograma"`
	CentroProteccionCodigo *string `json:"centro_proteccion_codigo"`
	CentroProteccionTexto  string  `json:"centro_proteccion_texto"`
	AtencionTemprana       *bool   `json:"propuesto_atencion_temprana"`
}

// MenorEstadistico excluye nombres de personas y centros escritos libremente.
type MenorEstadistico struct {
	ID                     string  `json:"id"`
	Orden                  int     `json:"orden"`
	Sexo                   string  `json:"sexo"`
	Nacimiento             *string `json:"fecha_nacimiento"`
	Escolarizacion         *string `json:"nivel_escolarizacion"`
	Subprograma            string  `json:"subprograma"`
	CentroProteccionCodigo *string `json:"centro_proteccion_codigo"`
	AtencionTemprana       *bool   `json:"propuesto_atencion_temprana"`
}
type CambioMenores struct {
	Version int      `json:"version"`
	Desde   string   `json:"vigente_desde"`
	Menores *[]Menor `json:"menores"`
}
type ResultadoMenores = ResultadoFamilia

type FichaMenores struct {
	ID          string    `json:"id"`
	EquipoID    string    `json:"equipo_id"`
	Estado      string    `json:"estado"`
	Version     int       `json:"version"`
	Aceptacion  time.Time `json:"fecha_aceptacion"`
	Subprograma string    `json:"subprograma"`
	Menores     []Menor   `json:"menores"`
}
type HistoriaMenores struct {
	Menores  []MenorEstadistico `json:"menores"`
	Vigencia *Vigencia          `json:"vigencia"`
}
type MenorConsulta struct {
	MenorEstadistico
	ExpedienteID string   `json:"expediente_id"`
	EquipoID     string   `json:"equipo_id"`
	Vigencia     Vigencia `json:"vigencia"`
}
type CatalogosMenores struct{ Opciones map[string][]string }

func (p CambioMenores) Validar(c CatalogosMenores, aceptacion, ahora time.Time) error {
	fecha, err := time.Parse("2006-01-02", p.Desde)
	if err != nil || fecha.Year() < 1 || fecha.Before(aceptacion) || p.Desde > FechaCivil(ahora) || p.Version < 1 || p.Menores == nil || len(*p.Menores) > 15 {
		return ErrMenores
	}
	ids, ordenes := map[string]bool{}, map[int]bool{}
	for _, m := range *p.Menores {
		if m.ID != "" && (!IDValido(m.ID) || ids[m.ID]) || m.Orden < 1 || m.Orden > 15 || ordenes[m.Orden] {
			return ErrMenores
		}
		ids[m.ID], ordenes[m.Orden] = true, true
		if m.Sexo != "H" && m.Sexo != "M" || m.Subprograma != "RIESGO_SOCIAL" && m.Subprograma != "REUNIFICACION" {
			return ErrMenores
		}
		cat := CatalogosFamilia{Opciones: c.Opciones}
		if !cat.Admite("nivel_escolarizacion", m.Escolarizacion) || !cat.Admite("centro_proteccion", m.CentroProteccionCodigo) || m.CentroProteccionCodigo != nil && m.CentroProteccionTexto != "" {
			return ErrMenores
		}
		if m.Nacimiento != nil {
			n, e := time.Parse("2006-01-02", *m.Nacimiento)
			if e != nil || n.Year() < 1 || n.After(fecha) {
				return ErrMenores
			}
		}
		for _, s := range []string{m.Nombre, m.Apellidos, m.CentroEducativo} {
			if !textoFamilia(s, 254) {
				return ErrMenores
			}
		}
		if !textoFamilia(m.CentroProteccionTexto, 254) {
			return ErrMenores
		}
	}
	return nil
}
func (m Menor) Estadistica() MenorEstadistico {
	return MenorEstadistico{m.ID, m.Orden, m.Sexo, m.Nacimiento, m.Escolarizacion, m.Subprograma, m.CentroProteccionCodigo, m.AtencionTemprana}
}

func (a Actor) AutorizarMenores(escritura bool) error { return a.AutorizarFamilia(escritura) }
