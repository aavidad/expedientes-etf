package paquete

import "errors"

var ErrPublicacion = errors.New("paquete.publicacion_invalida")

type Conflicto struct {
	Actual int
	Codigo string
}

func (*Conflicto) Error() string { return "catalogo.conflicto" }

type ReferenciaCatalogo struct {
	Codigo    string `json:"codigo"`
	Version   int    `json:"version"`
	Contenido string `json:"version_contenido"`
	Ruta      string `json:"ruta_relativa"`
	Huella    string `json:"huella_sha256"`
}
type Catalogos struct {
	Referencias map[string]int               `json:"referencias"`
	Textos      map[string]map[string]string `json:"textos"`
}
type Preparada struct {
	Anteriores  map[string]int
	Referencias []ReferenciaCatalogo
}
