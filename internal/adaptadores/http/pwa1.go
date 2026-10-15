package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
)

// rutasPWA publica sólo metadatos de aplicación y un aviso sin datos de sesión.
func rutasPWA(paquetes entrada.Paquetes, web fs.FS, idioma, tema string) []Ruta {
	mux := &muxHTTP{soloRutas: true}
	preferencias := func(r *http.Request) (string, string) {
		i, t := r.URL.Query().Get("idioma"), r.URL.Query().Get("tema")
		if i == "" {
			i = idioma
		}
		if t == "" {
			t = tema
		}
		return i, t
	}
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i, t := preferencias(r)
		textos, err := paquetes.Textos(r.Context(), i)
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		if !textosPWACompletos(textos, "pwa1.nombre", "pwa1.nombre_corto", "pwa1.descripcion") {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		actual, err := paquetes.Tema(r.Context(), t)
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		tokens, err := paquetes.Tokens(r.Context(), actual.Identificador)
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		iconos := []map[string]string{}
		for _, tamano := range []string{"192", "512"} {
			for _, uso := range []string{"any", "maskable"} {
				recurso := "iconos/etf-" + tamano + "-" + uso + ".png"
				nombre := actual.Identificador
				if _, err := paquetes.RecursoTema(r.Context(), nombre, recurso); err != nil {
					// Los temas publicados antes de PWA1 pueden carecer de iconos.
					// La marca del tema de reserva permite seguir instalando la app.
					nombre = tema
					if _, err := paquetes.RecursoTema(r.Context(), nombre, recurso); err != nil {
						nombre = "expedientes-etf"
					}
				}
				iconos = append(iconos, map[string]string{"src": "/api/temas/" + url.PathEscape(nombre) + "/recursos/" + recurso, "sizes": tamano + "x" + tamano, "type": "image/png", "purpose": uso})
			}
		}
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "/", "name": textos["pwa1.nombre"], "short_name": textos["pwa1.nombre_corto"], "description": textos["pwa1.descripcion"], "display": "standalone", "start_url": "/", "scope": "/", "theme_color": tokens["color.acento"], "background_color": tokens["color.fondo"], "icons": iconos})
	}), reglaRuta{Patron: "/manifest.webmanifest", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i, t := preferencias(r)
		actual, err := paquetes.Tema(r.Context(), t)
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		avisos, politicas, err := avisosPWA(paquetes, r)
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		if _, existe := avisos[i+"/"+actual.Identificador]; !existe {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		// El worker es idéntico byte a byte mientras no cambien catálogos ni temas:
		// así una comprobación de actualización del navegador no instala otro worker.
		contenido, err := json.Marshal(struct {
			Avisos         map[string]string `json:"avisos"`
			Predeterminado string            `json:"predeterminado"`
			Politicas      map[string]string `json:"politicas"`
		}{avisos, i + "/" + actual.Identificador, politicas})
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "web.no_disponible")
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		worker, err := fs.ReadFile(web, "service-worker.js")
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "web.no_disponible")
			return
		}
		_, _ = w.Write(bytes.Replace(worker, []byte("__CONFIGURACION__"), contenido, 1))

	}), reglaRuta{Patron: "/service-worker.js", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	return mux.rutas
}

// Los avisos contienen sólo textos publicados y tokens validados, nunca datos de sesión.
// Su CSP autoriza el estilo y el script de cada aviso por huella SHA-256, no por nonce.
func avisosPWA(paquetes entrada.Paquetes, r *http.Request) (map[string]string, map[string]string, error) {
	idiomas, err := paquetes.Idiomas(r.Context())
	if err != nil {
		return nil, nil, err
	}
	temas, err := paquetes.Temas(r.Context())
	if err != nil {
		return nil, nil, err
	}
	plantilla, err := template.New("sinConexion").Parse(`<!doctype html><html lang="{{.Idioma}}"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Titulo}}</title><style>body{margin:0;padding:{{index .Tokens "espacio.8"}};background:{{index .Tokens "color.fondo"}};color:{{index .Tokens "color.texto"}};font-family:{{index .Tokens "fuente.texto"}};font-size:{{index .Tokens "texto.normal"}}}main{padding:{{index .Tokens "espacio.8"}};background:{{index .Tokens "color.superficie"}};border-radius:{{index .Tokens "radio.tarjeta"}}}h1{font-family:{{index .Tokens "fuente.titulo"}};font-size:{{index .Tokens "texto.titulo-pagina"}};overflow-wrap:anywhere}a{display:inline-flex;align-items:center;min-height:{{index .Tokens "alto.control"}};padding:0 {{index .Tokens "espacio.6"}};background:{{index .Tokens "color.acento"}};color:{{index .Tokens "color.sobre-acento"}};border-radius:{{index .Tokens "radio.control"}}}a:focus-visible{outline:{{index .Tokens "borde.marcado"}} solid {{index .Tokens "color.foco"}};outline-offset:{{index .Tokens "espacio.3"}}}</style><main><h1>{{.Titulo}}</h1><p>{{.Ayuda}}</p><a href="/">{{.Reintentar}}</a></main><script>const datos={tipo:'preferencias',idioma:{{.Idioma}},tema:{{.Tema}}};navigator.serviceWorker.addEventListener('message',event=>{if(event.data?.tipo==='solicitar-preferencias'){if(event.ports[0])event.ports[0].postMessage(datos);else navigator.serviceWorker.controller?.postMessage(datos)}});navigator.serviceWorker.controller?.postMessage(datos);</script></html>`)
	if err != nil {
		return nil, nil, err
	}
	avisos, politicas := map[string]string{}, map[string]string{}
	for _, idioma := range idiomas {
		textos, err := paquetes.Textos(r.Context(), idioma.Codigo)
		if err != nil {
			return nil, nil, err
		}
		if !textosPWACompletos(textos, "pwa1.sin_conexion", "pwa1.sin_conexion_ayuda", "comun.reintentar") {
			return nil, nil, fs.ErrNotExist
		}
		for _, tema := range temas {
			tokens, err := paquetes.Tokens(r.Context(), tema.Identificador)
			if err != nil {
				return nil, nil, err
			}
			// El puerto de paquetes entrega tokens previamente validados. CSS evita que
			// html/template rechace las comillas permitidas en los nombres de fuente.
			estilos := map[string]template.CSS{}
			for clave, valor := range tokens {
				estilos[clave] = template.CSS(valor)
			}
			var pagina bytes.Buffer
			err = plantilla.Execute(&pagina, struct {
				Idioma, Titulo, Ayuda, Reintentar, Tema string
				Tokens                                  map[string]template.CSS
			}{idioma.Codigo, textos["pwa1.sin_conexion"], textos["pwa1.sin_conexion_ayuda"], textos["comun.reintentar"], tema.Identificador, estilos})
			if err != nil {
				return nil, nil, err
			}
			estilo, hayEstilo := contenidoEtiqueta(pagina.String(), "style")
			script, hayScript := contenidoEtiqueta(pagina.String(), "script")
			if !hayEstilo || !hayScript {
				return nil, nil, fs.ErrInvalid
			}
			clave := idioma.Codigo + "/" + tema.Identificador
			avisos[clave] = pagina.String()
			politicas[clave] = "default-src 'none'; script-src " + huellaCSP(script) + "; style-src " + huellaCSP(estilo) + "; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"
		}
	}
	return avisos, politicas, nil
}

// contenidoEtiqueta devuelve el texto de la única etiqueta indicada. Una segunda
// etiqueta, aunque llegara desde un catálogo, impide servir el aviso: la huella nunca
// autoriza contenido ajeno a la plantilla.
func contenidoEtiqueta(pagina, etiqueta string) (string, bool) {
	inicio := strings.Index(pagina, "<"+etiqueta+">")
	if inicio < 0 || strings.Count(strings.ToLower(pagina), "<"+etiqueta) != 1 {
		return "", false
	}
	inicio += len(etiqueta) + 2
	fin := strings.Index(pagina[inicio:], "</"+etiqueta+">")
	if fin < 0 {
		return "", false
	}
	return pagina[inicio : inicio+fin], true
}

func huellaCSP(contenido string) string {
	suma := sha256.Sum256([]byte(contenido))
	return "'sha256-" + base64.StdEncoding.EncodeToString(suma[:]) + "'"
}

func textosPWACompletos(textos map[string]string, claves ...string) bool {
	for _, clave := range claves {
		if strings.TrimSpace(textos[clave]) == "" {
			return false
		}
	}
	return true
}
