package estadistica

import (
	"cmp"
	"encoding/json"
	"slices"
	"time"
)

// AporteMemoria conserva las cantidades y dimensiones que aporta un expediente.
// No contiene nombres ni valores libres de la ficha.
type AporteMemoria struct {
	ExpedienteID string               `json:"expediente_id"`
	Numero       string               `json:"numero"`
	EquipoID     string               `json:"equipo_id"`
	Apartado     Apartado             `json:"apartado"`
	Total        Recuento             `json:"total"`
	Celdas       []CeldaMemoria       `json:"celdas"`
	Bases        map[Unidad]Dato[int] `json:"bases"`
}

// AportesMemoria usa el mismo motor y los mismos cortes que las cifras completas.
// La actividad de equipo (15 y 16) carece de relación con expedientes.
func AportesMemoria(e Entrada) ([]AporteMemoria, error) {
	r := []AporteMemoria{}
	for _, exp := range e.Expedientes {
		individual := e
		individual.Expedientes = []Expediente{exp}
		individual.Actividad = nil
		m, err := Calcular(individual)
		if err != nil {
			return nil, err
		}
		for codigo, ap := range m.Apartados {
			if codigo == ComisionesTecnicas || codigo == InformesTecnicos {
				continue
			}
			a := AporteMemoria{ExpedienteID: exp.ID, Numero: exp.Numero, EquipoID: exp.EquipoID, Apartado: codigo, Total: ap.Total, Celdas: []CeldaMemoria{}, Bases: map[Unidad]Dato[int]{}}
			for k, c := range ap.Celdas {
				if c.Denominador.Conocido {
					a.Bases[k.Unidad] = c.Denominador
				}
				if c.Numerador != 0 || c.SinDato != 0 || c.Denominador.Valor != 0 {
					a.Celdas = append(a.Celdas, CeldaMemoria{Grupo: k.Grupo, Unidad: k.Unidad, Celda: c})
				}
			}
			r = append(r, a)
		}
	}
	slices.SortFunc(r, func(a, b AporteMemoria) int {
		if c := cmp.Compare(a.ExpedienteID, b.ExpedienteID); c != 0 {
			return c
		}
		return cmp.Compare(a.Apartado, b.Apartado)
	})
	for i := range r {
		slices.SortFunc(r[i].Celdas, func(a, b CeldaMemoria) int {
			ab, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			return cmp.Compare(string(ab), string(bb))
		})
	}
	return r, nil
}

type SeleccionDesglose struct {
	Celda    int
	Medida   string
	Revision int
	Clave    *ClaveCelda
}
type FilaDesglose struct {
	ExpedienteID string `json:"expediente_id"`
	Numero       string `json:"numero"`
	EquipoID     string `json:"equipo_id"`
	Cantidad     int    `json:"cantidad"`
	Ficha        bool   `json:"ficha"`
}
type DetalleMemoria struct {
	Anio          int            `json:"anio"`
	EquipoID      string         `json:"equipo_id"`
	Apartado      Apartado       `json:"apartado"`
	Grupo         Grupo          `json:"grupo"`
	Unidad        Unidad         `json:"unidad"`
	Medida        string         `json:"medida"`
	Cifra         int            `json:"cifra"`
	PoblacionBase string         `json:"poblacion_base"`
	Filas         []FilaDesglose `json:"filas"`
	Aviso         string         `json:"aviso"`
	Definitivas   bool           `json:"definitivas"`
	Fecha         time.Time      `json:"fecha"`
}

func cantidadTotal(r Recuento, medida string) (int, bool) {
	switch medida {
	case "familias":
		return r.Familias, true
	case "menores":
		return r.Menores, true
	case "adultos":
		return r.Adultos, true
	case "comisiones":
		return r.Comisiones, true
	case "informes":
		return r.Informes, true
	case "familias_sin_relacion_menores":
		return r.FamiliasSinRelacionMenores, true
	}
	return 0, false
}
func cantidadCelda(c Celda, medida string) (int, bool) {
	switch medida {
	case "numerador":
		return c.Numerador, true
	case "sin_dato":
		return c.SinDato, true
	case "denominador":
		return c.Denominador.Valor, c.Denominador.Conocido
	}
	return 0, false
}

// SeleccionarDesglose devuelve una cifra concreta y verifica que sus aportes suman.
// Las fotos antiguas se declaran indisponibles, sin reconstruirlas con datos actuales.
func SeleccionarDesglose(informe InformeMemoria, codigo Apartado, s SeleccionDesglose) (DetalleMemoria, error) {
	r := DetalleMemoria{Anio: informe.Anio, EquipoID: informe.EquipoID, Apartado: codigo, Medida: s.Medida, Fecha: informe.ConocidoEn, Filas: []FilaDesglose{}}
	var ap *ApartadoMemoria
	for i := range informe.Apartados {
		if informe.Apartados[i].Codigo == codigo {
			ap = &informe.Apartados[i]
			break
		}
	}
	if ap == nil || s.Celda < -1 {
		return r, ErrEntrada
	}
	if s.Clave != nil {
		if s.Celda == -1 {
			return r, ErrEntrada
		}
		s.Celda = -2
		for i, c := range ap.Celdas {
			if c.Grupo == s.Clave.Grupo && c.Unidad == s.Clave.Unidad {
				s.Celda = i
				break
			}
		}
		if s.Celda == -2 {
			return r, ErrEntrada
		}
	}
	var valido bool
	if s.Celda == -1 {
		r.Cifra, valido = cantidadTotal(ap.Total, s.Medida)
		r.Unidad = Unidad(s.Medida)
	} else {
		if s.Celda >= len(ap.Celdas) {
			return r, ErrEntrada
		}
		c := ap.Celdas[s.Celda]
		r.Grupo, r.Unidad = c.Grupo, c.Unidad
		r.Cifra, valido = cantidadCelda(c.Celda, s.Medida)
	}
	if !valido {
		return r, ErrEntrada
	}
	if s.Medida == "denominador" {
		r.PoblacionBase = ap.Contrato.Denominador
	}
	if codigo == ComisionesTecnicas || codigo == InformesTecnicos {
		r.Aviso = "memoria.desglose.actividad"
		return r, nil
	}
	if informe.VersionDesglose != 1 {
		r.Aviso = "memoria.desglose.foto_antigua"
		return r, nil
	}
	suma := 0
	for _, a := range informe.Desglose {
		if a.Apartado != codigo {
			continue
		}
		n := 0
		if s.Celda == -1 {
			n, _ = cantidadTotal(a.Total, s.Medida)
		} else if s.Medida == "denominador" {
			n = a.Bases[r.Unidad].Valor
		} else {
			for _, c := range a.Celdas {
				if c.Grupo == r.Grupo && c.Unidad == r.Unidad {
					n, _ = cantidadCelda(c.Celda, s.Medida)
					break
				}
			}
		}
		if n == 0 {
			continue
		}
		r.Filas = append(r.Filas, FilaDesglose{ExpedienteID: a.ExpedienteID, Numero: a.Numero, EquipoID: a.EquipoID, Cantidad: n})
		suma += n
	}
	if suma != r.Cifra {
		r.Filas = []FilaDesglose{}
		r.Aviso = "memoria.desglose.no_reproducible"
	}
	slices.SortFunc(r.Filas, func(a, b FilaDesglose) int {
		if c := cmp.Compare(a.Numero, b.Numero); c != 0 {
			return c
		}
		return cmp.Compare(a.ExpedienteID, b.ExpedienteID)
	})
	return r, nil
}
