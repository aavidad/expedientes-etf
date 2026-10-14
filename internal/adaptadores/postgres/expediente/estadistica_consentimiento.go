package expediente

import (
	"cmp"
	"slices"

	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

// La reapertura reinicia el consentimiento incluso si comparte fecha con la
// firma anterior. El orden de la cadena de episodios resuelve ese empate.
func historiaConsentimiento(episodios []episodioLectura, hitos []hitoLectura) d.Historia[bool] {
	padres := make(map[string]string, len(episodios))
	for _, ep := range episodios {
		padres[ep.ID] = ep.Reapertura
	}
	profundidad := func(id string) int {
		n := 0
		for padre := padres[id]; padre != "" && n < len(episodios); padre = padres[padre] {
			n++
		}
		return n
	}
	type evento struct {
		revision d.Revision[bool]
		episodio string
		firma    bool
	}
	eventos := make([]evento, 0, len(episodios)+len(hitos))
	for _, ep := range episodios {
		eventos = append(eventos, evento{revision: d.Revision[bool]{ID: ep.ID + ":consentimiento", Vigencia: d.Vigencia{Desde: ep.Apertura.Time}, RegistradoEn: ep.Creado}, episodio: ep.ID})
	}
	for _, h := range hitos {
		if h.Tipo == "consentimiento" {
			eventos = append(eventos, evento{revision: d.Revision[bool]{ID: h.EpisodioID + ":firma", Vigencia: d.Vigencia{Desde: h.Fecha.Time}, RegistradoEn: h.Confirmado, Dato: d.Dato[bool]{Valor: true, Conocido: true}}, episodio: h.EpisodioID, firma: true})
		}
	}
	slices.SortStableFunc(eventos, func(a, b evento) int {
		if c := a.revision.Vigencia.Desde.Compare(b.revision.Vigencia.Desde); c != 0 {
			return c
		}
		if c := cmp.Compare(profundidad(a.episodio), profundidad(b.episodio)); c != 0 {
			return c
		}
		if a.firma != b.firma {
			if a.firma {
				return 1
			}
			return -1
		}
		return a.revision.RegistradoEn.Compare(b.revision.RegistradoEn)
	})
	var historia d.Historia[bool]
	for _, e := range eventos {
		agregarHistoria(&historia, e.revision)
	}
	return historia
}
