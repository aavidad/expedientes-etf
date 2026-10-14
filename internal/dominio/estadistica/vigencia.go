package estadistica

import "time"

// ValorEn selecciona sin modificar ni ordenar la historia. Un hueco es sin dato;
// un solapamiento es un error, nunca una elección por orden de entrada.
func (h Historia[T]) ValorEn(corte, conocidoEn time.Time) (Dato[T], string, error) {
	activas, err := h.activas(conocidoEn)
	if err != nil {
		return Dato[T]{}, "", err
	}
	for _, r := range activas {
		if r.Vigencia.contiene(corte) {
			return r.Dato, r.ID, nil
		}
	}
	return Dato[T]{}, "", nil
}

func (h Historia[T]) activas(conocidoEn time.Time) ([]Revision[T], error) {
	porID := make(map[string]Revision[T], len(h))
	for _, r := range h {
		if r.ID == "" || !r.Vigencia.valida() || r.RegistradoEn.IsZero() {
			return nil, ErrHistoria
		}
		if _, existe := porID[r.ID]; existe {
			return nil, ErrHistoria
		}
		porID[r.ID] = r
	}
	sustituidas := make(map[string]bool)
	for _, r := range h {
		if r.Sustituye == "" {
			continue
		}
		anterior, existe := porID[r.Sustituye]
		if !existe || !r.RegistradoEn.After(anterior.RegistradoEn) {
			return nil, ErrHistoria
		}
		if !r.RegistradoEn.After(conocidoEn) {
			if sustituidas[r.Sustituye] {
				return nil, ErrHistoria
			}
			sustituidas[r.Sustituye] = true
		}
	}
	var activas []Revision[T]
	for _, r := range h {
		if r.RegistradoEn.After(conocidoEn) || sustituidas[r.ID] {
			continue
		}
		for _, otra := range activas {
			if r.Vigencia.cruza(fecha(otra.Vigencia.Desde), finInclusivo(otra.Vigencia.Hasta)) {
				return nil, ErrSolapamiento
			}
		}
		activas = append(activas, r)
	}
	return activas, nil
}

func finInclusivo(hasta time.Time) time.Time {
	if hasta.IsZero() {
		return time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	}
	return fecha(hasta).AddDate(0, 0, -1)
}

// MesesCompletos compara fechas civiles, sin dividir días entre 30 ni contar
// el mes incompleto. Un 31/01 a 28/02 tiene cero meses completos.
func MesesCompletos(inicio, fin time.Time) (int, error) {
	inicio, fin = fecha(inicio), fecha(fin)
	if inicio.IsZero() || fin.IsZero() || fin.Before(inicio) {
		return 0, ErrEntrada
	}
	meses := (fin.Year()-inicio.Year())*12 + int(fin.Month()-inicio.Month())
	if fin.Day() < inicio.Day() {
		meses--
	}
	return meses, nil
}
