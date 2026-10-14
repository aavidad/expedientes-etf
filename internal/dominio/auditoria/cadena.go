package auditoria

// ErrorVerificacion identifica el primer fallo, sin copiar datos del evento.
// Indice empieza en cero; len(eventos) indica que falla el sello final.
type ErrorVerificacion struct {
	Clave    string
	Indice   int
	EventoID uint64
}

func (e *ErrorVerificacion) Error() string { return e.Clave }

// Verificar no ordena ni repara: recorre exactamente el orden recibido.
// Los puntos delimitan el tramo completo, incluidos los extremos eliminados.
// Sólo un punto custodiado fuera del fichero acredita que no se rehízo todo.
func Verificar(eventos []Evento, anterior, final Punto) error {
	if err := anterior.Validar(); err != nil {
		return &ErrorVerificacion{Clave: "auditoria.ancla_invalida", Indice: 0}
	}
	if err := final.Validar(); err != nil || final.ID < anterior.ID {
		return &ErrorVerificacion{Clave: "auditoria.sello_invalido", Indice: len(eventos), EventoID: final.ID}
	}
	previo := anterior
	for i, evento := range eventos {
		fallo := func(clave string) error {
			return &ErrorVerificacion{Clave: clave, Indice: i, EventoID: evento.ID}
		}
		if evento.ID <= previo.ID || evento.ID > final.ID {
			return fallo("auditoria.orden_invalido")
		}
		if evento.HuellaAnterior != previo.Huella {
			return fallo("auditoria.enlace_invalido")
		}
		if err := evento.Datos.Validar(); err != nil {
			return fallo(err.Error())
		}
		if !huellaValida(evento.Huella) || calcularHuella(evento) != evento.Huella {
			return fallo("auditoria.huella_invalida")
		}
		previo = evento.Punto()
	}
	if previo != final {
		return &ErrorVerificacion{Clave: "auditoria.sello_no_coincide", Indice: len(eventos), EventoID: final.ID}
	}
	return nil
}
