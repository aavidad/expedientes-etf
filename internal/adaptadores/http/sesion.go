package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
)

const cookieSesion = "__Host-etf-sesion"
const cookieCSRF = "__Host-etf-csrf"

// Origen es la dirección HTTPS pública exacta, configurada por el operador.
// Nunca se confía en Host ni en cabeceras Forwarded enviadas por el cliente.
// Proxies son las redes de los proxies HTTPS declarados por el operador: sólo
// de ellas se acepta X-Forwarded-For para saber qué cliente pide.
type ConfiguracionSesion struct {
	Servicio  identidad.Sesiones
	Origen    string
	ClaveCSRF []byte
	Proxies   []netip.Prefix
}
type sesionHTTP struct {
	servicio identidad.Sesiones
	origen   string
	clave    []byte
	proxies  []netip.Prefix
	rutas    []reglaRuta
}

func nuevaSesionHTTP(c ConfiguracionSesion) *sesionHTTP {
	u, err := url.Parse(c.Origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || len(c.ClaveCSRF) != 32 {
		return &sesionHTTP{}
	}
	return &sesionHTTP{servicio: c.Servicio, origen: c.Origen, clave: append([]byte(nil), c.ClaveCSRF...), proxies: append([]netip.Prefix(nil), c.Proxies...)}
}

func (s *sesionHTTP) esProxy(ip netip.Addr) bool {
	for _, red := range s.proxies {
		if red.Contains(ip) {
			return true
		}
	}
	return false
}

// origenCliente da la IP del cliente. Si la conexión viene de un proxy
// declarado, toma la última dirección de X-Forwarded-For que no sea otro proxy
// declarado; cualquier valor mal formado invalida la petición. Si no viene de
// un proxy, la cabecera se ignora.
func (s *sesionHTTP) origenCliente(r *http.Request) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	ip = ip.Unmap()
	if !s.esProxy(ip) {
		return ip, true
	}
	partes := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(partes) - 1; i >= 0; i-- {
		cliente, err := netip.ParseAddr(strings.TrimSpace(partes[i]))
		if err != nil || cliente.Zone() != "" {
			return netip.Addr{}, false
		}
		if cliente = cliente.Unmap(); !s.esProxy(cliente) {
			return cliente, true
		}
	}
	return netip.Addr{}, false
}
func opaco(valor string) bool {
	b, err := base64.RawURLEncoding.DecodeString(valor)
	return err == nil && len(b) == 32 && base64.RawURLEncoding.EncodeToString(b) == valor
}
func unicaCookie(r *http.Request, nombre string) string {
	valor := ""
	cantidad := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == nombre {
			valor = cookie.Value
			cantidad++
		}
	}
	if cantidad != 1 || !opaco(valor) {
		return ""
	}
	return valor
}
func ponerCookie(w http.ResponseWriter, nombre, valor string, eliminar bool) {
	c := &http.Cookie{Name: nombre, Value: valor, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
	if eliminar {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, c)
}
func (s *sesionHTTP) testigo(valor string) string {
	mac := hmac.New(sha256.New, s.clave)
	_, _ = mac.Write([]byte(valor))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *sesionHTTP) proteger(r *http.Request, valor string) bool {
	if s.origen == "" || len(s.clave) != 32 || valor == "" || len(r.Header.Values("Origin")) != 1 || r.Header.Get("Origin") != s.origen || len(r.Header.Values("X-CSRF-Token")) != 1 {
		return false
	}
	if sitio := r.Header.Get("Sec-Fetch-Site"); sitio != "" && sitio != "same-origin" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.testigo(valor))) == 1
}
func (s *sesionHTTP) admitirSesion(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	ip, valida := s.origenCliente(r)
	if !valida {
		responderError(w, http.StatusBadRequest, "identidad.peticion_no_valida")
		return r, false
	}
	r = r.WithContext(identidad.ConOrigen(r.Context(), ip.As16()))
	if s.servicio != nil {
		if err := s.servicio.AdmitirPeticion(r.Context()); err != nil {
			if err.Error() == "identidad.intentos_limitados" {
				w.Header().Set("Retry-After", "60")
				responderError(w, 429, "identidad.peticiones_limitadas")
				return r, false
			}
			errorSesion(w, err)
			return r, false
		}
	}
	return r, true
}

func (s *sesionHTTP) servir(w http.ResponseWriter, r *http.Request) {
	identificador := unicaCookie(r, cookieSesion)
	secreto := identificador
	if secreto == "" {
		secreto = unicaCookie(r, cookieCSRF)
	}
	if r.Method == http.MethodGet {
		if s.origen == "" {
			s.rechazar(w, r, 503, "identidad.configuracion_invalida")
			return
		}
		if identificador != "" {
			if s.servicio == nil {
				s.rechazar(w, r, 503, "identidad.configuracion_invalida")
				return
			}
			acceso, err := s.servicio.Consultar(identidad.ConConsultaPasiva(r.Context()), identificador)
			if err != nil {
				if err.Error() == "identidad.sesion_no_valida" || err.Error() == "identidad.usuario_no_disponible" {
					ponerCookie(w, cookieSesion, "", true)
				}
				errorSesion(w, err)
				return
			}
			s.responderAcceso(w, acceso, identificador)
			return
		}
		if secreto == "" {
			bytes := make([]byte, 32)
			_, _ = rand.Read(bytes)
			secreto = base64.RawURLEncoding.EncodeToString(bytes)
			ponerCookie(w, cookieCSRF, secreto, false)
		}
		responderJSON(w, 200, map[string]any{"estado": "sin_sesion", "csrf": s.testigo(secreto)})
		return
	}
	if s.servicio == nil {
		s.rechazar(w, r, 503, "identidad.configuracion_invalida")
		return
	}
	if r.Method == http.MethodPut {
		acceso, err := s.servicio.Consultar(r.Context(), identificador)
		if err != nil {
			errorSesion(w, err)
			return
		}
		s.responderAcceso(w, acceso, identificador)
		return
	}
	if r.Method == http.MethodDelete {
		if identificador != "" {
			if err := s.servicio.Salir(r.Context(), identificador); err != nil {
				errorSesion(w, err)
				return
			}
		}
		ponerCookie(w, cookieSesion, "", true)
		ponerCookie(w, cookieCSRF, "", true)
		w.WriteHeader(204)
		return
	}
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" {
		s.rechazar(w, r, 415, "identidad.peticion_no_valida")
		return
	}
	var datos struct {
		Cuenta     string `json:"cuenta"`
		Contrasena string `json:"contrasena"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&datos); err != nil {
		s.rechazar(w, r, 400, "identidad.peticion_no_valida")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		s.rechazar(w, r, 400, "identidad.peticion_no_valida")
		return
	}
	if len(datos.Cuenta) > 128 || len(datos.Contrasena) > 2048 {
		s.rechazar(w, r, 400, "identidad.peticion_no_valida")
		return
	}
	ip, valida := s.origenCliente(r)
	if !valida {
		s.rechazar(w, r, 400, "identidad.peticion_no_valida")
		return
	}
	abierta, err := s.servicio.Entrar(r.Context(), identidad.Inicio{Cuenta: datos.Cuenta, Contrasena: datos.Contrasena, IP: ip.As16()})
	datos.Contrasena = ""
	if err != nil {
		errorSesion(w, err)
		return
	}
	if identificador != "" {
		if err := s.servicio.Salir(r.Context(), identificador); err != nil {
			_ = s.servicio.Salir(r.Context(), abierta.Identificador)
			errorSesion(w, err)
			return
		}
	}
	ponerCookie(w, cookieSesion, abierta.Identificador, false)
	ponerCookie(w, cookieCSRF, "", true)
	s.responderAcceso(w, abierta.Acceso, abierta.Identificador)
}
func (s *sesionHTTP) responderAcceso(w http.ResponseWriter, a identidad.Acceso, identificador string) {
	cabecerasCaducidad(w, a)
	// La pantalla pendiente recibe sólo estado, sin nombre ni cuenta.
	datos := map[string]any{"estado": a.Estado, "csrf": s.testigo(identificador)}
	if string(a.Estado) == "activo" {
		datos["usuario"] = map[string]string{"id": a.Usuario.ID, "cuenta": a.Usuario.CuentaAD, "nombre": a.Usuario.NombreMostrado}
		datos["perfiles"] = a.Usuario.Perfiles
		datos["equipo_id"] = a.Usuario.EquipoID
		pantallas := []map[string]any{}
		for _, regla := range s.reglasRutas() {
			if !regla.Pantalla || regla.ClaveTitulo == "" || (!a.Usuario.RevisionRestauracion && regla.Patron == "/admin/restauracion") || (a.Usuario.RevisionRestauracion && regla.Patron != "/admin/restauracion") {
				continue
			}
			permitidos := []string{}
			for _, perfil := range regla.Perfiles {
				if a.Permite(perfil) {
					permitidos = append(permitidos, perfil)
				}
			}
			if len(permitidos) > 0 {
				pantallas = append(pantallas, map[string]any{"ruta": regla.Patron, "perfiles": permitidos, "clave": regla.ClaveTitulo})
			}
		}
		datos["pantallas"] = pantallas
	}
	responderJSON(w, 200, datos)
}
func errorSesion(w http.ResponseWriter, err error) {
	estado, clave := http.StatusServiceUnavailable, "identidad.persistencia_no_disponible"
	switch err.Error() {
	case "identidad.credenciales_invalidas":
		estado, clave = 401, err.Error()
	case "identidad.sesion_no_valida":
		estado, clave = 401, err.Error()
	case "restauracion.acceso_bloqueado":
		estado, clave = 403, err.Error()
	case "identidad.usuario_no_disponible":
		estado, clave = 403, err.Error()
	case "identidad.intentos_limitados":
		estado, clave = 429, err.Error()
		w.Header().Set("Retry-After", "60")
	case "identidad.directorio_no_disponible", "identidad.configuracion_invalida":
		clave = err.Error()
	}
	responderError(w, estado, clave)
}

// El rechazo de CSRF, JSON o método se registra mediante el puerto de entrada;
// no se llama a AD ni se acepta una identidad aportada por la petición.
func (s *sesionHTTP) rechazar(w http.ResponseWriter, r *http.Request, estado int, clave string) {
	if s.servicio != nil && (r.Method == http.MethodPost || r.Method == http.MethodDelete) {
		accion := "identidad.sesion_entrar"
		if r.Method == http.MethodDelete {
			accion = "identidad.sesion_salir"
		}
		resultado := "denegado"
		if estado >= 500 {
			resultado = "error"
		}
		if err := s.servicio.RegistrarRechazo(r.Context(), accion, resultado); err != nil {
			errorSesion(w, err)
			return
		}
	}
	responderError(w, estado, clave)
}

func cabecerasCaducidad(w http.ResponseWriter, a identidad.Acceso) {
	if a.CaducaEn.IsZero() || a.MaximaEn.IsZero() {
		return
	}
	w.Header().Set("X-ETF-Sesion-Caduca", a.CaducaEn.UTC().Format(time.RFC3339Nano))
	w.Header().Set("X-ETF-Sesion-Maxima", a.MaximaEn.UTC().Format(time.RFC3339Nano))
	w.Header().Set("X-ETF-Sesion-Ahora", time.Now().UTC().Format(time.RFC3339Nano))
}
