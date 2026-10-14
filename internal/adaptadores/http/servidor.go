package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	entradaExpediente "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
)

// Nuevo recibe sólo puertos de entrada y los recursos web desde composición.
// Opciones agrupa la configuración de cada módulo que publica rutas HTTP.
// Un módulo nuevo añade aquí su campo en lugar de alargar la firma de Nuevo.
type Opciones struct {
	Bandeja      entradaExpediente.Bandeja
	Catalogos    ConfiguracionCatalogos
	Sesion       ConfiguracionSesion
	Instalacion  ConfiguracionInstalacion
	Derivaciones ConfiguracionDerivaciones
	Documentos   ConfiguracionDocumentos
	Usuarios     ConfiguracionUsuarios
	Expedientes  ConfiguracionAceptaciones
	SMI          ConfiguracionSMI
	Actividad    ConfiguracionActividad
	SPAF         ConfiguracionSPAF
	Modulos      []Modulo
	Equipo       entradaExpediente.BandejaEquipo
}

func rutasPlataforma(paquetes entrada.Paquetes, web fs.FS, idioma, tema string, avisos []string, comprobarBase func(context.Context) error, sesion *sesionHTTP) []Ruta {
	mux := &muxHTTP{soloRutas: true}
	mux.declarar(http.HandlerFunc(sesion.servir), reglaRuta{Patron: "/api/sesion", Metodos: []string{http.MethodGet}, Publica: true, Sesion: true}, reglaRuta{Patron: "/api/sesion", Metodos: []string{http.MethodPost, http.MethodDelete}, Publica: true, CSRF: true, Sesion: true})
	// El tema mínimo aún no incorpora un icono de aplicación.
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), reglaRuta{Patron: "/favicon.ico", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		estado := http.StatusOK
		datos := map[string]any{"estado": "ok"}
		if comprobarBase != nil {
			ctx, cancelar := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancelar()
			datos["base"] = "ok"
			if err := comprobarBase(ctx); err != nil {
				estado = http.StatusServiceUnavailable
				datos["estado"], datos["base"] = "degradado", "no_disponible"
			}
		}
		if len(avisos) > 0 {
			datos["avisos"] = avisos
		}
		responderJSON(w, estado, datos)
	}), reglaRuta{Patron: "/salud", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idiomas, err := paquetes.Idiomas(r.Context())
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		responderJSON(w, http.StatusOK, idiomas)
	}), reglaRuta{Patron: "/api/idiomas", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		textos, err := paquetes.Textos(r.Context(), r.PathValue("codigo"))
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		responderJSON(w, http.StatusOK, textos)
	}), reglaRuta{Patron: "/api/idiomas/{codigo}/textos", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		temas, err := paquetes.Temas(r.Context())
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		responderJSON(w, http.StatusOK, temas)
	}), reglaRuta{Patron: "/api/temas", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recurso, err := paquetes.RecursoTema(r.Context(), r.PathValue("nombre"), r.PathValue("recurso"))
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		w.Header().Set("Content-Type", recurso.Tipo)
		_, _ = w.Write(recurso.Contenido)
	}), reglaRuta{Patron: "/api/temas/{nombre}/recursos/{recurso...}", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens, err := paquetes.Tokens(r.Context(), r.PathValue("nombre"))
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		claves := make([]string, 0, len(tokens))
		for clave := range tokens {
			claves = append(claves, clave)
		}
		sort.Strings(claves)
		var css strings.Builder
		css.WriteString(":root{\n")
		for _, clave := range claves {
			css.WriteString("--" + strings.ReplaceAll(clave, ".", "-") + ":" + tokens[clave] + ";\n")
		}
		css.WriteString("}\n")
		temaActual, err := paquetes.Tema(r.Context(), r.PathValue("nombre"))
		if err != nil {
			responderError(w, http.StatusNotFound, "paquete.no_disponible")
			return
		}
		for _, tipo := range []string{"texto", "titulo"} {
			if recurso, ok := temaActual.Fuentes[tipo]; ok {
				familia := strings.TrimSpace(strings.SplitN(tokens["fuente."+tipo], ",", 2)[0])
				css.WriteString("@font-face{font-family:" + familia + ";src:url('/api/temas/" + temaActual.Identificador + "/recursos/" + recurso + "') format('woff2');font-weight:" + tokens["peso.normal"] + " " + tokens["peso.negrita"] + ";font-display:swap;}\n")
			}
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(css.String()))
	}), reglaRuta{Patron: "/api/temas/{nombre}/tokens", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	pagina := func(w http.ResponseWriter, r *http.Request) {
		textos, err := paquetes.Textos(r.Context(), idioma)
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		idiomaActual := idioma
		lista, err := paquetes.Idiomas(r.Context())
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		encontrado := false
		for _, dato := range lista {
			if dato.Codigo == idioma {
				encontrado = true
				break
			}
		}
		if !encontrado {
			for _, dato := range lista {
				if dato.Predeterminado {
					idiomaActual = dato.Codigo
					break
				}
			}
		}
		temaActual, err := paquetes.Tema(r.Context(), tema)
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
			return
		}
		contenido, err := fs.ReadFile(web, "index.html")
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "web.no_disponible")
			return
		}
		nonce := base64.RawStdEncoding.EncodeToString(claveNonce())
		plantilla := strings.Replace(string(contenido), "</head>", `<meta property="csp-nonce" nonce="{{.Nonce}}" /><meta name="cargando" content="{{.Cargando}}" /></head>`, 1)
		pagina, err := template.New("inicio").Parse(plantilla)
		if err != nil {
			responderError(w, http.StatusServiceUnavailable, "web.no_disponible")
			return
		}
		var salida bytes.Buffer
		if err := pagina.Execute(&salida, struct{ Idioma, Tema, Titulo, ErrorCarga, Reintentar, Nonce, Cargando string }{
			idiomaActual, temaActual.Identificador, textos["comun.titulo"], textos["comun.error_carga"], textos["comun.reintentar"], nonce, textos["comun.cargando"],
		}); err != nil {
			responderError(w, http.StatusServiceUnavailable, "web.no_disponible")
			return
		}
		w.Header().Set("Content-Security-Policy", politicaContenido+"; style-src 'self' 'nonce-"+nonce+"'")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(salida.Bytes())
	}
	for _, ruta := range []string{"/{$}", "/entrar", "/pendiente"} {
		p := pantalla(ruta, "")
		mux.declarar(http.HandlerFunc(pagina), p.ReglaRuta)
	}
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sólo ficheros de Vite, sin índices de directorio ni fallback para API.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			nombre := strings.TrimPrefix(r.URL.Path, "/")
			if fs.ValidPath(nombre) && !strings.Contains(nombre, "/.") {
				tipo := map[string]string{".js": "text/javascript", ".css": "text/css"}[path.Ext(nombre)]
				contenido, err := fs.ReadFile(web, nombre)
				if tipo != "" && err == nil {
					w.Header().Set("Content-Type", tipo+"; charset=utf-8")
					_, _ = w.Write(contenido)
					return
				}
			}
		}
		responderError(w, http.StatusNotFound, "http.no_encontrado")
	}), reglaRuta{Patron: "/assets/{recurso...}", Metodos: []string{http.MethodGet, http.MethodHead}, Publica: true})
	return append(mux.rutas, rutasPWA(paquetes, web, idioma, tema)...)
}

// Nuevo mantiene la firma de los consumidores existentes. La composición del
// ejecutable usa NuevoValidado para devolver un error de registro al arrancar.
func Nuevo(paquetes entrada.Paquetes, web fs.FS, idioma, tema string, avisos []string, registro *slog.Logger, comprobarBase func(context.Context) error, opciones Opciones) http.Handler {
	handler, err := NuevoValidado(paquetes, web, idioma, tema, avisos, registro, comprobarBase, opciones)
	if err != nil {
		panic(err)
	}
	return handler
}

func NuevoValidado(paquetes entrada.Paquetes, web fs.FS, idioma, tema string, avisos []string, registro *slog.Logger, comprobarBase func(context.Context) error, opciones Opciones) (http.Handler, error) {
	mux, sesion, err := componerMux(paquetes, web, idioma, tema, avisos, comprobarBase, opciones)
	if err != nil {
		return nil, err
	}
	autorizado := sesion.autorizar(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		w.Header().Set("Content-Security-Policy", politicaContenido+"; style-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if r.TLS != nil || sesion.origen != "" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		respuesta := &respuestaRegistrada{ResponseWriter: w, estado: http.StatusOK}
		autorizado.ServeHTTP(respuesta, r)
		// La plantilla de ruta omite valores, consultas, cabeceras e IP del cliente.
		registro.Info("http.peticion", "ruta", r.Pattern, "estado", respuesta.estado, "duracion_ms", time.Since(inicio).Milliseconds())
	}), nil
}

type respuestaRegistrada struct {
	http.ResponseWriter
	estado int
}

func (r *respuestaRegistrada) WriteHeader(estado int) {
	r.estado = estado
	r.ResponseWriter.WriteHeader(estado)
}

func responderJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(datos)
}

func responderError(w http.ResponseWriter, estado int, clave string) {
	responderJSON(w, estado, map[string]any{"error": map[string]string{"clave": clave}})
}

const politicaContenido = "default-src 'none'; script-src 'self'; connect-src 'self'; font-src 'self'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; manifest-src 'self'; worker-src 'self'"

func claveNonce() []byte {
	b := make([]byte, 32)
	// crypto/rand.Read no devuelve error en Go 1.25: termina si falla el generador.
	_, _ = rand.Read(b)
	return b
}
