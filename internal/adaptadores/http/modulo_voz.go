package http

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"

	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
)

// ModuloVoz sirve únicamente recursos editoriales construidos, sin sintetizar en ejecución.
type ModuloVoz struct {
	Audios     fs.FS
	Documentos fs.FS
}

var ficheroVoz = regexp.MustCompile(`^[a-f0-9]{64}\.mp3$`)
var imagenManual = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpg|jpeg|webp|pdf)$`)

func (m ModuloVoz) Rutas() []Ruta {
	perfiles := []string{"administracion", "coordinacion", "profesional_etf", "consulta"}
	return []Ruta{
		pantalla("/ayuda", "voz.titulo", perfiles...),
		{ReglaRuta: lectura("/api/ayuda/manuales", perfiles...), Handler: http.HandlerFunc(m.manuales)},
		{ReglaRuta: lectura("/manuales/recursos/{recurso}", perfiles...), Handler: http.HandlerFunc(m.imagen)},
		{ReglaRuta: lectura("/voz/{grupo}/{recurso}", perfiles...), Handler: http.HandlerFunc(m.audio)},
	}
}

func manualesVozPermitidos(r *http.Request) map[string]bool {
	nombres := map[string]string{"profesional_etf": "profesionales", "coordinacion": "coordinacion", "administracion": "administracion"}
	permitidos := map[string]bool{}
	for _, perfil := range identidad.AccesoActual(r.Context()).Usuario.Perfiles {
		if nombre, ok := nombres[string(perfil)]; ok {
			permitidos[nombre] = true
		}
	}
	return permitidos
}

func (m ModuloVoz) manuales(w http.ResponseWriter, r *http.Request) {
	resultado := []json.RawMessage{}
	permitidos := manualesVozPermitidos(r)
	// El orden es estable también para personas con varios perfiles.
	for _, nombre := range []string{"profesionales", "coordinacion", "administracion"} {
		if !permitidos[nombre] {
			continue
		}
		contenido, err := leerVoz(m.Documentos, "manuales/web/"+nombre+".json")
		var manual map[string]json.RawMessage
		if err != nil || json.Unmarshal(contenido, &manual) != nil || manual == nil {
			responderError(w, http.StatusServiceUnavailable, "voz.manuales_no_disponibles")
			return
		}
		manual["nombre"], _ = json.Marshal(nombre)
		dato, _ := json.Marshal(manual)
		resultado = append(resultado, dato)
	}
	responderJSON(w, http.StatusOK, resultado)
}

func leerVoz(sistema fs.FS, nombre string) ([]byte, error) {
	if sistema == nil {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(sistema, nombre)
}

func (m ModuloVoz) audio(w http.ResponseWriter, r *http.Request) {
	grupo, nombre := r.PathValue("grupo"), r.PathValue("recurso")
	if grupo != "ayuda" && grupo != "manuales" {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	var datos map[string]json.RawMessage
	contenido, err := leerVoz(m.Audios, grupo+"/manifiesto.json")
	if err != nil || json.Unmarshal(contenido, &datos) != nil {
		datos = map[string]json.RawMessage{}
	}
	permitidos := manualesVozPermitidos(r)
	for id, dato := range datos {
		if grupo == "manuales" {
			permitido := false
			for manual := range permitidos {
				permitido = permitido || strings.HasPrefix(id, manual+"-s")
			}
			if !permitido {
				delete(datos, id)
				continue
			}
		}
		var entrada struct {
			Fichero string `json:"fichero"`
		}
		if json.Unmarshal(dato, &entrada) != nil || !ficheroVoz.MatchString(entrada.Fichero) {
			delete(datos, id)
			continue
		}
		archivo, err := fs.Stat(m.Audios, grupo+"/"+entrada.Fichero)
		if err != nil || !archivo.Mode().IsRegular() || archivo.Size() == 0 {
			delete(datos, id)
		}
	}
	if nombre == "manifiesto.json" {
		responderJSON(w, http.StatusOK, datos)
		return
	}
	if !ficheroVoz.MatchString(nombre) {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	permitido := false
	for _, dato := range datos {
		var entrada struct {
			Fichero string `json:"fichero"`
		}
		if json.Unmarshal(dato, &entrada) == nil && entrada.Fichero == nombre {
			permitido = true
			break
		}
	}
	if !permitido {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	contenido, err = leerVoz(m.Audios, grupo+"/"+nombre)
	if err != nil {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	http.ServeContent(w, r, nombre, time.Time{}, bytes.NewReader(contenido))
}

func (m ModuloVoz) imagen(w http.ResponseWriter, r *http.Request) {
	nombre := r.PathValue("recurso")
	if !imagenManual.MatchString(nombre) {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	contenido, err := leerVoz(m.Documentos, "manuales/web/recursos.json")
	var recursos map[string]struct {
		Ruta     string   `json:"ruta"`
		Manuales []string `json:"manuales"`
	}
	if err != nil || json.Unmarshal(contenido, &recursos) != nil {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	recurso := recursos[nombre]
	permitido := false
	permitidos := manualesVozPermitidos(r)
	for _, manual := range recurso.Manuales {
		permitido = permitido || permitidos[manual]
	}
	if !permitido {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	ruta := recurso.Ruta
	if !fs.ValidPath(ruta) || !(strings.HasPrefix(ruta, "capturas/") || strings.HasPrefix(ruta, "manuales/")) {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	contenido, err = leerVoz(m.Documentos, ruta)
	if err != nil {
		responderError(w, http.StatusNotFound, "voz.no_disponible")
		return
	}
	http.ServeContent(w, r, nombre, time.Time{}, bytes.NewReader(contenido))
}
