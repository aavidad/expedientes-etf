package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
)

// CamposCifras reutiliza sólo el formato y las celdas de la plantilla oficial.
// La reconstrucción enumera valores numéricos; nunca recibe el informe original.
func (m Motor) CamposCifras(ctx context.Context, r e.CifrasConsulta, p d.PresentacionMemoria) (map[string]string, error) {
	informe := e.InformeMemoria{FiltroMemoria: e.FiltroMemoria{Anio: r.Anio, EquipoID: r.EquipoID}}
	// El ámbito se expresa por su código de equipo, no por un nombre libre.
	informe.EquipoCodigo = r.EquipoCodigo
	for _, a := range r.Apartados {
		t := a.Total
		ap := e.ApartadoMemoria{Codigo: a.Codigo, Total: e.Recuento{Familias: t.Familias, Menores: t.Menores, Adultos: t.Adultos, Comisiones: t.Comisiones, Informes: t.Informes, FamiliasSinRelacionMenores: t.FamiliasSinRelacionMenores}, Avisos: a.Avisos}
		for _, v := range a.Celdas {
			g := v.Grupo
			c := e.Celda{Numerador: v.Numerador, SinDato: v.SinDato}
			if v.Denominador != nil {
				c.Denominador = e.Dato[int]{Valor: *v.Denominador, Conocido: true}
			}
			if v.Porcentaje != nil {
				c.Porcentaje = &e.Porcentaje{Centesimas: *v.Porcentaje, Aviso: v.AvisoPorcentaje}
			}
			ap.Celdas = append(ap.Celdas, e.CeldaMemoria{Grupo: e.Grupo{Subprograma: g.Subprograma, Origen: g.Origen, Sexo: g.Sexo, Tramo: g.Tramo, Motivo: g.Motivo, Papel: g.Papel, Estado: g.Estado, Causa: g.Causa, Organismo: g.Organismo}, Unidad: v.Unidad, Celda: c})
		}
		informe.Apartados = append(informe.Apartados, ap)
	}
	campos, err := m.CamposMemoria(ctx, informe, p)
	if err != nil {
		return nil, err
	}
	for _, clave := range []string{"a_entidad", "a_equipos", "a_denominacion", "a_observaciones", "a_psicologia_hombres", "a_psicologia_mujeres", "a_trabajo_social_hombres", "a_trabajo_social_mujeres", "a_educacion_social_hombres", "a_educacion_social_mujeres", "narrativa_d", "narrativa_e", "narrativa_f", "narrativa_g"} {
		campos[clave] = ""
	}
	campos["texto_1"] += "\n" + p.Campos["estadistica.consulta_fecha"] + " " + r.Fecha.UTC().Format("2006-01-02T15:04:05Z")
	estado := "estadistica.consulta_provisional"
	if r.Definitivas {
		estado = "estadistica.consulta_definitiva"
	}
	campos["texto_1"] += "\n" + p.Campos[estado]
	for _, clave := range []string{"cabecera_a", "datos_a", "personal_a", "observaciones_a", "cabecera_d", "cabecera_e", "cabecera_f", "cabecera_g"} {
		campos[clave] = ""
	}
	return campos, nil
}
