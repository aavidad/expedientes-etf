package estadistica

import "time"

func validarExpediente(exp Expediente, conocido time.Time) error {
	porID := make(map[string]Episodio)
	raices := 0
	raiz := ""
	for _, ep := range exp.Episodios {
		if ep.ID == "" || ep.Apertura.IsZero() {
			return ErrEntrada
		}
		if _, existe := porID[ep.ID]; existe {
			return ErrEntrada
		}
		if !ep.Aceptacion.IsZero() && fecha(ep.Aceptacion).After(fecha(ep.Apertura)) {
			return ErrEntrada
		}
		if !ep.Cierre.IsZero() && fecha(ep.Cierre).Before(fecha(ep.Apertura)) {
			return ErrEntrada
		}
		if ep.Cierre.IsZero() && ep.Motivo != "" {
			return ErrEntrada
		}
		if !motivoValido(ep.Motivo) {
			return ErrEntrada
		}
		if ep.ReaperturaDe == "" {
			raices++
			raiz = ep.ID
		}
		porID[ep.ID] = ep
	}
	if len(exp.Episodios) > 0 && raices != 1 {
		return ErrEntrada
	}
	reabiertos := make(map[string]bool)
	sucesor := make(map[string]string)
	for _, ep := range exp.Episodios {
		if ep.ReaperturaDe == "" {
			continue
		}
		previo, existe := porID[ep.ReaperturaDe]
		if !existe || previo.Cierre.IsZero() || fecha(ep.Apertura).Before(fecha(previo.Cierre)) || reabiertos[ep.ReaperturaDe] {
			return ErrEntrada
		}
		reabiertos[ep.ReaperturaDe] = true
		sucesor[ep.ReaperturaDe] = ep.ID
	}
	// Admite cierre y reapertura en el mismo día. Comprueba una sola cadena
	// conectada para impedir ciclos u otros episodios ajenos a la aceptación.
	visitados := make(map[string]bool)
	for id := raiz; id != ""; id = sucesor[id] {
		if visitados[id] {
			return ErrEntrada
		}
		visitados[id] = true
	}
	if len(visitados) != len(exp.Episodios) {
		return ErrEntrada
	}
	for _, v := range exp.SeguimientosTrasCese {
		if !v.valida() {
			return ErrEntrada
		}
	}
	if _, err := exp.Composicion.activas(conocido); err != nil {
		return err
	}
	if _, err := exp.Ingresos.activas(conocido); err != nil {
		return err
	}
	if _, err := exp.Factores.activas(conocido); err != nil {
		return err
	}
	for _, h := range []interface{ validar(time.Time) error }{
		validarHistoria(exp.Antecedentes, conocido), validarHistoria(exp.ServicioDerivante, conocido),
		validarHistoria(exp.Consentimiento, conocido), validarHistoria(exp.DerivadoSSCCTrasCierre, conocido),
		validarHistoria(exp.FamiliaNumerosa, conocido), validarHistoria(exp.Adultos, conocido),
		validarHistoria(exp.SituacionEconomica, conocido), validarHistoria(exp.Propuestas, conocido),
		validarHistoria(exp.RecursosCoordinacion, conocido), validarHistoria(exp.Responsables, conocido),
		validarHistoria(exp.ListasConocidas, conocido),
	} {
		if err := h.validar(conocido); err != nil {
			return err
		}
	}
	for _, h := range []Historia[[]string]{exp.RecursosCoordinacion, exp.Responsables, exp.ListasConocidas} {
		for _, rev := range h {
			if rev.Dato.Conocido && !listaUnica(rev.Dato.Valor) {
				return ErrEntrada
			}
		}
	}
	noAceptaciones := make(map[string]bool)
	for _, n := range exp.NoAceptaciones {
		if n.Fecha.IsZero() || (n.Subprograma != "" && !subprogramaValido(n.Subprograma)) {
			return ErrEntrada
		}
		clave := fecha(n.Fecha).Format("2006-01-02") + "\x00" + n.ServicioDerivante + "\x00" + n.Causa + "\x00" + string(n.Subprograma)
		if noAceptaciones[clave] {
			return ErrEntrada
		}
		noAceptaciones[clave] = true
	}
	adultos, err := exp.Adultos.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range adultos {
		if r.Dato.Conocido {
			for _, a := range r.Dato.Valor {
				if a.Sexo != "" && a.Sexo != Nino && a.Sexo != Nina {
					return ErrEntrada
				}
			}
		}
	}
	factores, err := exp.Factores.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range factores {
		if !r.Dato.Conocido {
			continue
		}
		vistos := map[string]bool{}
		for _, f := range r.Dato.Valor {
			clave := f.Lista + ":" + f.Codigo
			if f.Codigo == "" || vistos[clave] {
				return ErrEntrada
			}
			vistos[clave] = true
			excluidos := map[string]bool{}
			for _, id := range f.MenoresExcluidos {
				if id == "" || excluidos[id] {
					return ErrEntrada
				}
				excluidos[id] = true
			}
		}
	}
	propuestas, err := exp.Propuestas.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range propuestas {
		if !r.Dato.Conocido || !r.Dato.Valor.RespuestaSPM.Conocido {
			continue
		}
		v := r.Dato.Valor.RespuestaSPM.Valor
		if v != "" && v != RespuestaInicioProcedimiento && v != RespuestaEnEspera && v != RespuestaNoNecesaria {
			return ErrEntrada
		}
	}
	ingresos, err := exp.Ingresos.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range ingresos {
		if r.Dato.Conocido && r.Dato.Valor.NetosAnualesCentimos.Conocido && r.Dato.Valor.NetosAnualesCentimos.Valor < 0 {
			return ErrEntrada
		}
	}
	subprogramas, err := exp.Subprograma.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range subprogramas {
		if r.Dato.Conocido && !subprogramaValido(r.Dato.Valor) {
			return ErrEntrada
		}
	}
	for _, v := range exp.SeguimientosTrasCese {
		s, _, err := exp.Subprograma.ValorEn(v.Desde, conocido)
		if err != nil {
			return err
		}
		if s.Conocido && s.Valor != Reunificacion {
			return ErrEntrada
		}
	}
	menores, err := exp.Menores.activas(conocido)
	if err != nil {
		return err
	}
	for _, r := range menores {
		if !r.Dato.Conocido {
			continue
		}
		ids := make(map[string]bool)
		for _, m := range r.Dato.Valor {
			if m.ID == "" || ids[m.ID] || (m.Sexo != "" && m.Sexo != Nino && m.Sexo != Nina) || (m.Subprograma != "" && !subprogramaValido(m.Subprograma)) {
				return ErrEntrada
			}
			ids[m.ID] = true
		}
	}
	return nil
}

func listaUnica(v []string) bool {
	vistos := map[string]bool{}
	for _, x := range v {
		if x == "" || vistos[x] {
			return false
		}
		vistos[x] = true
	}
	return true
}

type validadorHistoria interface{ validar(time.Time) error }
type historiaValidable[T any] Historia[T]

func (h historiaValidable[T]) validar(t time.Time) error {
	_, err := Historia[T](h).activas(t)
	return err
}
func validarHistoria[T any](h Historia[T], t time.Time) validadorHistoria {
	return historiaValidable[T](h)
}

func parametrosValidos(p Parametros) bool {
	if len(p.TramosSMI) > 0 && (p.SMIAnualCentimos <= 0 || p.SMIAnualCentimos > int64(^uint64(0)>>1)/100) {
		return false
	}
	siguiente := int64(0)
	codigos := map[string]bool{}
	for i, t := range p.TramosSMI {
		if t.Codigo == "" || codigos[t.Codigo] || t.DesdeCentimos != siguiente {
			return false
		}
		codigos[t.Codigo] = true
		if t.HastaCentimos == nil {
			if i != len(p.TramosSMI)-1 {
				return false
			}
			break
		}
		if *t.HastaCentimos <= t.DesdeCentimos {
			return false
		}
		siguiente = *t.HastaCentimos
	}
	if len(p.TramosSMI) > 0 && p.TramosSMI[len(p.TramosSMI)-1].HastaCentimos != nil {
		return false
	}
	factores := map[string]bool{}
	for _, codigo := range p.FactoresFamilia {
		if codigo == "" || factores[codigo] {
			return false
		}
		factores[codigo] = true
	}
	for _, grupo := range [][]string{p.FactoresMenores, p.TipologiasDesproteccion, p.ResponsablesDesproteccion, p.RecursosCoordinacion} {
		vistos := map[string]bool{}
		for _, codigo := range grupo {
			if codigo == "" || vistos[codigo] {
				return false
			}
			vistos[codigo] = true
		}
	}
	return true
}

func subprogramaValido(s Subprograma) bool { return s == Riesgo || s == Reunificacion }

func motivoValido(m Motivo) bool {
	switch m {
	case "", ObjetivosCumplidos, ObjetivosParciales, Rechazo, NoAdherencia, Traslado, MayoriaEdad, PropuestaDSR, MedidaProtectora, IngresoCentro, OtrasCausas:
		return true
	default:
		return false
	}
}
