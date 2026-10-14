package paquete

import "regexp"

// Idioma describe un paquete de textos instalado; Nombre es su autodenominación.
type Idioma struct {
	Predeterminado bool   `json:"predeterminado"`
	Codigo         string `json:"codigo"`
	Nombre         string `json:"nombre"`
	Version        string `json:"version"`
}

const IdiomaPredeterminado = "es"

var codigoIdioma = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
var versionPaquete = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func CodigoIdiomaValido(codigo string) bool {
	return len(codigo) <= 63 && codigoIdioma.MatchString(codigo)
}
func VersionValida(version string) bool {
	return len(version) <= 32 && versionPaquete.MatchString(version)
}
func (i Idioma) Valido() bool {
	return CodigoIdiomaValido(i.Codigo) && len(i.Nombre) > 0 && len(i.Nombre) <= 128 && VersionValida(i.Version)
}
