package main

import (
	"crypto/rand"
	"crypto/x509"
	"net/netip"
	"os"
	"strings"
	"time"

	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ldap"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type relojIdentidad struct{}

func (relojIdentidad) Ahora() time.Time { return time.Now().UTC() }

type azarIdentidad struct{}

func (azarIdentidad) Llenar(b []byte) error { _, err := rand.Read(b); return err }

func configurarIdentidad(base *postgres.Base, leer func(string) string) (adaptadorHTTP.ConfiguracionSesion, error) {
	clave := make([]byte, 32)
	_, _ = rand.Read(clave)
	cfg := adaptadorHTTP.ConfiguracionSesion{Origen: leer("ETF_HTTP_ORIGEN"), ClaveCSRF: clave}
	proxies, err := leerProxies(leer("ETF_HTTP_PROXIES"))
	if err != nil {
		return cfg, dominio.ErrConfiguracion
	}
	cfg.Proxies = proxies
	politica, err := politicaSesion(leer)
	if err != nil {
		return cfg, err
	}
	ca, err := os.ReadFile(leer("ETF_AD_CA_RUTA"))
	if err != nil {
		return cfg, dominio.ErrConfiguracion
	}
	autoridades := x509.NewCertPool()
	if !autoridades.AppendCertsFromPEM(ca) {
		return cfg, dominio.ErrConfiguracion
	}
	directorio, err := ldap.Nuevo(ldap.Configuracion{
		URL: leer("ETF_AD_URL"), BaseDN: leer("ETF_AD_BASE_DN"), FormatoBind: leer("ETF_AD_FORMATO_BIND"),
		Autoridades: autoridades, TiempoLimite: 5 * time.Second,
	})
	if err != nil {
		return cfg, err
	}
	limitador, err := identidad.NuevoLimitador(dominio.PoliticaIntentos{Ventana: 15 * time.Minute, PorUsuario: 5, PorIP: 30, MaxClaves: 10000}, dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 100, MaxClaves: 10000})
	if err != nil {
		return cfg, err
	}
	usuarios, err := postgres.NuevosUsuariosConAdministrador(base, leer("ETF_PRIMER_ADMINISTRADOR"))
	if err != nil {
		return cfg, err
	}
	cfg.Servicio, err = identidad.NuevoServicio(directorio, usuarios, postgres.NuevasSesiones(base), relojIdentidad{}, azarIdentidad{}, politica, limitador, base)
	return cfg, err
}

// leerProxies interpreta ETF_HTTP_PROXIES: redes CIDR separadas por comas de
// los proxies HTTPS cuya cabecera X-Forwarded-For se acepta. Vacía: ninguno.
func leerProxies(valor string) ([]netip.Prefix, error) {
	var redes []netip.Prefix
	for _, parte := range strings.Split(valor, ",") {
		if parte = strings.TrimSpace(parte); parte == "" {
			continue
		}
		red, err := netip.ParsePrefix(parte)
		if err != nil || red != red.Masked() {
			return nil, dominio.ErrConfiguracion
		}
		redes = append(redes, red)
	}
	return redes, nil
}

// Los valores de instalación son configurables; la política impide sesiones
// sin límite y rechaza duraciones que no permiten el aviso de dos minutos.
func politicaSesion(leer func(string) string) (dominio.PoliticaSesion, error) {
	p := dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: 8 * time.Hour}
	for _, v := range []struct {
		clave   string
		destino *time.Duration
	}{{"ETF_SESION_INACTIVIDAD", &p.Inactividad}, {"ETF_SESION_MAXIMA", &p.Maxima}} {
		if texto := leer(v.clave); texto != "" {
			n, err := time.ParseDuration(texto)
			if err != nil {
				return p, dominio.ErrConfiguracion
			}
			*v.destino = n
		}
	}
	if p.Inactividad <= 2*time.Minute || p.Maxima > 24*time.Hour {
		return p, dominio.ErrConfiguracion
	}
	return p, p.Validar()
}
