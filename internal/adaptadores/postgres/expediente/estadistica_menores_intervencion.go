package expediente

import (
	"encoding/json"

	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

type menorEstadistica struct {
	ID, Sexo, Subprograma string
	Nacimiento            *fechaLectura `json:"fecha_nacimiento"`
	AtencionTemprana      *bool         `json:"propuesto_atencion_temprana"`
	CentroProteccion      *string       `json:"centro_proteccion_codigo"`
}

type intervencionEstadistica struct {
	Fase            *string  `json:"fase"`
	ListasConocidas []string `json:"listas_conocidas"`
	Marcas          []struct {
		Lista, Codigo    string
		MenoresExcluidos []string `json:"menores_excluidos"`
	} `json:"marcas"`
}

func incorporarMenoresIntervencion(v *datosLectura, menores, intervenciones []byte) error {
	var ms, is []vigenciaLectura
	if err := json.Unmarshal(menores, &ms); err != nil {
		return err
	}
	if err := json.Unmarshal(intervenciones, &is); err != nil {
		return err
	}
	for _, r := range ms {
		var lista []menorEstadistica
		if err := json.Unmarshal(r.Valor, &lista); err != nil {
			return err
		}
		valor := []d.Menor{}
		for _, m := range lista {
			menor := d.Menor{ID: m.ID, Sexo: sexoLectura(m.Sexo), Subprograma: subprogramaLectura(m.Subprograma), PropuestaAtencionTemprana: datoLectura(m.AtencionTemprana)}
			menor.CentroProteccion = datoLectura(m.CentroProteccion)
			if m.Nacimiento != nil {
				menor.Nacimiento = m.Nacimiento.Time
			}
			valor = append(valor, menor)
		}
		agregarHistoria(&v.Expediente.Menores, revisionLectura(r, d.Dato[[]d.Menor]{Conocido: true, Valor: valor}))
	}
	for _, r := range is {
		var i intervencionEstadistica
		if err := json.Unmarshal(r.Valor, &i); err != nil {
			return err
		}
		factores := []d.Factor{}
		recursos := []string{}
		for _, m := range i.Marcas {
			factores = append(factores, d.Factor{Lista: m.Lista, Codigo: m.Codigo, Presente: d.Dato[bool]{Conocido: true, Valor: true}, MenoresExcluidos: m.MenoresExcluidos})
			if m.Lista == "coordinacion" {
				recursos = append(recursos, m.Codigo)
			}
		}
		coordinacionConocida := false
		for _, lista := range i.ListasConocidas {
			if lista == "coordinacion" {
				coordinacionConocida = true
			}
		}
		agregarHistoria(&v.Fases, revisionLectura(r, datoLectura(i.Fase)))
		agregarHistoria(&v.Expediente.ListasConocidas, revisionLectura(r, d.Dato[[]string]{Conocido: true, Valor: i.ListasConocidas}))
		agregarHistoria(&v.Expediente.Factores, revisionLectura(r, d.Dato[[]d.Factor]{Conocido: true, Valor: factores}))
		agregarHistoria(&v.Expediente.RecursosCoordinacion, revisionLectura(r, d.Dato[[]string]{Conocido: coordinacionConocida, Valor: recursos}))
	}
	return nil
}
