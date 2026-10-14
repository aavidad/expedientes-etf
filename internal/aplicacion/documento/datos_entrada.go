package documento

import (
	"context"
	"errors"
	"fmt"
	expediente "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	familia "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	p "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	paquete "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
	"strings"
	"time"
)

// DatosEntrada combina puertos de los módulos propietarios. Todas las lecturas
// de base reciben la transacción que ya abrió Emitir; ninguna abre otra.
type DatosEntrada struct {
	Expedientes  expediente.DatosDocumentos
	Salidas      expediente.SalidasServicio
	Catalogos    catalogo.Repositorio
	Referencias  paquete.Referencias
	Presentacion salida.Presentacion
	Reloj        salida.Reloj
}

func (f DatosEntrada) Obtener(ctx context.Context, actor d.Actor, s d.Solicitud) (d.Datos, error) {
	if s.Tipo == d.Anexo6 {
		return f.paraAnexo6(ctx, actor, s)
	}
	if f.Expedientes == nil || f.Catalogos == nil || f.Referencias == nil || f.Presentacion == nil || f.Reloj == nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	instante := f.Reloj.Ahora()
	e, err := f.Expedientes.ParaDocumentos(ctx, expediente.Actor{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, s.ExpedienteID)
	if err != nil {
		if errors.Is(err, expediente.ErrNoEncontrada) || errors.Is(err, expediente.ErrAcceso) {
			return d.Datos{}, d.ErrAcceso
		}
		return d.Datos{}, d.ErrNoDisponible
	}
	finalizacion := d.EsFinalizacion(s.Tipo)
	if finalizacion && (e.EquipoID == nil || *e.EquipoID != actor.EquipoID) {
		return d.Datos{}, d.ErrAcceso
	}
	if (finalizacion || s.Tipo == d.ActaCierre) && e.Version != s.Version {
		return d.Datos{}, d.ConflictoVersion{VersionActual: e.Version}
	}
	origen := e.Datos.Extraccion.Tipo
	if e.ServicioDerivante == "OTRO_ETF" {
		origen = "etf"
	}
	aceptada := s.Tipo != d.ActaNoAceptacion && s.Tipo != d.OficioListaEspera
	var fechaTexto string
	var decision *d.DecisionDocumento
	switch s.Tipo {
	case d.ActaCierre:
		if e.ActaCierre == nil || e.ActaCierre.PropuestaPor == actor.UsuarioID {
			return d.Datos{}, d.ErrAcceso
		}
		if e.Estado != "aceptada" && e.Estado != "en_intervencion" && e.Estado != "derivada_spm" && e.Estado != "cerrada" {
			return d.Datos{}, d.ErrPeticion
		}
		if e.Aceptacion == nil || e.Numero == nil || e.EquipoID == nil || e.MunicipioID == nil {
			return d.Datos{}, d.ErrPeticion
		}
		fechaTexto = e.Aceptacion.FechaActa
	case d.Anexo11, d.Anexo12:
		if (e.Estado != "aceptada" && e.Estado != "en_intervencion" && e.Estado != "derivada_spm") || e.Aceptacion == nil || e.Numero == nil || e.EquipoID == nil || e.MunicipioID == nil || e.Finalizacion == nil {
			return d.Datos{}, d.ErrPeticion
		}
		if s.Tipo == d.Anexo11 && e.Aceptacion.Subprograma != "RIESGO_SOCIAL" || s.Tipo == d.Anexo12 && e.Aceptacion.Subprograma != "REUNIFICACION" {
			return d.Datos{}, d.ErrPeticion
		}
		fechaTexto = e.Aceptacion.FechaActa
	case d.ActaNoAceptacion:
		if e.Estado != "no_aceptada" {
			return d.Datos{}, d.ErrNoAceptada
		}
		if e.NoAceptacion == nil || e.NoAceptacion.Motivo == "" || e.NoAceptacion.Subprograma == "" {
			return d.Datos{}, d.ErrPeticion
		}
		fechaTexto = e.NoAceptacion.Fecha
		decision = &d.DecisionDocumento{Motivo: e.NoAceptacion.Motivo, Subprograma: e.NoAceptacion.Subprograma}
	case d.OficioListaEspera:
		if e.Estado != "lista_espera" {
			return d.Datos{}, d.ErrListaEspera
		}
		if e.FechaEspera == nil || origen != "anexo3" {
			return d.Datos{}, d.ErrPeticion
		}
		fechaTexto = *e.FechaEspera
		decision = &d.DecisionDocumento{}
	default:
		if (e.Estado != "aceptada" && !((d.EsConsentimiento(s.Tipo) || s.Tipo == d.Anexo5) && e.Estado == "en_intervencion")) || e.Aceptacion == nil || e.Numero == nil || e.EquipoID == nil || e.MunicipioID == nil {
			return d.Datos{}, d.ErrPeticion
		}
		fechaTexto = e.Aceptacion.FechaActa
		if s.Tipo == d.Anexo5 {
			fechaTexto = fechaCivilSPM(f.Reloj.Ahora())
			if !s.FechaReproduccion.IsZero() {
				fechaTexto = s.FechaReproduccion.Format("2006-01-02")
			}

		}
		if d.EsConsentimiento(s.Tipo) && ((s.Tipo == d.Anexo1 && e.Aceptacion.Subprograma != "RIESGO_SOCIAL") || (s.Tipo == d.Anexo2 && e.Aceptacion.Subprograma != "REUNIFICACION")) {
			return d.Datos{}, d.ErrPeticion
		}
	}
	if s.Tipo == d.OficioSSCC && origen != "anexo3" || s.Tipo == d.OficioSPM && origen != "anexo4" {
		return d.Datos{}, d.ErrPeticion
	}
	fecha, err := time.Parse("2006-01-02", fechaTexto)
	if err != nil {
		return d.Datos{}, d.ErrPeticion
	}
	if decision != nil {
		recepcion, err := time.Parse("2006-01-02", e.FechaRecepcion)
		if err != nil {
			return d.Datos{}, d.ErrPeticion
		}
		decision.Fecha = fecha.Format("02/01/2006")
		decision.FechaDerivacion = recepcion.Format("02/01/2006")
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
	if ref.Codigo == "" {
		ref = p.ReferenciaCatalogo{Codigo: s.Idioma}
	}
	textos, nombres, err := f.Presentacion.Cargar(ctx, s.IdiomaPaquete, ref)
	if err != nil {
		return d.Datos{}, d.ErrNoDisponible
	}
	a := &d.Aceptacion{Decision: decision, FechaActa: fecha.Format("02/01/2006"), MarcaSeleccion: textos["documento.marca_seleccion"], RotuloMenor: textos["documento.rotulo_menor"], Selecciones: map[string]string{}}
	if aceptada {
		a.Numero = *e.Numero
		a.Descripcion = e.Aceptacion.Descripcion
	}
	for _, eq := range cat.Equipos {
		a.Selecciones["equipo_"+eq.Codigo] = ""
		if aceptada && eq.ID == *e.EquipoID {
			a.Equipo = nombres[eq.ClaveTexto]
			a.Selecciones["equipo_"+eq.Codigo] = a.MarcaSeleccion
		}
	}
	for _, c := range cat.Centros {
		a.Selecciones["centro_"+strings.ToLower(c.Codigo)] = ""
		if aceptada && c.ID == e.Aceptacion.CentroID {
			a.Centro = nombres[c.ClaveTexto]
			a.Selecciones["centro_"+strings.ToLower(c.Codigo)] = a.MarcaSeleccion
		}
	}
	for _, m := range cat.Municipios {
		if e.MunicipioID != nil && m.ID == *e.MunicipioID {
			a.Municipio = nombres[m.ClaveTexto]
		}
	}
	if aceptada && (a.Equipo == "" || a.Centro == "" || a.Municipio == "") || a.MarcaSeleccion == "" || a.RotuloMenor == "" {
		return d.Datos{}, d.ErrNoDisponible
	}
	for _, puesto := range []string{"TS", "ES", "PS"} {
		a.Selecciones["profesional_"+strings.ToLower(puesto)] = ""
	}
	if aceptada {
		a.Profesional = textos["documento.puesto."+strings.ToLower(e.Aceptacion.ProfesionalPuesto)]
		a.Selecciones["profesional_"+strings.ToLower(e.Aceptacion.ProfesionalPuesto)] = a.MarcaSeleccion
	}
	for _, o := range []string{"anexo3", "anexo4", "etf"} {
		a.Selecciones["origen_"+o] = ""
	}
	a.Selecciones["origen_"+origen] = a.MarcaSeleccion
	// El extractor conserva columnas por orden, incluidos los huecos vacíos.
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
	a.ServicioDerivante = valor("servicio.nombre", 0)
	if e.ServicioDerivante != "" {
		a.ServicioDerivante = nombres["servicio_derivante."+e.ServicioDerivante]
		if a.ServicioDerivante == "" {
			return d.Datos{}, d.ErrNoDisponible
		}
	}
	a.DestinatarioSSCC = a.ServicioDerivante
	a.Direccion = valor("familia.direccion", 0)
	var contactos []string
	for i, n := range campos["familia.contacto.nombre"] {
		nombre := strings.TrimSpace(n + " " + valor("familia.contacto.apellidos", i))
		if nombre != "" {
			contactos = append(contactos, nombre)
		}
	}
	a.Familia = strings.Join(contactos, "; ")
	for i, n := range campos["sociodemografia.adulto.nombre"] {
		if strings.TrimSpace(n) == "" {
			continue
		}
		parentesco := valor("sociodemografia.adulto.parentesco", i)
		persona := d.Persona{Nombre: n, Nacimiento: valor("sociodemografia.adulto.nacimiento", i)}
		// No se inventa una fecha de nacimiento a partir de una edad.
		switch strings.ToLower(strings.TrimSpace(parentesco)) {
		case strings.ToLower(textos["documento.parentesco.madre"]):
			a.Madre = persona
		case strings.ToLower(textos["documento.parentesco.padre"]):
			a.Padre = persona
		default:
			a.Adultos = append(a.Adultos, d.Adulto{Persona: persona, Parentesco: parentesco})
		}
	}
	if s.Tipo == d.ActaAceptacion && e.CopiaTraslado != nil {
		if err := rellenarCopiaTraslado(a, *e.CopiaTraslado, nombres); err != nil {
			return d.Datos{}, err
		}
	}
	sexos := map[int]string{}
	if aceptada {
		for _, m := range e.Aceptacion.Menores {
			sexos[m.Orden] = m.Sexo
		}
	}
	for i, n := range campos["menores.persona.nombre"] {
		sexo := sexos[i+1]
		if aceptada && sexo != "H" && sexo != "M" {
			return d.Datos{}, d.ErrPeticion
		}
		a.Menores = append(a.Menores, d.Menor{Persona: d.Persona{Nombre: n, Nacimiento: valor("menores.persona.nacimiento", i)}, Sexo: textos["documento.sexo."+strings.ToLower(sexo)], CodigoSexo: sexo})
	}
	if s.Tipo == d.Anexo5 {
		rellenarDatosSPM(a, campos)

	}
	if d.EsConsentimiento(s.Tipo) {
		// Sólo se proyectan nombres; no se recoge ningún identificador del representante.
		a.Selecciones["fecha_documento"] = instante.Format(textos["documento.consentimiento.formato_fecha"])
		nombresMenores := make([]string, 0, len(a.Menores))
		for _, menor := range a.Menores {
			nombresMenores = append(nombresMenores, menor.Nombre)
		}
		a.Selecciones["menores_lista"] = strings.Join(nombresMenores, "; ")
		if textos["documento.consentimiento.formato_fecha"] == "" {
			return d.Datos{}, d.ErrNoDisponible
		}
	}
	equipo := ""
	if e.EquipoID != nil {
		equipo = *e.EquipoID
	}
	ahora := instante

	if s.Tipo == d.ActaCierre && !s.FechaReproduccion.IsZero() {
		ahora = s.FechaReproduccion
	}
	if s.Tipo == d.Anexo5 {
		ahora, _ = time.Parse("2006-01-02", fechaTexto)
	}
	if s.Tipo == d.ActaCierre {
		acta := e.ActaCierre
		motivo := nombres["motivo_acta_cierre."+acta.MotivoActa]
		if motivo == "" {
			return d.Datos{}, d.ErrNoDisponible
		}
		if acta.MotivoActaTexto != "" {
			motivo = acta.MotivoActaTexto
		}
		a.Selecciones["motivo_cierre"] = motivo
		a.Selecciones["fecha_cierre"] = acta.Fecha
	}
	versiones := map[string]string{"expediente": fmt.Sprint(e.Version), "derivacion": e.Datos.EsquemaVersion, "lectura_anexo": e.Lectura.ID, "catalogo_asignacion": v.ID, "textos_catalogo": fmt.Sprint(ref.Version) + ":" + ref.Huella}
	if finalizacion {
		final := e.Finalizacion
		motivo := nombres["motivo_finalizacion."+final.Motivo]
		if motivo == "" {
			return d.Datos{}, d.ErrNoDisponible
		}
		a.Finalizacion = map[string]string{"fecha_documento": ahora.Format("02/01/2006"), "motivo": motivo, "motivo_texto": final.MotivoTexto, "derivacion_tras_cierre": final.DerivacionTrasCierre, "coordinacion_recursos": final.CoordinacionRecursos, "objetivos_abordados": final.ObjetivosAbordados, "objetivos_pendientes": final.ObjetivosPendientes}
		for _, codigo := range []string{"OBJ_TOTAL", "OBJ_PARCIAL", "SIN_ADHERENCIA", "ABANDONO", "TRASLADO", "MAYORIA_EDAD", "INGRESO_CENTRO", "DECLARACION_RIESGO", "MEDIDA_PROTECTORA", "OTROS", "SIN_CONSENTIMIENTO"} {
			a.Finalizacion["motivo_"+strings.ToLower(codigo)] = ""
			if codigo == final.Motivo {
				a.Finalizacion["motivo_"+strings.ToLower(codigo)] = a.MarcaSeleccion
			}
		}
		if final.Motivo == "DECLARACION_RIESGO" || final.Motivo == "MEDIDA_PROTECTORA" || final.Motivo == "SIN_CONSENTIMIENTO" {
			a.Finalizacion["motivo_otros"] = a.MarcaSeleccion
		}
		versiones["catalogo_cierre"] = final.CatalogoVersionID
	}
	return d.Datos{ExpedienteID: e.ID, EquipoID: equipo, FechaDocumento: time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC), Aceptacion: a, Versiones: versiones}, nil
}

// El traslado usa los datos vigentes copiados, conservados para repetir la emisión.
func rellenarCopiaTraslado(a *d.Aceptacion, copia familia.CopiaTraslado, nombres map[string]string) error {
	a.Direccion = copia.Familia.Direccion
	a.Madre, a.Padre, a.Adultos = d.Persona{}, d.Persona{}, nil
	var contactos []string
	for _, adulto := range copia.Adultos {
		nombre := strings.TrimSpace(adulto.Nombre + " " + adulto.Apellidos)
		if nombre == "" {
			continue
		}
		contactos = append(contactos, nombre)
		persona := d.Persona{Nombre: nombre}
		if adulto.Nacimiento != nil {
			persona.Nacimiento = *adulto.Nacimiento
		}
		switch adulto.Papel {
		case "MADRE":
			a.Madre = persona
		case "PADRE":
			a.Padre = persona
		default:
			parentesco := nombres["papel_parental."+adulto.Papel]
			if parentesco == "" {
				return d.ErrNoDisponible
			}
			a.Adultos = append(a.Adultos, d.Adulto{Persona: persona, Parentesco: parentesco})
		}
	}
	a.Familia = strings.Join(contactos, "; ")
	return nil
}
