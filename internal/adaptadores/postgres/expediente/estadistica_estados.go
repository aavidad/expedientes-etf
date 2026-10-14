package expediente

import (
	"cmp"
	"slices"
	"time"

	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

func historiaEstados(id string, recepcion time.Time, cambios []cambioLectura) d.Historia[string] {
	h := d.Historia[string]{}
	agregarHistoria(&h, d.Revision[string]{ID: id + ":recepcion", Vigencia: d.Vigencia{Desde: recepcion}, RegistradoEn: recepcion, Dato: d.Dato[string]{Conocido: true, Valor: "derivacion_recibida"}})
	cambios = slices.Clone(cambios)
	slices.SortFunc(cambios, func(a, b cambioLectura) int {
		if c := a.Registrado.Compare(b.Registrado); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	for i := 0; i < len(cambios); {
		c := cambios[i]
		dato := d.Dato[string]{Conocido: true, Valor: c.Estado}
		j := i + 1
		for j < len(cambios) && cambios[j].Registrado.Equal(c.Registrado) {
			// La carga histórica puede registrar varios estados en un instante.
			// Sin un orden causal persistido, el UUID no permite elegir el final.
			if cambios[j].Estado != c.Estado {
				dato = d.Dato[string]{}
			}
			j++
		}
		agregarHistoria(&h, d.Revision[string]{ID: c.ID, Vigencia: d.Vigencia{Desde: c.Registrado}, RegistradoEn: c.Registrado, Dato: dato})
		i = j
	}
	return h
}
