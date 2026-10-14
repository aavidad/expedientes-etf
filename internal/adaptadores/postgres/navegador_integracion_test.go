//go:build navegador_integracion

package postgres

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ldap"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type relojNavegador struct{}

func (relojNavegador) Ahora() time.Time { return time.Now().UTC() }

type azarNavegador struct{}

func (azarNavegador) Llenar(b []byte) error { _, err := rand.Read(b); return err }

func TestNavegadorIdentidad(t *testing.T) {
	b, ctx := baseIntegracion(t)
	directorio, id, contrasena := directorioNavegador(t)
	limitador, _ := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 200, MaxClaves: 1000})
	servicio, err := aplicacion.NuevoServicio(directorio, NuevosUsuarios(b), NuevasSesiones(b), relojNavegador{}, azarNavegador{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour}, limitador, b)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	disco := ficheros.NuevosPaquetes(filepath.Join(root, "idiomas"), filepath.Join(root, "temas"))
	servidor := httptest.NewUnstartedServer(nil)
	servidor.Config.ErrorLog = log.New(io.Discard, "", 0)
	servidor.StartTLS()
	defer servidor.Close()
	clave := make([]byte, 32)
	_, _ = rand.Read(clave)
	servidor.Config.Handler = adaptadorHTTP.Nuevo(paquete.NuevaConsulta(disco, disco), os.DirFS(filepath.Join(root, "web/dist")), "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), b.Comprobar, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: servicio, Origen: servidor.URL, ClaveCSRF: clave}})
	// Playwright abre exclusivamente Chrome instalado; no descarga navegador.
	limite, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
	defer cancelar()
	comando := exec.CommandContext(limite, "node", filepath.Join(root, "web/tests/identidad.navegador.mjs"))
	evidencia := os.Getenv("ETF_EVIDENCIA_RUTA")
	if evidencia == "" {
		evidencia = t.TempDir()
	}
	comando.Env = append(os.Environ(), "ETF_PRUEBA_URL="+servidor.URL, "ETF_PRUEBA_CLAVE="+contrasena, "ETF_PRUEBA_LDAP="+id, "ETF_EVIDENCIA_RUTA="+evidencia)
	salida, err := comando.CombinedOutput()
	t.Log(string(salida))
	if err != nil {
		t.Fatal(err)
	}
}

// Directorio LDAPS aislado: misma imagen y esquema que las pruebas F0.09a.
func directorioNavegador(t *testing.T) (*ldap.Directorio, string, string) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	docker := func(args ...string) string {
		t.Helper()
		salida, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker: %v %s", err, salida)
		}
		return strings.TrimSpace(string(salida))
	}
	var azar [24]byte
	_, _ = rand.Read(azar[:])
	nombre := fmt.Sprintf("etf-f009b-ldap-%x", azar[:6])
	ip := fmt.Sprintf("10.252.%d.2", azar[0])
	red := nombre + "-red"
	docker("network", "create", "--internal", "--subnet", fmt.Sprintf("10.252.%d.0/24", azar[0]), red)
	t.Cleanup(func() { _ = exec.Command("docker", "network", "rm", red).Run() })
	ruta := t.TempDir()
	if err := os.Chmod(ruta, 0755); err != nil {
		t.Fatal(err)
	}
	privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "directorio.etf.interno"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP(ip)}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &privada.PublicKey, privada)
	if err != nil {
		t.Fatal(err)
	}
	certificado := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	contrasena := base64.RawURLEncoding.EncodeToString(azar[:])
	privadaDER, err := x509.MarshalPKCS8PrivateKey(privada)
	if err != nil {
		t.Fatal(err)
	}
	archivos := map[string][]byte{"servidor.pem": certificado, "servidor.key": pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privadaDER})}
	for _, nombre := range []string{"slapd.conf", "identidad.schema"} {
		contenido, err := os.ReadFile(filepath.Join("../ldap/testdata", nombre))
		if err != nil {
			t.Fatal(err)
		}
		archivos[nombre] = contenido
	}
	archivos["datos.ldif"] = []byte(fmt.Sprintf(`dn: dc=etf,dc=interno
objectClass: top
objectClass: domain
dc: etf

dn: ou=personas,dc=etf,dc=interno
objectClass: organizationalUnit
ou: personas

dn: uid=ines.moreno,ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
objectClass: identidadETF
uid: ines.moreno
cn: Inés Moreno Castillo
sn: Moreno Castillo
displayName: Inés Moreno Castillo
userPassword: %s
objectGUID:: MyIRAFVEd2aImaq7zN3u/w==

dn: uid=marta.lopez,ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
objectClass: identidadETF
uid: marta.lopez
cn: Marta López Rivas
sn: López Rivas
displayName: Marta López Rivas
userPassword: %s
objectGUID:: MyIRAFVEd2aImaq7zN3u/g==
`, contrasena, contrasena))
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(ruta, nombre), contenido, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(ruta, nombre), 0644); err != nil {
			t.Fatal(err)
		}
	}
	id := docker("run", "--detach", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--memory=128m", "--cpus=1", "--pids-limit=32", "--network", red, "--ip", ip, "--tmpfs", "/tmp:rw,nosuid,nodev,size=32m,uid=65534,gid=65534", "--mount", "type=bind,src="+ruta+",dst=/fixture,readonly", "etf-ldap-pruebas:f0.09a")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", id).Run() })
	ca := x509.NewCertPool()
	ca.AppendCertsFromPEM(certificado)
	directorio, err := ldap.Nuevo(ldap.Configuracion{URL: "ldaps://" + net.JoinHostPort(ip, "1636"), BaseDN: "ou=personas,dc=etf,dc=interno", FormatoBind: "uid={usuario},ou=personas,dc=etf,dc=interno", BindDN: true, AtributoCuenta: "uid", Autoridades: ca, TiempoLimite: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for inicio := time.Now(); ; time.Sleep(100 * time.Millisecond) {
		if _, err := directorio.Autenticar(ctx, "ines.moreno", contrasena); err == nil {
			break
		}
		if time.Since(inicio) > 10*time.Second {
			t.Fatalf("LDAPS no disponible; estado=%s", docker("inspect", "--format", "{{.State.Status}} {{.State.ExitCode}}", id))
		}
	}
	return directorio, id, contrasena
}
