package documento

import "slices"

const Anexo1 = "consentimiento/anexo1"
const Anexo2 = "consentimiento/anexo2"

func EsConsentimiento(tipo string) bool { return tipo == Anexo1 || tipo == Anexo2 }

// AutorizarEmision mantiene los documentos de coordinación separados del consentimiento.
func (a Actor) AutorizarEmision(tipo string) error {
	if !EsConsentimiento(tipo) {
		return a.AutorizarEntrada()
	}
	if !IDValido(a.UsuarioID) {
		return ErrAcceso
	}
	if slices.Contains(a.Perfiles, "coordinacion") || slices.Contains(a.Perfiles, "profesional_etf") && IDValido(a.EquipoID) {
		return nil
	}
	return ErrAcceso
}
