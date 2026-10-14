package ldap

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

func certificadoLocal(t *testing.T, ips ...net.IP) (*x509.CertPool, []byte, []byte) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "ldap.etf.interno"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true, IsCA: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	cert.IPAddresses = append(cert.IPAddresses, ips...)
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	publico := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privado, err := x509.MarshalPKCS8PrivateKey(clave)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(publico) {
		t.Fatal("certificado no cargado")
	}
	return raices, publico, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privado})
}

func configLocal(t *testing.T, direccion string) Configuracion {
	raices, _, _ := certificadoLocal(t)
	return Configuracion{URL: "ldaps://" + direccion, BaseDN: "dc=etf,dc=interno", FormatoBind: "{usuario}@etf.interno", Autoridades: raices, TiempoLimite: time.Second}
}

func TestConfiguracionExigeLDAPS(t *testing.T) {
	for _, caso := range []string{"ldap", "secreto_url", "sin_ca", "sin_plazo", "filtro_atributo", "bind", "dn", "puerto"} {
		t.Run(caso, func(t *testing.T) {
			c := configLocal(t, "127.0.0.1:1636")
			switch caso {
			case "ldap":
				c.URL = "ldap://127.0.0.1:1636"
			case "secreto_url":
				c.URL = "ldaps://ines:clave@127.0.0.1:1636"
			case "sin_ca":
				c.Autoridades = nil
			case "sin_plazo":
				c.TiempoLimite = 0
			case "filtro_atributo":
				c.AtributoCuenta = "uid)(objectClass=*"
			case "bind":
				c.FormatoBind = "ines@etf.interno"
			case "dn":
				c.BaseDN = "dc"
			case "puerto":
				c.URL = "ldaps://127.0.0.1:99999"
			}
			if _, err := Nuevo(c); !errors.Is(err, dominio.ErrConfiguracion) {
				t.Fatal(err)
			}
		})
	}
}

func TestVaciaEInyeccionNoAbrenConexion(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	d, err := Nuevo(configLocal(t, l.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}
	for _, cred := range [][2]string{{"ines.moreno", ""}, {"*)(uid=*)", "clave"}, {"ines@etf.interno", "clave"}} {
		if _, err := d.Autenticar(context.Background(), cred[0], cred[1]); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
			t.Fatal(err)
		}
	}
	if err := l.(*net.TCPListener).SetDeadline(time.Now().Add(30 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if c, err := l.Accept(); err == nil {
		c.Close()
		t.Fatal("se llamó al directorio")
	}
}

func TestTiempoTotalYCancelacion(t *testing.T) {
	for _, cancelado := range []bool{false, true} {
		t.Run(map[bool]string{false: "plazo", true: "cancelacion"}[cancelado], func(t *testing.T) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			aceptada := make(chan net.Conn, 1)
			go func() {
				c, err := l.Accept()
				if err == nil {
					aceptada <- c
				}
			}()
			c := configLocal(t, l.Addr().String())
			c.TiempoLimite = 100 * time.Millisecond
			d, err := Nuevo(c)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			desde := time.Now()
			fin := make(chan error, 1)
			go func() { _, err := d.Autenticar(ctx, "ines.moreno", "clave-local"); fin <- err }()
			var conexion net.Conn
			select {
			case conexion = <-aceptada:
				defer conexion.Close()
			case <-time.After(time.Second):
				t.Fatal("no conectó")
			}
			if cancelado {
				cancel()
			}
			select {
			case err := <-fin:
				esperado := error(dominio.ErrDirectorioNoDisponible)
				if cancelado {
					esperado = context.Canceled
				}
				if !errors.Is(err, esperado) {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("tiempo límite incumplido")
			}
			if time.Since(desde) > time.Second {
				t.Fatal("plazo superado")
			}
		})
	}
}

func TestDirectorioCaidoYGUID(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Nuevo(configLocal(t, l.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	if _, err := d.Autenticar(context.Background(), "ines.moreno", "clave-local"); !errors.Is(err, dominio.ErrDirectorioNoDisponible) {
		t.Fatal(err)
	}
	bytes := []byte{0x33, 0x22, 0x11, 0x00, 0x55, 0x44, 0x77, 0x66, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	guid, err := guidAD(bytes)
	esperado := dominio.GUID{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	if err != nil || guid != esperado {
		t.Fatalf("%x %v", guid, err)
	}
	for _, invalido := range [][]byte{nil, make([]byte, 15), make([]byte, 16), make([]byte, 17)} {
		if _, err := guidAD(invalido); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
			t.Fatal(err)
		}
	}
}
