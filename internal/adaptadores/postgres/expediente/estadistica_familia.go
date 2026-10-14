package expediente

import (
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type familiaLectura struct {
	Composicion *string `json:"composicion_familiar"`
	Numerosa    *bool   `json:"familia_numerosa"`
	Economia    *string `json:"situacion_economica"`
	Ingresos    *int64  `json:"ingresos_anuales_centimos"`
	Tramo       *string `json:"tramo_smi"`
}
type adultoLectura struct {
	Papel, Sexo string
	Nacimiento  *fechaLectura `json:"fecha_nacimiento"`
	Educacion   *string       `json:"nivel_educativo"`
	Trabajo     *string       `json:"situacion_laboral"`
}

func datoLectura[T any](v *T) d.Dato[T] {
	if v == nil {
		return d.Dato[T]{}
	}
	return d.Dato[T]{Conocido: true, Valor: *v}
}
func revisionLectura[T any](v vigenciaLectura, valor d.Dato[T]) d.Revision[T] {
	return d.Revision[T]{ID: v.ID, Vigencia: d.Vigencia{Desde: v.Desde.Time}, RegistradoEn: v.Registrado, Dato: valor}
}
func agregarHistoria[T any](h *d.Historia[T], r d.Revision[T]) {
	if len(*h) > 0 {
		anterior := (*h)[len(*h)-1]
		if anterior.Vigencia.Desde.Format("2006-01-02") == r.Vigencia.Desde.Format("2006-01-02") {
			(*h)[len(*h)-1] = r
			return
		}
		(*h)[len(*h)-1].Vigencia.Hasta = r.Vigencia.Desde
	}
	*h = append(*h, r)
}
func incorporarFamilia(exp *d.Expediente, familias, adultos []vigenciaLectura) error {
	for _, v := range familias {
		var f familiaLectura
		if err := json.Unmarshal(v.Valor, &f); err != nil {
			return err
		}
		agregarHistoria(&exp.Composicion, revisionLectura(v, datoLectura(f.Composicion)))
		agregarHistoria(&exp.FamiliaNumerosa, revisionLectura(v, datoLectura(f.Numerosa)))
		agregarHistoria(&exp.SituacionEconomica, revisionLectura(v, datoLectura(f.Economia)))
		ingreso := d.Ingresos{NetosAnualesCentimos: datoLectura(f.Ingresos), Tramo: datoLectura(f.Tramo)}
		agregarHistoria(&exp.Ingresos, revisionLectura(v, d.Dato[d.Ingresos]{Valor: ingreso, Conocido: f.Ingresos != nil || f.Tramo != nil}))
	}
	for _, v := range adultos {
		var aa []adultoLectura
		if err := json.Unmarshal(v.Valor, &aa); err != nil {
			return err
		}
		lista := []d.Adulto{}
		for _, a := range aa {
			adulto := d.Adulto{Papel: a.Papel, Sexo: sexoLectura(a.Sexo)}
			if a.Nacimiento != nil {
				adulto.Nacimiento = a.Nacimiento.Time
			}
			if a.Educacion != nil {
				adulto.NivelEducativo = *a.Educacion
			}
			if a.Trabajo != nil {
				adulto.SituacionLaboral = *a.Trabajo
			}
			lista = append(lista, adulto)
		}
		agregarHistoria(&exp.Adultos, revisionLectura(v, d.Dato[[]d.Adulto]{Valor: lista, Conocido: true}))
	}
	return nil
}
func sexoLectura(s string) d.Sexo {
	switch s {
	case "H":
		return d.Nino
	case "M":
		return d.Nina
	}
	return ""
}
