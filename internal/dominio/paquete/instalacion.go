package paquete

import (
	"errors"
	"regexp"
	"slices"
)

const (
	TipoIdioma = "idioma"
	TipoTema   = "tema"
)

var ErrPaqueteInvalido = errors.New("paquete.archivo_invalido")
var ErrConflicto = errors.New("paquete.conflicto")
var ErrPermiso = errors.New("paquete.acceso_denegado")
var ErrAntivirus = errors.New("paquete.antivirus_no_disponible")
var ErrInfectado = errors.New("paquete.infectado")
var ErrPreparacion = errors.New("paquete.preparacion_caducada")
var ErrOcupado = errors.New("paquete.ocupado")

// ContenidoInstalacion es una versión validada aún sin publicar. Sus archivos
// no se sirven hasta registrar y activar la referencia con auditoría.
type ContenidoInstalacion struct {
	Tipo, Codigo, Version, Huella string
	Archivos                      map[string][]byte
}

type Actor struct {
	UsuarioID string
	Perfiles  []string
}

var usuarioInstalacion = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var huellaInstalacion = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (a Actor) PuedeAdministrar() bool {
	return usuarioInstalacion.MatchString(a.UsuarioID) && a.UsuarioID != "00000000-0000-0000-0000-000000000000" && slices.Contains(a.Perfiles, "administracion")
}

func IdentificadorValido(tipo, codigo string) bool {
	return tipo == TipoIdioma && CodigoIdiomaValido(codigo) || tipo == TipoTema && NombreTemaValido(codigo)
}

// VersionInstalada describe contenido verificado en disco; no acredita activación.
type VersionInstalada struct {
	Tipo    string `json:"tipo"`
	Codigo  string `json:"codigo"`
	Version string `json:"version_contenido"`
	Huella  string `json:"huella"`
	Ruta    string `json:"-"`
}

func (v VersionInstalada) Valida() bool {
	return IdentificadorValido(v.Tipo, v.Codigo) && VersionValida(v.Version) && huellaInstalacion.MatchString(v.Huella)
}

type Referencia struct {
	VersionInstalada
	Revision int64 `json:"version"`
}

type Preparacion struct {
	VersionInstalada
	Ticket string `json:"ticket"`
}
