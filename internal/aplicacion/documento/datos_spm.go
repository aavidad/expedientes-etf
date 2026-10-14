package documento

import (
	"fmt"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"time"
)

func fechaCivilSPM(t time.Time) string {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err == nil {
		t = t.In(zona)
	} //etf:tecnico
	return t.Format("2006-01-02")
}

// Los datos que todavía no existen en la ficha permanecen vacíos; no se
// completan con estimaciones ni con datos inventados por el servidor.
func rellenarDatosSPM(a *d.Aceptacion, campos map[string][]string) {
	valor := func(k string, i int) string {
		if i < len(campos[k]) {
			return campos[k][i]
		}
		return ""
	}
	a.CamposSPM = map[string]string{"fecha_documento": a.FechaActa, "fecha_entrada_sscc": valor("recepcion.fecha_entrada", 0), "entidad_derivante": a.ServicioDerivante, "equipo_menores": valor("servicio.equipo_menores", 0),
		"profesional": "", "profesional_telefono": "", "profesional_email": "",
		"estructura_familiar": valor("sociodemografia.estructura", 0), "convivencia_otras": valor("sociodemografia.otras_formas", 0),
		"vivienda": valor("sociodemografia.vivienda", 0), "situacion_laboral": valor("sociodemografia.situacion_laboral", 0), "situacion_economica": valor("sociodemografia.situacion_economica", 0), "ingresos": valor("sociodemografia.ingresos", 0), "tramo_ingresos": valor("sociodemografia.tramo_ingresos", 0)}
	a.FilasSPM = map[string][]map[string]string{"spm_contactos": {}, "spm_menores": {}, "spm_adultos": {}, "spm_significativas": {}}
	for i, n := range campos["familia.contacto.nombre"] {
		a.FilasSPM["spm_contactos"] = append(a.FilasSPM["spm_contactos"], map[string]string{"nombre": n, "apellidos": valor("familia.contacto.apellidos", i), "telefono": valor("familia.contacto.telefono", i), "email": valor("familia.contacto.correo", i)})
	}
	for i, m := range a.Menores {
		a.FilasSPM["spm_menores"] = append(a.FilasSPM["spm_menores"], map[string]string{"nombre": m.Nombre, "nacimiento": m.Nacimiento, "escolarizacion": valor("menores.persona.escolarizacion", i), "centro_educativo": valor("menores.persona.centro", i)})
	}
	for i, n := range campos["sociodemografia.adulto.nombre"] {
		a.FilasSPM["spm_adultos"] = append(a.FilasSPM["spm_adultos"], map[string]string{"orden": fmt.Sprint(i + 1), "nombre": n, "parentesco": valor("sociodemografia.adulto.parentesco", i), "edad": valor("sociodemografia.adulto.edad", i), "estudios": valor("sociodemografia.adulto.estudios", i), "otros": valor("sociodemografia.adulto.otros_datos", i)})
	}
	for i, n := range campos["sociodemografia.persona_significativa.nombre"] {
		a.FilasSPM["spm_significativas"] = append(a.FilasSPM["spm_significativas"], map[string]string{"nombre": n, "apellidos": valor("sociodemografia.persona_significativa.apellidos", i), "parentesco": valor("sociodemografia.persona_significativa.parentesco", i), "telefono": valor("sociodemografia.persona_significativa.telefono", i)})
	}

	// El Word conserva una fila vacía cuando no hay personas opcionales.
	camposVacios := map[string][]string{"spm_contactos": {"nombre", "apellidos", "telefono", "email"}, "spm_menores": {"nombre", "nacimiento", "escolarizacion", "centro_educativo"}, "spm_adultos": {"orden", "nombre", "parentesco", "edad", "estudios", "otros"}, "spm_significativas": {"nombre", "apellidos", "parentesco", "telefono"}}
	for coleccion, claves := range camposVacios {
		if len(a.FilasSPM[coleccion]) == 0 {
			fila := map[string]string{}
			for _, clave := range claves {
				fila[clave] = ""
			}
			a.FilasSPM[coleccion] = []map[string]string{fila}
		}
	}

}
