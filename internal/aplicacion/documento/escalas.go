package documento

import (
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	ev "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	"sort"
	"strconv"
	"strings"
)

// rellenarEscalas copia resultados anotados. No suma ítems ni aplica baremos.
func rellenarEscalas(a *d.Aceptacion, datos ev.DatosDocumento, nombres, textos, versiones map[string]string) {
	if datos.Registro.Definiciones == nil {
		return
	}
	definiciones := map[string]ev.Escala{}
	for _, escala := range datos.Registro.Definiciones.Escalas {
		definiciones[escala.ID] = escala
	}
	aplicaciones := append([]ev.Aplicacion(nil), datos.Registro.Escalas...)
	sort.Slice(aplicaciones, func(i, j int) bool { return aplicaciones[i].ID < aplicaciones[j].ID })
	valores := map[string][]string{}
	for _, aplicacion := range aplicaciones {
		if aplicacion.Momento != "inicial" && aplicacion.Momento != "intermedia" && aplicacion.Momento != "final" {
			continue
		}
		escala := definiciones[aplicacion.VersionEscalaID]
		campos := map[string]string{}
		campoDescripcion := ""
		for _, item := range escala.Items {
			campos[item.Codigo] = item.CampoDocumento
			if campoDescripcion == "" && strings.HasPrefix(item.CampoDocumento, "escala_") {
				if indice := strings.LastIndex(item.CampoDocumento, "_"); indice > 0 {
					campoDescripcion = item.CampoDocumento[:indice] + "_descripcion"
				}
			}
		}
		idSujeto := aplicacion.Sujeto.AdultoID
		if aplicacion.Sujeto.MenorID != "" {
			idSujeto = aplicacion.Sujeto.MenorID
		}
		if aplicacion.Sujeto.ProfesionalID != "" {
			idSujeto = aplicacion.Sujeto.ProfesionalID
		}
		identificar := func(valor string) string {
			if inicial := datos.Iniciales[idSujeto]; inicial != "" {
				return inicial + ": " + valor
			}
			return valor
		}
		for _, puntuacion := range aplicacion.Puntuaciones {
			campo := campos[puntuacion.Item]
			if !strings.HasPrefix(campo, "escala_") {
				continue
			}
			valor := ""
			if puntuacion.ValorNumerico != nil {
				valor = strconv.FormatFloat(*puntuacion.ValorNumerico, 'f', -1, 64)
				if decimal := textos["documento.escala.separador_decimal"]; decimal != "" {
					valor = strings.ReplaceAll(valor, ".", decimal)
				}
			} else if puntuacion.ValorCodigo != nil {
				valor = nombres["evaluacion.respuesta."+strings.ToLower(aplicacion.Instrumento)+"."+*puntuacion.ValorCodigo]
			}
			if valor != "" {
				clave := campo + "_" + aplicacion.Momento
				valores[clave] = append(valores[clave], identificar(valor))
			}
		}
		if aplicacion.DescripcionCualitativa != "" && campoDescripcion != "" {
			valor := aplicacion.DescripcionCualitativa
			if momento := textos["fi2.momento."+aplicacion.Momento]; momento != "" {
				valor = momento + ": " + valor
			}
			valores[campoDescripcion] = append(valores[campoDescripcion], identificar(valor))
		}
		versiones["evaluacion_"+aplicacion.ID] = strconv.Itoa(aplicacion.Version)
		versiones["escala_"+escala.ID] = strconv.Itoa(escala.Version)
	}
	for clave, resultados := range valores {
		a.Finalizacion[clave] = strings.Join(resultados, "; ")
	}
}
