package documento

import (
	"fmt"
	expediente "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"strings"
	"time"
)

// La ficha vigente sustituye la lectura inicial, incluidos los vacíos guardados.
func rellenarFichaAnexo6(f *expediente.FichaParaAnexo6, c map[string]string, filas map[string][]map[string]string, nombres map[string]string, fecha time.Time, separadorDecimal string) {
	if f == nil {
		return
	}
	etiqueta := func(lista string, codigo *string) string {
		if codigo == nil {
			return ""
		}
		return nombres[lista+"."+*codigo]
	}
	dia := func(valor *string) string {
		if valor == nil {
			return ""
		}
		t, err := time.Parse("2006-01-02", *valor)
		if err != nil {
			return ""
		}
		return t.Format("02/01/2006")
	}
	if f.Familia != nil {
		a := f.Familia
		c["direccion"] = a.Direccion
		c["estructura"] = etiqueta("composicion_familiar", a.Composicion)
		if a.ComposicionOtras != "" {
			c["estructura"] += " " + a.ComposicionOtras
		}
		c["vivienda"] = etiqueta("vivienda", a.Vivienda)
		c["situacion_economica"] = etiqueta("situacion_economica", a.Economia)
		c["tramo_ingresos"] = etiqueta("tramo_ingresos_smi", a.Tramo)
		c["situacion_laboral"], c["ingresos"] = "", ""
		if a.Ingresos != nil {
			c["ingresos"] = fmt.Sprint(*a.Ingresos/100) + separadorDecimal + fmt.Sprintf("%02d", *a.Ingresos%100)
		}
		filas["contactos"] = []map[string]string{{"nombre": "", "apellidos": "", "telefono": a.Telefono, "correo": a.Email}}
	}
	if f.Adultos != nil {
		filas["adultos"] = []map[string]string{}
		filas["contactos"] = nil
		for _, a := range *f.Adultos {
			edad := ""
			if a.Nacimiento != nil {
				n, err := time.Parse("2006-01-02", *a.Nacimiento)
				if err == nil {
					años := fecha.Year() - n.Year()
					if fecha.Month() < n.Month() || fecha.Month() == n.Month() && fecha.Day() < n.Day() {
						años--
					}
					edad = fmt.Sprint(años)
				}
			} else if a.Edad != nil {
				edad = fmt.Sprint(*a.Edad)
			}
			papel := a.Papel
			filas["adultos"] = append(filas["adultos"], map[string]string{"papel": etiqueta("papel_parental", &papel), "nombre": strings.TrimSpace(a.Nombre + " " + a.Apellidos), "parentesco": etiqueta("papel_parental", &papel), "edad": edad, "estudios": etiqueta("nivel_educativo", a.Educacion), "otros_datos": a.OtrosDatos})
			if a.Telefono != "" || a.Email != "" {
				filas["contactos"] = append(filas["contactos"], map[string]string{"nombre": a.Nombre, "apellidos": a.Apellidos, "telefono": a.Telefono, "correo": a.Email})
			}
		}
		if f.Familia != nil && (f.Familia.Telefono != "" || f.Familia.Email != "") {
			filas["contactos"] = append(filas["contactos"], map[string]string{"nombre": "", "apellidos": "", "telefono": f.Familia.Telefono, "correo": f.Familia.Email})
		}
	}
	iniciales := filas["menores"]
	filas["menores"] = []map[string]string{}
	for _, m := range f.Menores {
		fila := map[string]string{"nombre": strings.TrimSpace(m.Nombre + " " + m.Apellidos), "nacimiento": dia(m.Nacimiento), "escolarizacion": etiqueta("nivel_escolarizacion", m.Escolarizacion), "centro": m.CentroEducativo}
		if m.DatosDesdeAnexo && m.Orden > 0 && m.Orden <= len(iniciales) {
			fila = iniciales[m.Orden-1]
		}
		filas["menores"] = append(filas["menores"], fila)
	}
}
