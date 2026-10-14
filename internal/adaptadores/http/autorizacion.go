package http

import (
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"net/http"
	"path"
	"slices"
	"strings"
)

// tablaRutas conserva la vista de permisos usada por las pruebas de autorización.
// Se deriva del registro de los módulos; no declara rutas adicionales.
func tablaRutas() []reglaRuta {
	return reglasDeRutas(modulosPredeterminados(Opciones{}, nuevaSesionHTTP(ConfiguracionSesion{})))
}

// muxHTTP conserva todas las rutas realmente registradas para probar cobertura.
type muxHTTP struct {
	*http.ServeMux
	patrones  []string
	rutas     []Ruta
	soloRutas bool
}

func nuevoMuxHTTP() *muxHTTP { return &muxHTTP{ServeMux: http.NewServeMux()} }
func (m *muxHTTP) Handle(patron string, handler http.Handler) {
	m.patrones = append(m.patrones, patron)
	m.ServeMux.Handle(patron, handler)
}
func (m *muxHTTP) HandleFunc(patron string, handler http.HandlerFunc) { m.Handle(patron, handler) }

func (s *sesionHTTP) autorizar(siguiente http.Handler) http.Handler {
	mux := http.NewServeMux()
	porPatron := make(map[string][]reglaRuta)
	for _, regla := range s.reglasRutas() {
		porPatron[regla.Patron] = append(porPatron[regla.Patron], regla)
	}
	for patron, reglas := range porPatron {
		mux.HandleFunc(patron, func(w http.ResponseWriter, r *http.Request) {
			for _, regla := range reglas {
				if slices.Contains(regla.Metodos, r.Method) {
					s.autorizarRegla(w, r, regla, siguiente)
					return
				}
			}
			var metodos []string
			for _, regla := range reglas {
				metodos = append(metodos, regla.Metodos...)
			}
			w.Header().Set("Allow", strings.Join(metodos, ", "))
			if reglas[0].Sesion {
				s.rechazar(w, r, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
				return
			}
			responderError(w, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No hay regla genérica para pantallas ni para familias de rutas API.
		// Se rechazan variantes que el mux redirigiría a un camino autorizado.
		limpio := path.Clean(r.URL.Path)
		if r.URL.Path != limpio && (limpio == "/" || r.URL.Path != limpio+"/") {
			responderError(w, http.StatusNotFound, "http.no_encontrado")
			return
		}
		_, patron := mux.Handler(r)
		if patron == "" {
			responderError(w, http.StatusForbidden, "identidad.acceso_denegado")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *sesionHTTP) autorizarRegla(w http.ResponseWriter, r *http.Request, regla reglaRuta, siguiente http.Handler) {
	identificador := unicaCookie(r, cookieSesion)
	if !regla.Publica {
		if s.servicio == nil {
			responderError(w, http.StatusForbidden, "identidad.acceso_denegado")
			return
		}
		ip, valida := s.origenCliente(r)
		if !valida {
			responderError(w, http.StatusBadRequest, "identidad.peticion_no_valida")
			return
		}
		r = r.WithContext(entrada.ConOrigen(r.Context(), ip.As16()))
		if identificador == "" {
			// El rechazo anónimo escribe en la cadena de auditoría: antes se
			// gasta el presupuesto por origen para que no crezca sin límite.
			var admitida bool
			if r, admitida = s.admitirSesion(w, r); !admitida {
				return
			}
			s.rechazarAcceso(w, r, http.StatusUnauthorized, "identidad.sesion_no_valida")
			return
		}
		a, err := s.servicio.Consultar(r.Context(), identificador)
		if err != nil {
			errorSesion(w, err)
			return
		}
		r = r.WithContext(entrada.ConAcceso(r.Context(), a))
		permitido := false
		for _, perfil := range regla.Perfiles {
			if a.Permite(perfil) && (!regla.ExigeAmbito || perfil != "profesional_etf" || a.Usuario.EquipoID != "") {
				permitido = true
				break
			}
		}
		if !permitido {
			s.rechazarAcceso(w, r, http.StatusForbidden, "identidad.acceso_denegado")
			return
		}
	}
	if regla.Sesion {
		var admitida bool
		r, admitida = s.admitirSesion(w, r)
		if !admitida {
			return
		}
	}
	if regla.CSRF {
		secreto := identificador
		if regla.Publica && secreto == "" {
			secreto = unicaCookie(r, cookieCSRF)
		}
		if !s.proteger(r, secreto) {
			if regla.Publica {
				s.rechazar(w, r, http.StatusForbidden, "identidad.csrf_no_valido")
			} else {
				s.rechazarAcceso(w, r, http.StatusForbidden, "identidad.csrf_no_valido")
			}
			return
		}
	}
	siguiente.ServeHTTP(w, r)
}

func (s *sesionHTTP) rechazarAcceso(w http.ResponseWriter, r *http.Request, estado int, clave string) {
	if err := s.servicio.RegistrarRechazo(r.Context(), "identidad.autorizacion", "denegado"); err != nil {
		errorSesion(w, err)
		return
	}
	responderError(w, estado, clave)
}
