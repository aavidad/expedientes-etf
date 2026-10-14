package documento

import (
	"context"
	"errors"
	"fmt"
	expediente "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"strings"
	"time"
)

// paraAnexo6 toma los datos del módulo propietario. Los apartados narrativos
// no forman parte del relleno, aunque el expediente tenga un motivo registrado.
func (f DatosEntrada) paraAnexo6(ctx context.Context, actor d.Actor, s d.Solicitud) (d.Datos, error) {
	if f.Salidas == nil || f.Catalogos == nil || f.Referencias == nil || f.Presentacion == nil || f.Reloj == nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	e, salida, err := f.Salidas.ParaAnexo6(ctx, expediente.Actor{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, s.ExpedienteID, s.SalidaID)
	if err != nil {
		if errors.Is(err, expediente.ErrAcceso) || errors.Is(err, expediente.ErrNoEncontrada) {
			return d.Datos{}, d.ErrAcceso
		}
		if errors.Is(err, expediente.ErrEstado) {
			return d.Datos{}, d.ErrPeticion
		}
		return d.Datos{}, d.ErrNoDisponible
	}
	if e.Numero == nil || e.EquipoID == nil {
		return d.Datos{}, d.ErrPeticion
	}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	ahora := f.Reloj.Ahora().In(zona)
	fecha := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC)
	if !s.FechaRepeticion.IsZero() {
		fecha = s.FechaRepeticion
	}
	v, err := f.Catalogos.Vigente(ctx, "asignacion_municipios", fecha)
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	cat, err := f.Catalogos.Datos(ctx, v)
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	ref, err := f.Referencias.Consultar(ctx, s.Idioma)
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	_, nombres, err := f.Presentacion.Cargar(ctx, s.IdiomaPaquete, ref)
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	campos := map[string][]string{}
	for _, c := range e.Datos.Extraccion.Campos {
		campos[c.Codigo] = append(campos[c.Codigo], c.Valor)
	}
	valor := func(k string, i int) string {
		if i < len(campos[k]) {
			return campos[k][i]
		}
		return ""
	}
	c := map[string]string{"numero": *e.Numero, "fecha_documento": fecha.Format("02/01/2006"), "servicio_destino": nombres["servicio_destino."+salida.Servicio], "direccion": valor("familia.direccion", 0), "entidad_derivante": ""}
	for _, eq := range cat.Equipos {
		if eq.ID == *e.EquipoID {
			c["equipo"] = nombres[eq.ClaveTexto]
		}
	}
	if c["servicio_destino"] == "" || c["equipo"] == "" {
		return d.Datos{}, d.ErrNoDisponible
	}
	// Los contactos del servicio de origen no se atribuyen al ETF derivante.
	c["profesional"], c["telefono_etf"], c["correo_etf"], c["fecha_entrada_sscc"] = "", "", "", ""
	for _, k := range []string{"estructura", "vivienda", "ingresos", "situacion_laboral", "situacion_economica", "tramo_ingresos"} {
		c[k] = valor("sociodemografia."+k, 0)
	}
	filas := map[string][]map[string]string{"contactos": {}, "menores": {}, "adultos": {}}
	for i, n := range campos["familia.contacto.nombre"] {
		filas["contactos"] = append(filas["contactos"], map[string]string{"nombre": n, "apellidos": valor("familia.contacto.apellidos", i), "telefono": valor("familia.contacto.telefono", i), "correo": valor("familia.contacto.correo", i)})
	}
	a := &d.Aceptacion{}
	for i, n := range campos["menores.persona.nombre"] {
		if strings.TrimSpace(n) == "" {
			continue
		}
		a.Menores = append(a.Menores, d.Menor{Persona: d.Persona{Nombre: n}})
		filas["menores"] = append(filas["menores"], map[string]string{"nombre": n, "nacimiento": valor("menores.persona.nacimiento", i), "escolarizacion": valor("menores.persona.escolarizacion", i), "centro": valor("menores.persona.centro", i)})
	}
	for i, n := range campos["sociodemografia.adulto.nombre"] {
		if strings.TrimSpace(n) != "" {
			filas["adultos"] = append(filas["adultos"], map[string]string{"papel": valor("sociodemografia.adulto.papel", i), "nombre": n, "parentesco": valor("sociodemografia.adulto.parentesco", i), "edad": valor("sociodemografia.adulto.edad", i), "estudios": valor("sociodemografia.adulto.estudios", i), "otros_datos": valor("sociodemografia.adulto.otros_datos", i)})
		}
	}
	for _, bloque := range []string{"contactos", "adultos"} {
		if len(filas[bloque]) == 0 {
			filas[bloque] = []map[string]string{{"papel": "", "nombre": "", "apellidos": "", "telefono": "", "correo": "", "parentesco": "", "edad": "", "estudios": "", "otros_datos": ""}}
		}
	}
	return d.Datos{ExpedienteID: e.ID, EquipoID: *e.EquipoID, FechaDocumento: fecha, Aceptacion: a, Salida: &d.SalidaDocumento{Campos: c, Filas: filas}, Versiones: map[string]string{"expediente": fmt.Sprint(e.Version), "salida_servicio": salida.ID + ":" + fmt.Sprint(salida.Version), "catalogo_destino": salida.CatalogoID, "catalogo_asignacion": v.ID, "textos_catalogo": fmt.Sprint(ref.Version) + ":" + ref.Huella, "lectura_anexo": e.Lectura.ID}}, nil
}
