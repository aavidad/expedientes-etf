package ldap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
	ldap "github.com/go-ldap/ldap/v3"
)

type Configuracion struct {
	URL    string
	BaseDN string
	// FormatoBind contiene una sola sustitución {usuario}: UPN en AD,
	// o un DN con BindDN para servidores de pruebas y directorios compatibles.
	FormatoBind    string
	BindDN         bool
	AtributoCuenta string
	Autoridades    *x509.CertPool
	TiempoLimite   time.Duration
}

type Directorio struct {
	config    Configuracion
	direccion string
	servidor  string
}

var _ salida.Directorio = (*Directorio)(nil)

func Nuevo(c Configuracion) (*Directorio, error) {
	u, err := url.Parse(c.URL)
	if err != nil || u.Scheme != "ldaps" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		c.BaseDN == "" || c.TiempoLimite <= 0 || c.Autoridades == nil || len(c.Autoridades.Subjects()) == 0 ||
		strings.Count(c.FormatoBind, "{usuario}") != 1 {
		return nil, dominio.ErrConfiguracion
	}
	if _, err := ldap.ParseDN(c.BaseDN); err != nil {
		return nil, dominio.ErrConfiguracion
	}
	if c.AtributoCuenta == "" {
		c.AtributoCuenta = "sAMAccountName"
	}
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`).MatchString(c.AtributoCuenta) {
		return nil, dominio.ErrConfiguracion
	}
	puerto := u.Port()
	if puerto == "" {
		puerto = "636"
	}
	numero, err := strconv.Atoi(puerto)
	if err != nil || numero < 1 || numero > 65535 {
		return nil, dominio.ErrConfiguracion
	}
	// No compartir una autoridad mutable con quien construyó el adaptador.
	c.Autoridades = c.Autoridades.Clone()
	return &Directorio{config: c, direccion: net.JoinHostPort(u.Hostname(), puerto), servidor: u.Hostname()}, nil
}

func (d *Directorio) Autenticar(ctx context.Context, cuenta, contrasena string) (dominio.IdentidadDirectorio, error) {
	// Esta guarda precede incluso a la creación de una conexión TLS.
	if contrasena == "" {
		return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
	}
	cuenta, err := dominio.NormalizarCuenta(cuenta)
	if err != nil {
		return dominio.IdentidadDirectorio{}, err
	}
	if err := ctx.Err(); err != nil {
		return dominio.IdentidadDirectorio{}, err
	}
	llamada, cancelar := context.WithTimeout(ctx, d.config.TiempoLimite)
	defer cancelar()
	conexion, err := (&net.Dialer{}).DialContext(llamada, "tcp", d.direccion)
	if err != nil {
		return dominio.IdentidadDirectorio{}, fallo(ctx)
	}
	defer conexion.Close()
	plazo, _ := llamada.Deadline()
	if err := conexion.SetDeadline(plazo); err != nil {
		return dominio.IdentidadDirectorio{}, fallo(ctx)
	}
	// Cubre también cancelación mientras Bind/Search están esperando datos.
	parar := context.AfterFunc(llamada, func() { _ = conexion.Close() })
	defer parar()
	cifrada := tls.Client(conexion, &tls.Config{RootCAs: d.config.Autoridades, ServerName: d.servidor, MinVersion: tls.VersionTLS12})
	if err := cifrada.HandshakeContext(llamada); err != nil {
		return dominio.IdentidadDirectorio{}, fallo(ctx)
	}
	cliente := ldap.NewConn(cifrada, true)
	cliente.Start()
	defer cliente.Close()
	cliente.SetTimeout(time.Until(plazo))
	nombreBind := cuenta
	if d.config.BindDN {
		nombreBind = ldap.EscapeDN(cuenta)
	}
	nombreBind = strings.Replace(d.config.FormatoBind, "{usuario}", nombreBind, 1)
	if err := cliente.Bind(nombreBind, contrasena); err != nil {
		if ctx.Err() != nil {
			return dominio.IdentidadDirectorio{}, ctx.Err()
		}
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
		}
		return dominio.IdentidadDirectorio{}, dominio.ErrDirectorioNoDisponible
	}
	filtro := "(" + d.config.AtributoCuenta + "=" + ldap.EscapeFilter(cuenta) + ")"
	resultado, err := cliente.Search(ldap.NewSearchRequest(d.config.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		2, 0, false, filtro, []string{"objectGUID", d.config.AtributoCuenta, "displayName"}, nil))
	if err != nil {
		return dominio.IdentidadDirectorio{}, fallo(ctx)
	}
	if len(resultado.Entries) != 1 {
		return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
	}
	entrada := resultado.Entries[0]
	guids := entrada.GetRawAttributeValues("objectGUID")
	cuentas := entrada.GetAttributeValues(d.config.AtributoCuenta)
	if len(guids) != 1 || len(cuentas) != 1 {
		return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
	}
	guid, err := guidAD(guids[0])
	if err != nil {
		return dominio.IdentidadDirectorio{}, err
	}
	login, err := dominio.NormalizarCuenta(cuentas[0])
	if err != nil || login != cuenta {
		return dominio.IdentidadDirectorio{}, dominio.ErrCredencialesInvalidas
	}
	if err := llamada.Err(); err != nil {
		return dominio.IdentidadDirectorio{}, fallo(ctx)
	}
	return dominio.IdentidadDirectorio{GUID: guid, CuentaAD: login, NombreMostrado: entrada.GetAttributeValue("displayName")}, nil
}

func fallo(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return dominio.ErrDirectorioNoDisponible
}

// AD codifica los tres primeros campos del UUID en little endian.
func guidAD(bytes []byte) (dominio.GUID, error) {
	if len(bytes) != 16 {
		return dominio.GUID{}, dominio.ErrCredencialesInvalidas
	}
	guid := dominio.GUID{bytes[3], bytes[2], bytes[1], bytes[0], bytes[5], bytes[4], bytes[7], bytes[6]}
	copy(guid[8:], bytes[8:])
	if guid == (dominio.GUID{}) {
		return dominio.GUID{}, dominio.ErrCredencialesInvalidas
	}
	return guid, nil
}
