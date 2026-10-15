package documento

import (
	"fmt"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"strconv"
	"strings"
	"time"
)

// rellenarSPAF sólo presenta la tabla calculada; no decide inclusiones ni totales.
func rellenarSPAF(t est.TablaSPAF, equipo string, textos, nombres map[string]string) (d.Relleno, error) {
	sinDato := textos["documento.spaf.sin_dato"]
	anexo := textos["documento.spaf.anexo"]
	acta := textos["documento.spaf.acta"]
	formato := textos["documento.spaf.fecha_formato"]
	if sinDato == "" || anexo == "" || acta == "" || formato == "" {
		return d.Relleno{}, d.ErrNoDisponible
	}
	r := d.Relleno{Campos: map[string]string{"equipo": equipo, "mes": strconv.Itoa(t.Mes), "anio": strconv.Itoa(t.Anio), "altas": strconv.Itoa(t.Resumen.Altas), "bajas": strconv.Itoa(t.Resumen.Bajas), "casos": strconv.Itoa(t.Resumen.Casos), "menores": strconv.Itoa(t.Resumen.Menores)}, Filas: map[string][]map[string]string{"casos_spaf": {}}}
	var fallo error
	etiqueta := func(lista, codigo string) string {
		if codigo == "" {
			return sinDato
		}
		clave := lista + "." + codigo
		if strings.HasPrefix(codigo, lista+".") {
			clave = codigo
		}
		if v := nombres[clave]; v != "" {
			return v
		}
		fallo = d.ErrNoDisponible
		return ""
	}
	dato := func(v est.Dato[string], lista string) string {
		if !v.Conocido {
			return sinDato
		}
		if lista == "" {
			return v.Valor
		}
		return etiqueta(lista, v.Valor)
	}
	lista := func(v est.Dato[[]string], tipo string) string {
		if !v.Conocido {
			return sinDato
		}
		valores := []string{}
		for _, codigo := range v.Valor {
			valores = append(valores, etiqueta(tipo, codigo))
		}
		return strings.Join(valores, "; ")
	}
	fecha := func(v est.Dato[time.Time]) string {
		if !v.Conocido {
			return sinDato
		}
		return v.Valor.Format(formato)
	}
	for _, f := range t.Filas {
		menores := []string{}
		centros := []string{}
		if f.Menores.Conocido {
			for _, m := range f.Menores.Valor {
				menores = append(menores, strings.TrimSpace(m.Apellidos+", "+m.Nombre)+" / "+fecha(m.Nacimiento))
				if m.CentroProteccion.Conocido && strings.TrimSpace(m.CentroProteccion.Valor) != "" {
					centros = append(centros, m.CentroProteccion.Valor)
				}
			}
		} else {
			menores = append(menores, sinDato)
		}
		guardadores := []string{}
		todosEnCentro := f.Menores.Conocido && len(f.Menores.Valor) > 0 && len(centros) == len(f.Menores.Valor)
		if !todosEnCentro {
			if f.Guardadores.Conocido {
				for _, g := range f.Guardadores.Valor {
					guardadores = append(guardadores, g.Nombre+" / "+etiqueta("papel_parental", g.Parentesco))
				}
			} else {
				guardadores = append(guardadores, sinDato)
			}
		}
		guardadores = append(guardadores, centros...)

		numero := sinDato
		if f.NumeroMenores.Conocido {
			numero = strconv.Itoa(f.NumeroMenores.Valor)
		}
		derivacion := sinDato
		if f.Derivacion.Conocido {
			derivacion = fmt.Sprintf(anexo, f.Derivacion.Valor.Anexo) + " / " + f.Derivacion.Valor.Fecha.Format(formato)
		}
		fechas := []string{acta + " / " + fecha(f.Aceptacion)}
		for _, a := range f.FechasAnexos {
			fechas = append(fechas, fmt.Sprintf(anexo, a.Anexo)+" / "+a.Fecha.Format(formato))
		}
		tiempo := sinDato
		if f.TemporalizacionMeses.Conocido {
			tiempo = strconv.Itoa(f.TemporalizacionMeses.Valor)
		}
		cierres := []string{}
		for _, c := range f.Cierres {
			cierres = append(cierres, c.Fecha.Format(formato)+" / "+dato(c.Motivo, "motivo_spaf"))
		}
		r.Filas["casos_spaf"] = append(r.Filas["casos_spaf"], map[string]string{"numero": f.Numero, "menores": strings.Join(menores, "; "), "numero_menores": numero, "guardadores": strings.Join(guardadores, "; "), "municipio_zts": dato(f.Municipio, "municipio"), "caracteristicas": lista(f.CaracteristicasFamilia, "caracteristica_familia"), "derivacion": derivacion, "servicio": dato(f.ServicioDerivante, "servicio_derivante") + " / " + dato(f.CoordinadorCaso, ""), "fechas": strings.Join(fechas, "; "), "fase": dato(f.FaseIntervencion, "fase_intervencion"), "temporalizacion": tiempo, "finalidad_pronostico": dato(f.Finalidad, "finalidad") + " / " + dato(f.Pronostico, "pronostico"), "instrumentos": lista(f.Instrumentos, "instrumento"), "cierres": strings.Join(cierres, "; ")})
	}
	return r, fallo
}
