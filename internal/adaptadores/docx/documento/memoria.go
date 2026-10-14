package documento

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"maps"
	"regexp"
	"strconv"
	"strings"
)

func (Motor) CamposMemoria(ctx context.Context, informe e.InformeMemoria, p d.PresentacionMemoria) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.Campos["sin_dato"] == "" || p.Campos["separador_decimal"] == "" {
		return nil, d.ErrPlantilla
	}
	campos := maps.Clone(p.Campos)
	n := informe.Narrativas
	campos["anio"] = strconv.Itoa(informe.Anio)
	campos["equipo"] = informe.EquipoCodigo
	if informe.EquipoID == "" {
		campos["equipo"] = campos["provincial"]
	}
	campos["a_entidad"] = n.A.EntidadLocal
	campos["a_equipos"] = contarManual(n.A.EquiposConvenio, campos["sin_dato"])
	campos["a_denominacion"] = n.A.Denominacion
	campos["a_observaciones"] = n.A.Observaciones
	for _, v := range []struct {
		codigo string
		p      e.PersonalMemoria
	}{{"a_psicologia", n.A.Psicologia}, {"a_trabajo_social", n.A.TrabajoSocial}, {"a_educacion_social", n.A.EducacionSocial}} {
		campos[v.codigo+"_hombres"] = contarManual(v.p.Hombres, campos["sin_dato"])
		campos[v.codigo+"_mujeres"] = contarManual(v.p.Mujeres, campos["sin_dato"])
	}
	campos["narrativa_d"], campos["narrativa_e"], campos["narrativa_f"], campos["narrativa_g"] = n.D, n.E, n.F, n.G
	apartados := map[string]e.ApartadoMemoria{}
	for _, a := range informe.Apartados {
		apartados[string(a.Codigo)] = a
		sum := 0
		for _, c := range a.Celdas {
			sum += c.Celda.SinDato
		}
		campos["sin_dato_"+strings.ReplaceAll(string(a.Codigo), ".", "_")] = strconv.Itoa(sum)
		campos["sin_relacion_"+strings.ReplaceAll(string(a.Codigo), ".", "_")] = strconv.Itoa(a.Total.FamiliasSinRelacionMenores)
		if a.Avisos["estadistica.divisor_cero"] > 0 {
			campos["aviso_"+strings.ReplaceAll(string(a.Codigo), ".", "_")] = campos["aviso_divisor_cero"]
		} else if len(a.Avisos) > 0 {
			campos["aviso_"+strings.ReplaceAll(string(a.Codigo), ".", "_")] = campos["aviso_datos_incompletos"]
		} else {
			campos["aviso_"+strings.ReplaceAll(string(a.Codigo), ".", "_")] = ""
		}
	}
	for _, spec := range p.Celdas {
		a, ok := apartados[spec.Apartado]
		if !ok {
			return nil, d.ErrPlantilla
		}
		valor, sinDato := 0, a.Avisos["estadistica.catalogo_dimension_sin_dato"] > 0 || a.Avisos["estadistica.poblacion_parental_sin_dato"] > 0 || a.Avisos["estadistica.no_aceptados_sin_equipo"] > 0
		seleccion := []e.Celda{}
		for _, c := range a.Celdas {
			if string(c.Unidad) != spec.Unidad {
				continue
			}
			g := map[string]string{"subprograma": string(c.Grupo.Subprograma), "origen": c.Grupo.Origen, "sexo": string(c.Grupo.Sexo), "tramo": c.Grupo.Tramo, "motivo": string(c.Grupo.Motivo), "papel": c.Grupo.Papel, "estado": c.Grupo.Estado, "causa": c.Grupo.Causa, "organismo": c.Grupo.Organismo}
			coincide, desconocido := true, true
			for k, v := range spec.Filtros {
				actual, existe := g[k]
				if !existe {
					return nil, d.ErrPlantilla
				}
				if actual != v {
					coincide = false
					if actual != "" {
						desconocido = false
					}
				}
			}
			if c.Celda.SinDato > 0 && (coincide || desconocido) {
				sinDato = true
			}
			if coincide {
				valor += c.Celda.Numerador
				seleccion = append(seleccion, c.Celda)
			}
		}
		if spec.Unidad == "menores" && a.Total.FamiliasSinRelacionMenores > 0 {
			sinDato = true
		}
		if spec.Total {
			switch spec.Unidad {
			case "familias":
				valor = a.Total.Familias
			case "menores":
				valor = a.Total.Menores
			case "adultos":
				valor = a.Total.Adultos
			case "comisiones":
				valor = a.Total.Comisiones
			case "informes":
				valor = a.Total.Informes
			default:
				return nil, d.ErrPlantilla
			}
		}
		resultado := strconv.Itoa(valor)
		if spec.Porcentaje {
			resultado = campos["sin_dato"]
			if len(seleccion) == 1 && seleccion[0].Porcentaje != nil && seleccion[0].Porcentaje.Aviso == "" {
				n := seleccion[0].Porcentaje.Centesimas
				resultado = strconv.Itoa(n/100) + campos["separador_decimal"] + digitos(n%100)
			} else if len(seleccion) == 1 && seleccion[0].Porcentaje != nil && seleccion[0].Porcentaje.Aviso == "estadistica.divisor_cero" {
				resultado = "0" + campos["separador_decimal"] + "00"
			}
		}
		if sinDato {
			resultado = campos["sin_dato"]
		}
		campos[spec.Campo] = resultado
	}
	// Se expanden sólo las referencias de los textos fijos. Las narrativas se
	// insertan como valores: sus llaves nunca se interpretan como instrucciones.
	marcas := regexp.MustCompile(`\{\{([a-z][a-z0-9_.]*)\}\}`)
	for clave, texto := range p.Campos {
		fallo := false
		campos[clave] = marcas.ReplaceAllStringFunc(texto, func(marca string) string {
			nombre := marca[2 : len(marca)-2]
			valor, ok := campos[nombre]
			if !ok {
				fallo = true
			}
			return valor
		})
		if fallo {
			return nil, d.ErrPlantilla
		}
	}

	return campos, nil
}
func digitos(n int) string {
	s := strconv.Itoa(n)
	if n < 10 {
		s = "0" + s
	}
	return s
}

func contarManual(n *int, sinDato string) string {
	if n == nil {
		return sinDato
	}
	return strconv.Itoa(*n)
}
