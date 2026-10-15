package estadistica

import "time"

type descripcionApartado struct {
	a                Apartado
	poblacion, fecha string
}

var apartadosAdicionales = []descripcionApartado{
	{Antecedentes, "AT", "corte"}, {ServicioDerivante, "AT", "corte"},
	{NoAceptados, "NC", "fecha_no_aceptacion"}, {Pendientes, "PE", "corte"},
	{DerivadosSSCC, "CI", "cierre_episodio"}, {PropuestasDSR, "AT", "corte"},
	{PIFDerivadoDSR, "AT", "corte"}, {PropuestasMedida, "AT", "corte"},
	{RespuestaSPM, "AT", "corte"}, {SeguimientoReunificacion, "AT", "cese_medida"},
	{ComposicionFamiliar, "AT", "corte"}, {FamiliasNumerosas, "AT", "corte"},
	{EdadParental, "AT", "corte"}, {NivelEducativoParental, "AT", "corte"},
	{SituacionLaboralParental, "AT", "corte"}, {SituacionEconomica, "AT", "corte"},
	{IngresosSMI, "AT", "corte"}, {FactoresFamilia, "AT", "corte"},
}

func inicializarApartadosAdicionales(m *Memoria, version string, corte, fin time.Time) {
	for _, d := range apartadosAdicionales {
		m.Apartados[d.a] = nuevoResultado(d.a, d.poblacion, d.fecha, version, corte, fin)
	}
	for _, a := range []Apartado{Antecedentes, ServicioDerivante, ComposicionFamiliar, FamiliasNumerosas, EdadParental, NivelEducativoParental, SituacionLaboralParental, SituacionEconomica, IngresosSMI, FactoresFamilia} {
		m.Apartados[a].Contrato.Denominador = "atendidos_del_apartado_1"
	}
	m.Apartados[NoAceptados].Contrato.Denominador = "no_aceptados_con_fecha_en_el_anio"
	m.Apartados[Pendientes].Contrato.Denominador = "lista_espera_o_aceptados_sin_consentimiento_mas_de_30_dias"
	m.Apartados[DerivadosSSCC].Contrato.Denominador = "cerrados_en_el_anio"
	m.Apartados[PropuestasDSR].Contrato.Denominador = "atendidos_riesgo_del_apartado_1"
	m.Apartados[PIFDerivadoDSR].Contrato.Denominador = "atendidos_riesgo_del_apartado_1"
	m.Apartados[PropuestasMedida].Contrato.Denominador = "atendidos_riesgo_del_apartado_1"
	m.Apartados[RespuestaSPM].Contrato.Denominador = "propuestas_del_apartado_12_1"
	m.Apartados[SeguimientoReunificacion].Contrato.Denominador = "reunificacion_del_apartado_1"
}

func calcularApartadosAdicionales(m *Memoria, exp Expediente, atendido bool, s Subprograma, menores Dato[[]Menor], inicio, corte, fin, conocido time.Time, params Parametros) error {
	if atendido {
		if err := sumarResponsables(m, exp, corte, conocido, params.ResponsablesDesproteccion); err != nil {
			return err
		}
		if err := calcularDatosAtendidos(m, exp, s, menores, corte, fin, conocido, params); err != nil {
			return err
		}
	}
	if err := calcularNoAceptados(m, exp, inicio, corte); err != nil {
		return err
	}
	if err := calcularPendientes(m, exp, corte, conocido, fin); err != nil {
		return err
	}
	if err := calcularDerivacionCierre(m, exp, inicio, corte, conocido); err != nil {
		return err
	}
	if atendido && s == Reunificacion {
		for _, v := range exp.SeguimientosTrasCese {
			if v.cruza(inicio, corte) {
				r := m.Apartados[SeguimientoReunificacion]
				r.Total.Familias++
				r.sumar(Grupo{Subprograma: Reunificacion}, Familias, false)
				break
			}
		}
	}
	return nil
}

func calcularDatosAtendidos(m *Memoria, exp Expediente, s Subprograma, menores Dato[[]Menor], corte, fin, conocido time.Time, params Parametros) error {
	if err := sumarHistoriaBool(m, exp.Antecedentes, Antecedentes, corte, conocido, Grupo{Subprograma: s}, true); err != nil {
		return err
	}
	if err := sumarHistoriaTexto(m, exp.ServicioDerivante, ServicioDerivante, corte, conocido, Grupo{Subprograma: s}, false); err != nil {
		return err
	}
	if err := sumarHistoriaBool(m, exp.FamiliaNumerosa, FamiliasNumerosas, corte, conocido, Grupo{Subprograma: s}, false); err != nil {
		return err
	}
	if err := sumarHistoriaTexto(m, exp.Composicion, ComposicionFamiliar, corte, conocido, Grupo{Subprograma: s}, false); err != nil {
		return err
	}
	if err := sumarHistoriaTexto(m, exp.SituacionEconomica, SituacionEconomica, corte, conocido, Grupo{Subprograma: s}, false); err != nil {
		return err
	}
	if err := sumarIngresos(m, exp, s, corte, conocido, params); err != nil {
		return err
	}
	if err := sumarFactores(m, exp, s, corte, conocido, params.FactoresFamilia); err != nil {
		return err
	}
	if err := sumarAdultos(m, exp, corte, fin, conocido); err != nil {
		return err
	}
	if s == Reunificacion {
		return nil
	}
	p, _, err := exp.Propuestas.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	if p.Conocido {
		for _, v := range []struct {
			a Apartado
			d Dato[bool]
		}{{PropuestasDSR, p.Valor.DeclaracionRiesgo}, {PIFDerivadoDSR, p.Valor.PIFDerivadoDSR}, {PropuestasMedida, p.Valor.MedidaProtectora}} {
			if v.d.Conocido && v.d.Valor {
				sumarCelda(m.Apartados[v.a], Grupo{Subprograma: s, Estado: "si"}, Familias, false)
				m.Apartados[v.a].Total.Familias++
			} else if !v.d.Conocido {
				sumarCelda(m.Apartados[v.a], Grupo{Subprograma: s}, Familias, true)
				m.Apartados[v.a].Total.Familias++
			}
		}
		if p.Valor.MedidaProtectora.Conocido && p.Valor.MedidaProtectora.Valor {
			d := p.Valor.RespuestaSPM
			sumarCelda(m.Apartados[RespuestaSPM], Grupo{Estado: d.Valor}, Familias, !d.Conocido)
			m.Apartados[RespuestaSPM].Total.Familias++
		}
	} else {
		for _, a := range []Apartado{PropuestasDSR, PIFDerivadoDSR, PropuestasMedida} {
			sumarCelda(m.Apartados[a], Grupo{Subprograma: s}, Familias, true)
			m.Apartados[a].Total.Familias++
		}
	}
	if menores.Conocido {
		for _, men := range menores.Valor {
			for _, a := range []Apartado{PropuestasDSR, PIFDerivadoDSR, PropuestasMedida} {
				r := m.Apartados[a]
				p, _, _ := exp.Propuestas.ValorEn(corte, conocido)
				if p.Conocido {
					var d Dato[bool]
					switch a {
					case PropuestasDSR:
						d = p.Valor.DeclaracionRiesgo
					case PIFDerivadoDSR:
						d = p.Valor.PIFDerivadoDSR
					default:
						d = p.Valor.MedidaProtectora
					}
					if d.Conocido && d.Valor {
						r.Total.Menores++
						sumarCelda(r, Grupo{Subprograma: men.Subprograma, Estado: "si"}, Menores, false)
					} else if !d.Conocido {
						r.Total.Menores++
						sumarCelda(r, Grupo{Subprograma: men.Subprograma}, Menores, true)
					}
				} else {
					r.Total.Menores++
					sumarCelda(r, Grupo{Subprograma: men.Subprograma}, Menores, true)
				}
			}
		}
	}
	return nil
}

func calcularNoAceptados(m *Memoria, exp Expediente, inicio, corte time.Time) error {
	for _, n := range exp.NoAceptaciones {
		f := fecha(n.Fecha)
		if f.IsZero() || f.Before(inicio) || f.After(corte) {
			continue
		}
		g := Grupo{Subprograma: n.Subprograma, Estado: n.ServicioDerivante, Causa: n.Causa}
		m.Apartados[NoAceptados].Total.Familias++
		sumarCelda(m.Apartados[NoAceptados], g, Familias, n.Subprograma == "" || n.ServicioDerivante == "" || n.Causa == "")
	}
	return nil
}

func calcularPendientes(m *Memoria, exp Expediente, corte, conocido, fin time.Time) error {
	// Un expediente ya aceptado al corte no puede seguir en lista de espera.
	noAceptadoAlCorte := true
	for _, ep := range exp.Episodios {
		if !ep.Aceptacion.IsZero() && !fecha(ep.Aceptacion).After(corte) {
			noAceptadoAlCorte = false
			break
		}
	}
	if noAceptadoAlCorte {
		estado, _, err := exp.Estados.ValorEn(corte, conocido)
		if err != nil {
			return err
		}
		if estado.Conocido && estado.Valor == "lista_espera" {
			return sumarPendiente(m, exp, corte, conocido, fin, false)
		}
	}
	for _, ep := range exp.Episodios {
		if ep.Aceptacion.IsZero() || fecha(ep.Aceptacion).After(corte) || fecha(corte).Sub(fecha(ep.Aceptacion)).Hours() <= 30*24 || (!ep.Cierre.IsZero() && !fecha(ep.Cierre).After(corte)) {
			continue
		}
		cons, _, err := exp.Consentimiento.ValorEn(corte, conocido)
		if err != nil {
			return err
		}
		if cons.Conocido && cons.Valor {
			continue
		}
		return sumarPendiente(m, exp, corte, conocido, fin, !cons.Conocido)
	}
	return nil
}

func sumarPendiente(m *Memoria, exp Expediente, corte, conocido, fin time.Time, sinConsentimiento bool) error {
	s, menores, err := datosEn(exp, corte, conocido)
	if err != nil {
		return err
	}
	r := m.Apartados[Pendientes]
	r.Total.Familias++
	sumarCelda(r, Grupo{Subprograma: s}, Familias, s == "" || sinConsentimiento)
	if !menores.Conocido {
		r.Total.FamiliasSinRelacionMenores++
	} else {
		computables := menoresComputables(menores, fin)
		r.Total.Menores += len(computables.Valor)
		for _, men := range computables.Valor {
			sumarCelda(r, Grupo{Subprograma: men.Subprograma}, Menores, men.Subprograma == "")
		}
	}
	return nil
}

func calcularDerivacionCierre(m *Memoria, exp Expediente, inicio, corte, conocido time.Time) error {
	for _, ep := range exp.Episodios {
		if ep.Cierre.IsZero() || fecha(ep.Cierre).Before(inicio) || fecha(ep.Cierre).After(corte) {
			continue
		}
		d, _, err := exp.DerivadoSSCCTrasCierre.ValorEn(fecha(ep.Cierre), conocido)
		if err != nil {
			return err
		}
		if !d.Conocido || d.Valor {
			r := m.Apartados[DerivadosSSCC]
			r.Total.Familias++
			sumarCelda(r, Grupo{Estado: "si"}, Familias, !d.Conocido)
		}
		break
	}
	return nil
}

func sumarHistoriaBool(m *Memoria, h Historia[bool], a Apartado, corte, conocido time.Time, g Grupo, soloVerdad bool) error {
	d, _, err := h.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	g.Estado = "no"
	sin := !d.Conocido
	if d.Conocido && d.Valor {
		g.Estado = "si"
	}
	if soloVerdad && d.Conocido && !d.Valor {
		return nil
	}
	r := m.Apartados[a]
	r.Total.Familias++
	sumarCelda(r, g, Familias, sin)
	return nil
}
func sumarHistoriaTexto(m *Memoria, h Historia[string], a Apartado, corte, conocido time.Time, g Grupo, ignorarVacio bool) error {
	d, _, err := h.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	g.Estado = d.Valor
	r := m.Apartados[a]
	r.Total.Familias++
	sumarCelda(r, g, Familias, !d.Conocido || (!ignorarVacio && d.Valor == ""))
	return nil
}
func sumarCelda(r *Resultado, g Grupo, u Unidad, sin bool) { r.sumar(g, u, sin) }

func sumarAdultos(m *Memoria, exp Expediente, corte, fin, conocido time.Time) error {
	d, _, err := exp.Adultos.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	if !d.Conocido {
		sumarCelda(m.Apartados[EdadParental], Grupo{}, Adultos, true)
		return nil
	}
	for _, adulto := range d.Valor {
		if !adulto.Nacimiento.IsZero() && fecha(adulto.Nacimiento).After(fin) {
			return ErrEntrada
		}
		g := Grupo{Papel: adulto.Papel, Sexo: adulto.Sexo}
		for _, a := range []Apartado{EdadParental, NivelEducativoParental, SituacionLaboralParental} {
			m.Apartados[a].Total.Adultos++
		}
		if adulto.Nacimiento.IsZero() {
			g.Tramo = ""
		} else {
			edad := fin.Year() - adulto.Nacimiento.Year()
			switch {
			case edad < 20:
				g.Tramo = "edad_menos_20"
			case edad < 30:
				g.Tramo = "edad_20_29"
			case edad < 40:
				g.Tramo = "edad_30_39"
			case edad < 50:
				g.Tramo = "edad_40_49"
			default:
				g.Tramo = "edad_50_mas"
			}
		}
		sumarCelda(m.Apartados[EdadParental], g, Adultos, adulto.Papel == "" || adulto.Sexo == "" || g.Tramo == "")
		for _, v := range []struct {
			a     Apartado
			texto string
		}{{NivelEducativoParental, adulto.NivelEducativo}, {SituacionLaboralParental, adulto.SituacionLaboral}} {
			g.Tramo = ""
			g.Estado = v.texto
			sumarCelda(m.Apartados[v.a], g, Adultos, v.texto == "" || adulto.Papel == "" || adulto.Sexo == "")
		}
	}
	return nil
}

func sumarIngresos(m *Memoria, exp Expediente, s Subprograma, corte, conocido time.Time, params Parametros) error {
	d, _, err := exp.Ingresos.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	g := Grupo{Subprograma: s}
	valor := ""
	if d.Conocido {
		valor = d.Valor.Tramo.Valor
		if !d.Valor.Tramo.Conocido && d.Valor.NetosAnualesCentimos.Conocido && params.SMIAnualCentimos > 0 {
			ingreso := d.Valor.NetosAnualesCentimos.Valor
			entero := ingreso / params.SMIAnualCentimos
			if entero > int64(^uint64(0)>>1)/100 {
				return ErrEntrada
			}
			ratio := entero*100 + (ingreso%params.SMIAnualCentimos)*100/params.SMIAnualCentimos
			for _, t := range params.TramosSMI {
				if ratio >= t.DesdeCentimos && (t.HastaCentimos == nil || ratio < *t.HastaCentimos) {
					valor = t.Codigo
					break
				}
			}
		}
	}
	g.Tramo = valor
	r := m.Apartados[IngresosSMI]
	r.Total.Familias++
	sumarCelda(r, g, Familias, valor == "")
	return nil
}

func sumarFactores(m *Memoria, exp Expediente, s Subprograma, corte, conocido time.Time, codigos []string) error {
	d, err := factoresEn(exp, "factor_familia", corte, conocido)
	if err != nil {
		return err
	}
	observados := map[string]Dato[bool]{}
	if d.Conocido {
		for _, f := range d.Valor {
			observados[f.Codigo] = f.Presente
		}
	}
	r := m.Apartados[FactoresFamilia]
	r.Total.Familias++
	for _, codigo := range codigos {
		v, ok := observados[codigo]
		g := Grupo{Subprograma: s, Estado: codigo}
		if !d.Conocido || (ok && !v.Conocido) {
			sumarCelda(r, g, Familias, true)
		} else if ok && v.Valor {
			sumarCelda(r, g, Familias, false)
		} else {
			asegurarCelda(r, g, Familias)
		}
	}
	return nil
}

func anexarPorcentajes(m *Memoria) error {
	for _, a := range []Apartado{PropuestasDSR, PIFDerivadoDSR, PropuestasMedida} {
		asegurarCelda(m.Apartados[a], Grupo{Subprograma: Riesgo, Estado: "si"}, Familias)
		asegurarCelda(m.Apartados[a], Grupo{Subprograma: Riesgo, Estado: "si"}, Menores)
	}
	for _, respuesta := range []string{RespuestaInicioProcedimiento, RespuestaEnEspera, RespuestaNoNecesaria} {
		asegurarCelda(m.Apartados[RespuestaSPM], Grupo{Estado: respuesta}, Familias)
	}
	asegurarCelda(m.Apartados[SeguimientoReunificacion], Grupo{Subprograma: Reunificacion}, Familias)
	base := func(a Apartado, s Subprograma, u Unidad) Dato[int] {
		if u == Menores && m.Apartados[a].Total.FamiliasSinRelacionMenores > 0 {
			return Dato[int]{}
		}
		n := 0
		for k, c := range m.Apartados[a].Celdas {
			if k.Unidad != u {
				continue
			}
			if s != "" && k.Grupo.Subprograma != s && k.Grupo.Subprograma != "" {
				continue
			}
			if c.SinDato > 0 && (s == "" || k.Grupo.Subprograma == "") {
				return Dato[int]{}
			}
			if s == "" || k.Grupo.Subprograma == s {
				n += c.Numerador + c.SinDato
			}
		}
		return Dato[int]{Valor: n, Conocido: true}
	}
	for _, a := range []Apartado{PropuestasDSR, PIFDerivadoDSR, PropuestasMedida, RespuestaSPM, SeguimientoReunificacion} {
		r := m.Apartados[a]
		for k, c := range r.Celdas {
			den := base(CasosAtendidos, Riesgo, k.Unidad)
			if a == RespuestaSPM {
				den = base(PropuestasMedida, "", Familias)
			}
			if a == SeguimientoReunificacion {
				den = base(CasosAtendidos, Reunificacion, Familias)
			}
			c.Denominador = den
			if c.SinDato == 0 {
				p, err := CalcularPorcentaje(c.Numerador, den)
				if err != nil {
					return err
				}
				c.Porcentaje = &p
				if p.Aviso != "" {
					r.Avisos[p.Aviso]++
				}
			}
			r.Celdas[k] = c
		}
	}
	return nil
}

func asegurarCelda(r *Resultado, g Grupo, unidad Unidad) {
	k := ClaveCelda{Grupo: g, Unidad: unidad}
	if _, existe := r.Celdas[k]; !existe {
		r.Celdas[k] = Celda{Poblacion: r.Contrato.Poblacion, VersionRegla: r.Contrato.VersionRegla}
	}
}
