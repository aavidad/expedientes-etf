package estadistica

import (
	"cmp"
	"encoding/json"
	"slices"
	"time"
)

// CifrasConsulta es un contrato numérico independiente del informe completo.
// Sus tipos anidados enumeran campos: ampliar el informe no amplía esta salida.
type CifrasConsulta struct {
	EquipoCodigo string             `json:"equipo_codigo"`
	Anio         int                `json:"anio"`
	EquipoID     string             `json:"equipo_id"`
	Fecha        time.Time          `json:"fecha_cifras"`
	Corte        time.Time          `json:"corte"`
	Definitivas  bool               `json:"definitivas"`
	Apartados    []ApartadoConsulta `json:"apartados"`
}
type ApartadoConsulta struct {
	Codigo Apartado        `json:"codigo"`
	Total  TotalConsulta   `json:"total"`
	Celdas []CeldaConsulta `json:"celdas"`
	Avisos map[string]int  `json:"avisos"`
}
type TotalConsulta struct {
	Familias                   int `json:"familias"`
	Menores                    int `json:"menores"`
	Adultos                    int `json:"adultos"`
	Comisiones                 int `json:"comisiones"`
	Informes                   int `json:"informes"`
	FamiliasSinRelacionMenores int `json:"familias_sin_relacion_menores"`
}
type GrupoConsulta struct {
	Subprograma Subprograma `json:"subprograma"`
	Origen      string      `json:"origen"`
	Sexo        Sexo        `json:"sexo"`
	Tramo       string      `json:"tramo"`
	Motivo      Motivo      `json:"motivo"`
	Papel       string      `json:"papel"`
	Estado      string      `json:"estado"`
	Causa       string      `json:"causa"`
	Organismo   string      `json:"organismo"`
}
type CeldaConsulta struct {
	AvisoPorcentaje string        `json:"aviso_porcentaje"`
	Grupo           GrupoConsulta `json:"grupo"`
	Unidad          Unidad        `json:"unidad"`
	Numerador       int           `json:"numerador"`
	Denominador     *int          `json:"denominador"`
	Porcentaje      *int          `json:"porcentaje_centesimas"`
	SinDato         int           `json:"sin_dato"`
}

func AutorizarConsultaMemoria(a ActorMemoria, f FiltroMemoria) error {
	if !UUIDValidoSPAF(a.UsuarioID) || !slices.Contains(a.Perfiles, "consulta") {
		return ErrPermisoMemoria
	}
	if f.Anio < 1 || f.Anio > 9998 || (f.EquipoID != "" && !UUIDValidoSPAF(f.EquipoID)) {
		return ErrEntrada
	}
	return nil
}

// ProyectarCifras recibe el resultado del motor, nunca narrativas ni fuentes.
func ProyectarCifras(m Memoria, equipo string) CifrasConsulta {
	r := CifrasConsulta{Anio: m.Anio, EquipoID: equipo, Fecha: m.ConocidoEn, Corte: m.Corte, Apartados: []ApartadoConsulta{}}
	for codigo, a := range m.Apartados {
		ap := ApartadoConsulta{Codigo: codigo, Total: TotalConsulta{a.Total.Familias, a.Total.Menores, a.Total.Adultos, a.Total.Comisiones, a.Total.Informes, a.Total.FamiliasSinRelacionMenores}, Avisos: AvisosConsulta(a.Avisos), Celdas: []CeldaConsulta{}}
		for k, c := range a.Celdas {
			g := k.Grupo
			v := CeldaConsulta{Grupo: GrupoConsulta{g.Subprograma, g.Origen, g.Sexo, g.Tramo, g.Motivo, g.Papel, g.Estado, g.Causa, g.Organismo}, Unidad: k.Unidad, Numerador: c.Numerador, SinDato: c.SinDato}
			if c.Denominador.Conocido {
				n := c.Denominador.Valor
				v.Denominador = &n
			}
			if c.Porcentaje != nil {
				v.AvisoPorcentaje = AvisoPorcentajeConsulta(c.Porcentaje.Aviso)
			}
			if c.Porcentaje != nil && (c.Porcentaje.Aviso == "" || c.Porcentaje.Aviso == "estadistica.divisor_cero") {
				n := c.Porcentaje.Centesimas
				v.Porcentaje = &n
			}
			ap.Celdas = append(ap.Celdas, v)
		}
		slices.SortFunc(ap.Celdas, func(a, b CeldaConsulta) int {
			ab, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			return cmp.Compare(string(ab), string(bb))
		})
		r.Apartados = append(r.Apartados, ap)
	}
	slices.SortFunc(r.Apartados, func(a, b ApartadoConsulta) int { return cmp.Compare(a.Codigo, b.Codigo) })
	return r
}

type VistaMemoria struct {
	FiltroMemoria
	Fecha       time.Time         `json:"fecha_cifras"`
	Definitivas bool              `json:"definitivas"`
	Apartados   []ApartadoMemoria `json:"apartados"`
}
type DesgloseMemoria struct {
	Apartado Apartado `json:"apartado"`
	Grupo    Grupo    `json:"grupo"`
	Unidad   Unidad   `json:"unidad"`
	Numeros  []string `json:"numeros"`
}

// AvisosConsulta sólo admite claves del contrato, nunca textos como claves.
func AvisosConsulta(avisos map[string]int) map[string]int {
	r := map[string]int{}
	for k, v := range avisos {
		switch k {
		case "estadistica.aceptacion_sin_dato", "estadistica.motivo_sin_dato", "estadistica.recuento_actividad_sin_dato", "estadistica.seguimiento_subprograma_sin_dato", "estadistica.subprograma_sin_dato", "estadistica.varios_cierres_pendiente", "estadistica.denominador_pendiente", "estadistica.divisor_cero", "estadistica.no_aceptados_sin_equipo", "estadistica.poblacion_parental_sin_dato", "estadistica.catalogo_dimension_sin_dato":
			r[k] = v
		}
	}
	return r
}

func AvisoPorcentajeConsulta(aviso string) string {
	switch aviso {
	case "estadistica.divisor_cero", "estadistica.denominador_pendiente":
		return aviso
	}
	return ""
}
