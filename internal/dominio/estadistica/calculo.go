package estadistica

import "time"

// Calcular implementa apartados con datos estructurados y deja cada base de
// porcentaje explícita o pendiente cuando la población no permite conocerla.
func Calcular(e Entrada) (Memoria, error) {
	if e.Anio < 1 || e.Anio > 9999 || e.FechaCalculo.IsZero() || e.ConocidoEn.IsZero() {
		return Memoria{}, ErrEntrada
	}
	inicio := time.Date(e.Anio, 1, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(e.Anio, 12, 31, 0, 0, 0, 0, time.UTC)
	corte := fecha(e.FechaCalculo)
	if corte.Before(inicio) {
		return Memoria{}, ErrEntrada
	}
	if corte.After(fin) {
		corte = fin
	}
	reglas, version, err := e.Reglas.ValorEn(fin, e.ConocidoEn)
	if err != nil {
		return Memoria{}, err
	}
	if !reglas.Conocido || !tramosValidos(reglas.Valor.TramosRiesgo) || !tramosValidos(reglas.Valor.TramosReunificacion) || !parametrosValidos(reglas.Valor) {
		return Memoria{}, ErrReglas
	}
	m := Memoria{Anio: e.Anio, Corte: corte, ConocidoEn: e.ConocidoEn, Apartados: make(map[Apartado]*Resultado)}
	for _, c := range []struct {
		a                     Apartado
		poblacion, fechaDatos string
	}{
		{CasosAtendidos, "AT_ME", "corte"}, {MenoresPorSexo, "ME", "corte"},
		{CasosCerrados, "CI", "cierre_episodio"}, {TratamientoConcluido, "CO", "cierre_episodio"},
		{TratamientoInterrumpido, "NC", "cierre_episodio"}, {DuracionRiesgo, "CO", "cierre_episodio"},
		{DuracionReunificacion, "CO", "cierre_episodio"}, {MenoresPorEdad, "ME", "corte"},
	} {
		m.Apartados[c.a] = nuevoResultado(c.a, c.poblacion, c.fechaDatos, version, corte, fin)
	}
	for _, c := range []struct {
		a    Apartado
		p, f string
	}{
		{CoordinacionRecursos, "ME", "corte"}, {AtencionTemprana, "ME", "corte"},
		{ComisionesTecnicas, "ACT", "fecha"}, {InformesTecnicos, "ACT", "fecha"},
		{FactoresMenores, "ME", "corte"}, {TipologiasDesproteccion, "ME", "corte"},
		{ResponsablesDesproteccion, "AT", "corte"},
	} {
		m.Apartados[c.a] = nuevoResultado(c.a, c.p, c.f, version, corte, fin)
	}
	for _, a := range []Apartado{ComisionesTecnicas, InformesTecnicos} {
		m.Apartados[a].Contrato.Poblacion = "ACT"
	}
	if e.Actividad != nil {
		if e.Actividad.Anio != e.Anio || e.Actividad.Comisiones < 0 || e.Actividad.Informes < 0 || uint64(e.Actividad.Comisiones) > uint64(^uint(0)>>1) || uint64(e.Actividad.Informes) > uint64(^uint(0)>>1) {
			return Memoria{}, ErrEntrada
		}
		var sumaInformes int64
		for _, total := range e.Actividad.PorOrganismo {
			if total < 0 || sumaInformes > int64(^uint64(0)>>1)-total {
				return Memoria{}, ErrEntrada
			}
			sumaInformes += total
		}
		if sumaInformes != e.Actividad.Informes {
			return Memoria{}, ErrEntrada
		}
		m.Apartados[ComisionesTecnicas].Total.Comisiones = int(e.Actividad.Comisiones)
		m.Apartados[ComisionesTecnicas].Celdas[ClaveCelda{Grupo: Grupo{Estado: "comision"}, Unidad: Comisiones}] = Celda{Numerador: int(e.Actividad.Comisiones), Poblacion: "ACT", VersionRegla: version}
		m.Apartados[InformesTecnicos].Total.Informes = int(e.Actividad.Informes)
		for codigo, n := range e.Actividad.PorOrganismo {
			if uint64(n) > uint64(^uint(0)>>1) {
				return Memoria{}, ErrEntrada
			}
			celda := Celda{Numerador: int(n), Poblacion: "ACT", VersionRegla: version}
			if codigo == "" {
				celda.Numerador = 0
				celda.SinDato = int(n)
			}
			m.Apartados[InformesTecnicos].Celdas[ClaveCelda{Grupo: Grupo{Organismo: codigo}, Unidad: Informes}] = celda
		}
	} else {
		m.Apartados[ComisionesTecnicas].Avisos["estadistica.recuento_actividad_sin_dato"]++
		m.Apartados[InformesTecnicos].Avisos["estadistica.recuento_actividad_sin_dato"]++
	}
	inicializarApartadosAdicionales(&m, version, corte, fin)
	ids := make(map[string]bool)
	for _, exp := range e.Expedientes {
		if exp.ID == "" || (exp.EquipoID == "" && len(exp.Episodios) > 0) || ids[exp.ID] {
			return Memoria{}, ErrEntrada
		}
		ids[exp.ID] = true
		if err := validarExpediente(exp, e.ConocidoEn); err != nil {
			return Memoria{}, err
		}
		if len(exp.Episodios) == 0 {
			if err := calcularApartadosAdicionales(&m, exp, false, "", Dato[[]Menor]{}, inicio, corte, fin, e.ConocidoEn, reglas.Valor); err != nil {
				return Memoria{}, err
			}
			continue
		} // derivación aún no aceptada
		aceptacion := aceptacionInicial(exp)
		origen := origenAceptacion(aceptacion, inicio)
		incluido, seguimientoSinDato, err := atendido(exp, inicio, corte, e.ConocidoEn)
		if err != nil {
			return Memoria{}, err
		}
		if seguimientoSinDato {
			m.Apartados[CasosAtendidos].Avisos["estadistica.seguimiento_subprograma_sin_dato"]++
		}
		var s Subprograma
		var menores Dato[[]Menor]
		if incluido {
			s, menores, err = datosEn(exp, corte, e.ConocidoEn)
			if err != nil {
				return Memoria{}, err
			}
			computables := menoresComputables(menores, fin)
			r := m.Apartados[CasosAtendidos]
			g := Grupo{Subprograma: s, Origen: origen}
			sumarCaso(r, g, computables, s == "" || origen == "")
			for _, a := range []Apartado{MenoresPorSexo, MenoresPorEdad} {
				r := m.Apartados[a]
				r.Total.Menores += len(computables.Valor)
				if !computables.Conocido {
					r.Total.FamiliasSinRelacionMenores++
				}
				for _, menor := range computables.Valor {
					g := Grupo{Subprograma: menor.Subprograma, Sexo: menor.Sexo}
					if a == MenoresPorEdad {
						g.Tramo = tramoEdad(menor.Nacimiento, fin)
					}
					sinDato := g.Subprograma == "" || g.Sexo == "" || (a == MenoresPorEdad && g.Tramo == "")
					r.sumar(g, Menores, sinDato)
				}
			}
			if err := calcularApartadosResto(&m, exp, menores, corte, e.ConocidoEn, reglas.Valor); err != nil {
				return Memoria{}, err
			}
		}
		if err := calcularApartadosAdicionales(&m, exp, incluido, s, menores, inicio, corte, fin, e.ConocidoEn, reglas.Valor); err != nil {
			return Memoria{}, err
		}
		var cierres []Episodio
		for _, ep := range exp.Episodios {
			if !ep.Cierre.IsZero() && !fecha(ep.Cierre).Before(inicio) && !fecha(ep.Cierre).After(corte) {
				cierres = append(cierres, ep)
			}
		}
		if len(cierres) == 0 {
			continue
		}
		if len(cierres) > 1 {
			// No se elige ni el primero ni el último; CI incluye una familia y
			// CO/NC y duración quedan fuera del reparto hasta decisión del Programa.
			r := m.Apartados[CasosCerrados]
			r.Total.Familias++
			r.Total.FamiliasSinRelacionMenores++
			r.sumar(Grupo{Origen: origen}, Familias, true)
			for _, a := range []Apartado{CasosCerrados, TratamientoConcluido, TratamientoInterrumpido, DuracionRiesgo, DuracionReunificacion} {
				m.Apartados[a].Avisos["estadistica.varios_cierres_pendiente"]++
			}
			continue
		}
		ep := cierres[0]
		s, menores, err = datosEn(exp, fecha(ep.Cierre), e.ConocidoEn)
		if err != nil {
			return Memoria{}, err
		}
		computables := menoresComputables(menores, fin)
		sumarCaso(m.Apartados[CasosCerrados], Grupo{Subprograma: s, Origen: origen}, computables, s == "" || origen == "")
		if ep.Motivo == "" {
			for _, a := range []Apartado{CasosCerrados, TratamientoConcluido, TratamientoInterrumpido} {
				m.Apartados[a].Avisos["estadistica.motivo_sin_dato"]++
			}
			continue
		}
		if !concluido(ep.Motivo) {
			sumarCaso(m.Apartados[TratamientoInterrumpido], Grupo{Subprograma: s, Motivo: ep.Motivo}, computables, s == "")
			continue
		}
		sumarCaso(m.Apartados[TratamientoConcluido], Grupo{Subprograma: s, Motivo: ep.Motivo}, computables, s == "")
		if s == "" {
			for _, a := range []Apartado{DuracionRiesgo, DuracionReunificacion} {
				m.Apartados[a].Avisos["estadistica.subprograma_sin_dato"]++
			}
			continue
		}
		a, tramos := DuracionRiesgo, reglas.Valor.TramosRiesgo
		if s == Reunificacion {
			a, tramos = DuracionReunificacion, reglas.Valor.TramosReunificacion
		}
		r := m.Apartados[a]
		r.Total.Familias++
		g := Grupo{Subprograma: s}
		if ep.Aceptacion.IsZero() {
			r.sumar(g, Familias, true)
			r.Avisos["estadistica.aceptacion_sin_dato"]++
			continue
		}
		meses, err := MesesCompletos(ep.Aceptacion, ep.Cierre)
		if err != nil {
			return Memoria{}, err
		}
		g.Tramo = buscarTramo(tramos, meses)
		r.sumar(g, Familias, false)
	}
	if err := anexarPorcentajes(&m); err != nil {
		return Memoria{}, err
	}
	return m, nil
}

func sumarCaso(r *Resultado, g Grupo, menores Dato[[]Menor], sinDato bool) {
	r.Total.Familias++
	r.sumar(g, Familias, sinDato)
	r.Total.Menores += len(menores.Valor)
	if !menores.Conocido {
		r.Total.FamiliasSinRelacionMenores++
	}
	for _, menor := range menores.Valor {
		g.Subprograma = menor.Subprograma
		necesitaOrigen := r.Contrato.Apartado == CasosAtendidos || r.Contrato.Apartado == CasosCerrados
		r.sumar(g, Menores, g.Subprograma == "" || (g.Origen == "" && necesitaOrigen))
	}
}

func datosEn(exp Expediente, corte, conocido time.Time) (Subprograma, Dato[[]Menor], error) {
	s, _, err := exp.Subprograma.ValorEn(corte, conocido)
	if err != nil {
		return "", Dato[[]Menor]{}, err
	}
	m, _, err := exp.Menores.ValorEn(corte, conocido)
	if err != nil {
		return "", Dato[[]Menor]{}, err
	}
	if !s.Conocido {
		s.Valor = ""
	}
	if !m.Conocido {
		m.Valor = nil
	}
	for _, menor := range m.Valor {
		if !menor.Nacimiento.IsZero() && fecha(menor.Nacimiento).After(corte) {
			return "", Dato[[]Menor]{}, ErrEntrada
		}
	}
	return s.Valor, m, nil
}

func menoresComputables(d Dato[[]Menor], fin time.Time) Dato[[]Menor] {
	r := Dato[[]Menor]{Conocido: d.Conocido}
	for _, m := range d.Valor {
		// Una fecha ausente conserva al menor en ME y deja su edad sin dato.
		// La nota 43 permite 18 en el año; 19 o más no son menores computables.
		if m.Nacimiento.IsZero() || (!fecha(m.Nacimiento).After(fin) && fin.Year()-m.Nacimiento.Year() <= 18) {
			r.Valor = append(r.Valor, m)
		}
	}
	return r
}

func tramoEdad(nacimiento, fin time.Time) string {
	if nacimiento.IsZero() {
		return ""
	}
	edad := fin.Year() - nacimiento.Year() // fin es siempre 31/12
	switch {
	case edad <= 5:
		return "edad_0_5"
	case edad <= 11:
		return "edad_6_11"
	default:
		return "edad_12_17"
	}
}

func aceptacionInicial(exp Expediente) time.Time {
	for _, ep := range exp.Episodios {
		if ep.ReaperturaDe == "" {
			return fecha(ep.Aceptacion)
		}
	}
	return time.Time{}
}

func origenAceptacion(aceptacion, inicio time.Time) string {
	if aceptacion.IsZero() {
		return ""
	}
	if aceptacion.Before(inicio) {
		return "anterior"
	}
	return "incorporado"
}

func atendido(exp Expediente, inicio, corte, conocido time.Time) (bool, bool, error) {
	abierto := false
	for _, ep := range exp.Episodios {
		if !fecha(ep.Apertura).After(corte) && (ep.Cierre.IsZero() || !fecha(ep.Cierre).Before(inicio)) {
			abierto = true
		}
	}
	seguimientoSinDato := false
	for _, v := range exp.SeguimientosTrasCese {
		if v.cruza(inicio, corte) {
			s, _, err := exp.Subprograma.ValorEn(v.Desde, conocido)
			if err != nil {
				return false, false, err
			}
			if s.Conocido && s.Valor == Reunificacion {
				abierto = true
			} else {
				seguimientoSinDato = true
			}
		}
	}
	return abierto, seguimientoSinDato, nil
}

func concluido(m Motivo) bool { return m == ObjetivosCumplidos || m == ObjetivosParciales }

func buscarTramo(tramos []Tramo, meses int) string {
	for _, t := range tramos {
		if meses >= t.DesdeMeses && (t.HastaMeses == nil || meses < *t.HastaMeses) {
			return t.Codigo
		}
	}
	return "" // tramosValidos exige cobertura completa
}

func tramosValidos(tramos []Tramo) bool {
	if len(tramos) == 0 {
		return false
	}
	siguiente := 0
	codigos := make(map[string]bool)
	for i, t := range tramos {
		if t.Codigo == "" || codigos[t.Codigo] || t.DesdeMeses != siguiente {
			return false
		}
		codigos[t.Codigo] = true
		if t.HastaMeses == nil {
			return i == len(tramos)-1
		}
		if *t.HastaMeses <= t.DesdeMeses {
			return false
		}
		siguiente = *t.HastaMeses
	}
	return false
}
