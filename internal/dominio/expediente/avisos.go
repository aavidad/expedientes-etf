package expediente

import "time"

// DatosAvisos contiene únicamente fechas, códigos y atributos estadísticos.
type DatosAvisos struct {
	AdultosConocidos bool                `json:"adultos_conocidos"`
	ID               string              `json:"id"`
	EquipoID         string              `json:"equipo_id"`
	Estado           string              `json:"estado"`
	Subprograma      string              `json:"subprograma"`
	Aceptacion       string              `json:"fecha_aceptacion"`
	Consentimiento   *string             `json:"consentimiento"`
	AtendidoAnio     bool                `json:"atendido_anio"`
	Familia          FamiliaEstadistica  `json:"familia"`
	Adultos          []AdultoEstadistico `json:"adultos"`
	Menores          []MenorEstadistico  `json:"menores"`
	Intervencion     *Intervencion       `json:"intervencion"`
	FaseDesde        *string             `json:"fase_desde"`
}

type ReglasAvisos struct {
	ConsentimientoDias int
	Duraciones         map[string]int
}

type Aviso struct {
	Codigo      string `json:"codigo"`
	ClaveTexto  string `json:"clave_texto"`
	Apartado    string `json:"apartado,omitempty"`
	Campo       string `json:"campo"`
	SujetoID    string `json:"sujeto_id,omitempty"`
	Enlace      string `json:"enlace"`
	FechaLimite string `json:"fecha_limite,omitempty"`
}
type AvisosExpediente struct {
	ID            string  `json:"expediente_id"`
	Corte         string  `json:"corte"`
	VersionPlazos int     `json:"version_plazos"`
	Avisos        []Aviso `json:"avisos"`
}
type RecuentoAvisos struct {
	EquipoID    string         `json:"equipo_id"`
	Total       int            `json:"total"`
	Expedientes int            `json:"expedientes"`
	PorCodigo   map[string]int `json:"por_codigo"`
}

// CalcularAvisos compara fechas civiles, sin depender de horas ni cambios de horario.
func CalcularAvisos(d DatosAvisos, hoy string, reglas ReglasAvisos, memoria bool) ([]Aviso, error) {
	fecha, err := time.Parse("2006-01-02", hoy)
	if err != nil || reglas.ConsentimientoDias < 0 {
		return nil, ErrPeticion
	}
	r := []Aviso{}
	agregar := func(codigo, apartado, campo, sujeto, limite string) {
		enlace := "/expedientes/" + d.ID + "#" + campo
		if sujeto != "" {
			enlace += "-" + sujeto
		}
		r = append(r, Aviso{Codigo: codigo, ClaveTexto: "expediente.aviso_" + codigo, Apartado: apartado, Campo: campo, SujetoID: sujeto, Enlace: enlace, FechaLimite: limite})
	}
	if d.Estado == Aceptada || d.Estado == EnIntervencion {
		if d.Consentimiento == nil {
			inicio, e := time.Parse("2006-01-02", d.Aceptacion)
			if e != nil {
				return nil, ErrNoDisponible
			}
			limite := inicio.AddDate(0, 0, reglas.ConsentimientoDias)
			if fecha.After(limite) {
				agregar("consentimiento", "", "consentimiento", "", FechaCivil(limite))
			}
		}
		if d.Intervencion != nil && d.Intervencion.Fase != nil && d.FaseDesde != nil {
			dias, ok := reglas.Duraciones[d.Subprograma+":"+*d.Intervencion.Fase]
			if !ok || dias < 0 {
				return nil, ErrNoDisponible
			}
			inicio, e := time.Parse("2006-01-02", *d.FaseDesde)
			if e != nil {
				return nil, ErrNoDisponible
			}
			limite := inicio.AddDate(0, 0, dias)
			if fecha.After(limite) {
				agregar("fase", "", "fase", "", FechaCivil(limite))
			}
		}
	}
	if !memoria {
		return r, nil
	}
	falta := func(apartado, campo, sujeto string, ausente bool) {
		if ausente {
			agregar("memoria_dato_faltante", apartado, campo, sujeto, "")
		}
	}
	f := d.Familia
	for _, apartado := range []string{"19", "20", "21"} {
		falta(apartado, "adultos", "", !d.AdultosConocidos)
	}
	falta("17", "composicion_familiar", "", f.Composicion == nil)
	falta("18", "familia_numerosa", "", f.Numerosa == nil)
	falta("22", "situacion_economica", "", f.Economia == nil)
	falta("23", "ingresos_anuales_centimos", "", f.Ingresos == nil && f.Tramo == nil)
	for _, a := range d.Adultos {
		falta("19", "sexo", a.ID, a.Sexo == "")
		falta("19", "fecha_nacimiento", a.ID, a.Nacimiento == nil && a.Edad == nil)
		falta("20", "nivel_educativo", a.ID, a.Educacion == nil)
		falta("21", "situacion_laboral", a.ID, a.Trabajo == nil)
	}
	for _, m := range d.Menores {
		falta("25", "sexo", m.ID, m.Sexo == "")
		falta("25", "fecha_nacimiento", m.ID, m.Nacimiento == nil)
		falta("25", "subprograma", m.ID, m.Subprograma == "")
		falta("14.1", "propuesto_atencion_temprana", m.ID, m.AtencionTemprana == nil)
	}
	for _, c := range []struct{ lista, apartado string }{{"coordinacion", "14"}, {"factor_familia", "24"}, {"factor_menor", "26"}, {"tipologia", "27"}} {
		falta(c.apartado, c.lista, "", d.Intervencion == nil || d.Intervencion.Listas[c.lista] == nil)
	}
	return r, nil
}
